package mysql

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"db-diff/internal/schema"
)

// Connector implements the connector.Connector interface for MySQL.
type Connector struct {
	db *sql.DB
}

// New creates a new MySQL Connector.
func New() *Connector {
	return &Connector{}
}

// Connect opens and pings the MySQL database using the provided DSN.
// DSN format: "mysql://user:pass@host:port/dbname" or standard MySQL DSN.
func (c *Connector) Connect(dsn string) error {
	// Strip the "mysql://" scheme prefix if present; go-sql-driver uses its own format.
	normalized := strings.TrimPrefix(dsn, "mysql://")

	db, err := sql.Open("mysql", normalized)
	if err != nil {
		return fmt.Errorf("mysql: opening connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("mysql: pinging database: %w", err)
	}
	c.db = db
	return nil
}

// Close releases the underlying database connection.
func (c *Connector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// ExtractSchema queries information_schema to build a complete Schema for dbName.
func (c *Connector) ExtractSchema(dbName string) (*schema.Schema, error) {
	s := &schema.Schema{
		Name:   dbName,
		Tables: make(map[string]schema.Table),
		Views:  make(map[string]schema.View),
	}

	if err := c.extractTables(s, dbName); err != nil {
		return nil, err
	}
	if err := c.extractColumns(s, dbName); err != nil {
		return nil, err
	}
	if err := c.extractIndexes(s, dbName); err != nil {
		return nil, err
	}
	if err := c.extractConstraints(s, dbName); err != nil {
		return nil, err
	}
	if err := c.extractViews(s, dbName); err != nil {
		return nil, err
	}

	return s, nil
}

func (c *Connector) extractTables(s *schema.Schema, dbName string) error {
	rows, err := c.db.Query(`
		SELECT TABLE_NAME, IFNULL(TABLE_COMMENT, '')
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`, dbName)
	if err != nil {
		return fmt.Errorf("mysql: querying tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, comment string
		if err := rows.Scan(&name, &comment); err != nil {
			return fmt.Errorf("mysql: scanning table row: %w", err)
		}
		s.Tables[name] = schema.Table{
			Name:    name,
			Comment: comment,
		}
	}
	return rows.Err()
}

func (c *Connector) extractColumns(s *schema.Schema, dbName string) error {
	rows, err := c.db.Query(`
		SELECT
			TABLE_NAME,
			COLUMN_NAME,
			ORDINAL_POSITION,
			COLUMN_DEFAULT,
			IS_NULLABLE,
			DATA_TYPE,
			COLUMN_TYPE,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			IFNULL(COLUMN_COMMENT, '')
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`, dbName)
	if err != nil {
		return fmt.Errorf("mysql: querying columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			tableName  string
			colName    string
			ordinal    int
			defVal     sql.NullString
			isNullable string
			dataType   string
			rawType    string
			charMaxLen sql.NullInt64
			numPrec    sql.NullInt64
			numScale   sql.NullInt64
			comment    string
		)
		if err := rows.Scan(&tableName, &colName, &ordinal, &defVal, &isNullable,
			&dataType, &rawType, &charMaxLen, &numPrec, &numScale, &comment); err != nil {
			return fmt.Errorf("mysql: scanning column row: %w", err)
		}

		col := schema.Column{
			Name:       colName,
			OrdinalPos: ordinal,
			DataType:   dataType,
			RawType:    rawType,
			Nullable:   strings.EqualFold(isNullable, "YES"),
			Comment:    comment,
		}
		if defVal.Valid {
			col.Default = &defVal.String
		}
		if charMaxLen.Valid {
			col.CharMaxLen = &charMaxLen.Int64
		}
		if numPrec.Valid {
			col.NumPrecision = &numPrec.Int64
		}
		if numScale.Valid {
			col.NumScale = &numScale.Int64
		}

		if t, ok := s.Tables[tableName]; ok {
			t.Columns = append(t.Columns, col)
			s.Tables[tableName] = t
		}
	}
	return rows.Err()
}

func (c *Connector) extractIndexes(s *schema.Schema, dbName string) error {
	rows, err := c.db.Query(`
		SELECT
			TABLE_NAME,
			INDEX_NAME,
			NON_UNIQUE,
			SEQ_IN_INDEX,
			COLUMN_NAME
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`, dbName)
	if err != nil {
		return fmt.Errorf("mysql: querying indexes: %w", err)
	}
	defer rows.Close()

	// group rows by (table, index)
	type indexKey struct{ table, index string }
	type indexEntry struct {
		nonUnique bool
		columns   []string
	}
	order := []indexKey{}
	entries := map[indexKey]*indexEntry{}

	for rows.Next() {
		var tableName, indexName, colName string
		var nonUnique bool
		var seq int
		if err := rows.Scan(&tableName, &indexName, &nonUnique, &seq, &colName); err != nil {
			return fmt.Errorf("mysql: scanning index row: %w", err)
		}
		k := indexKey{tableName, indexName}
		if _, ok := entries[k]; !ok {
			entries[k] = &indexEntry{nonUnique: nonUnique}
			order = append(order, k)
		}
		entries[k].columns = append(entries[k].columns, colName)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, k := range order {
		e := entries[k]
		t, ok := s.Tables[k.table]
		if !ok {
			continue
		}
		t.Indexes = append(t.Indexes, schema.Index{
			Name:      k.index,
			Columns:   e.columns,
			Unique:    !e.nonUnique,
			IsPrimary: k.index == "PRIMARY",
		})
		s.Tables[k.table] = t
	}
	return nil
}

func (c *Connector) extractConstraints(s *schema.Schema, dbName string) error {
	rows, err := c.db.Query(`
		SELECT
			kcu.TABLE_NAME,
			kcu.CONSTRAINT_NAME,
			tc.CONSTRAINT_TYPE,
			kcu.COLUMN_NAME,
			kcu.ORDINAL_POSITION,
			IFNULL(kcu.REFERENCED_TABLE_NAME, ''),
			IFNULL(kcu.REFERENCED_COLUMN_NAME, '')
		FROM information_schema.KEY_COLUMN_USAGE kcu
		JOIN information_schema.TABLE_CONSTRAINTS tc
			ON tc.CONSTRAINT_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
			AND tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
		WHERE kcu.TABLE_SCHEMA = ?
		ORDER BY kcu.TABLE_NAME, kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`, dbName)
	if err != nil {
		return fmt.Errorf("mysql: querying constraints: %w", err)
	}
	defer rows.Close()

	type cKey struct{ table, constraint string }
	type cEntry struct {
		cType      string
		columns    []string
		refTable   string
		refColumns []string
	}
	order := []cKey{}
	entries := map[cKey]*cEntry{}

	for rows.Next() {
		var tableName, constraintName, constraintType, colName, refTable, refCol string
		var ordinal int
		if err := rows.Scan(&tableName, &constraintName, &constraintType, &colName, &ordinal, &refTable, &refCol); err != nil {
			return fmt.Errorf("mysql: scanning constraint row: %w", err)
		}
		k := cKey{tableName, constraintName}
		if _, ok := entries[k]; !ok {
			entries[k] = &cEntry{cType: constraintType, refTable: refTable}
			order = append(order, k)
		}
		e := entries[k]
		e.columns = append(e.columns, colName)
		if refCol != "" {
			e.refColumns = append(e.refColumns, refCol)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, k := range order {
		e := entries[k]
		t, ok := s.Tables[k.table]
		if !ok {
			continue
		}
		t.Constraints = append(t.Constraints, schema.Constraint{
			Name:       k.constraint,
			Type:       e.cType,
			Columns:    e.columns,
			RefTable:   e.refTable,
			RefColumns: e.refColumns,
		})
		s.Tables[k.table] = t
	}
	return nil
}

func (c *Connector) extractViews(s *schema.Schema, dbName string) error {
	rows, err := c.db.Query(`
		SELECT TABLE_NAME, VIEW_DEFINITION
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`, dbName)
	if err != nil {
		return fmt.Errorf("mysql: querying views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			return fmt.Errorf("mysql: scanning view row: %w", err)
		}
		s.Views[name] = schema.View{Name: name, Definition: def}
	}
	return rows.Err()
}

// sortTableSlices sorts column/index/constraint slices for deterministic diffs.
func sortTableSlices(t *schema.Table) {
	sort.Slice(t.Indexes, func(i, j int) bool {
		return t.Indexes[i].Name < t.Indexes[j].Name
	})
	sort.Slice(t.Constraints, func(i, j int) bool {
		return t.Constraints[i].Name < t.Constraints[j].Name
	})
}
