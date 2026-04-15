package postgres

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/tidylogic/db-diff/internal/schema"
)

// Connector implements the connector.Connector interface for PostgreSQL.
type Connector struct {
	db *sql.DB
}

// New creates a new PostgreSQL Connector.
func New() *Connector {
	return &Connector{}
}

// Connect opens and pings the PostgreSQL database.
// DSN format: "postgres://user:pass@host:port/dbname" or "postgresql://...".
func (c *Connector) Connect(dsn string) error {
	// pgx/stdlib accepts "postgres://" and "postgresql://" natively.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("postgres: opening connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("postgres: pinging database: %w", err)
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

// ExtractSchema queries information_schema and pg_catalog to build a complete Schema.
func (c *Connector) ExtractSchema(schemaName string) (*schema.Schema, error) {
	s := &schema.Schema{
		Name:   schemaName,
		Tables: make(map[string]schema.Table),
		Views:  make(map[string]schema.View),
	}

	if err := c.extractTables(s, schemaName); err != nil {
		return nil, err
	}
	if err := c.extractColumns(s, schemaName); err != nil {
		return nil, err
	}
	if err := c.extractIndexes(s, schemaName); err != nil {
		return nil, err
	}
	if err := c.extractConstraints(s, schemaName); err != nil {
		return nil, err
	}
	if err := c.extractViews(s, schemaName); err != nil {
		return nil, err
	}

	return s, nil
}

func (c *Connector) extractTables(s *schema.Schema, schemaName string) error {
	rows, err := c.db.Query(`
		SELECT t.table_name, COALESCE(obj_description(pc.oid), '')
		FROM information_schema.tables t
		LEFT JOIN pg_catalog.pg_class pc
			ON pc.relname = t.table_name
		LEFT JOIN pg_catalog.pg_namespace pn
			ON pn.oid = pc.relnamespace AND pn.nspname = t.table_schema
		WHERE t.table_schema = $1 AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name`, schemaName)
	if err != nil {
		return fmt.Errorf("postgres: querying tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, comment string
		if err := rows.Scan(&name, &comment); err != nil {
			return fmt.Errorf("postgres: scanning table row: %w", err)
		}
		s.Tables[name] = schema.Table{Name: name, Comment: comment}
	}
	return rows.Err()
}

func (c *Connector) extractColumns(s *schema.Schema, schemaName string) error {
	rows, err := c.db.Query(`
		SELECT
			c.table_name,
			c.column_name,
			c.ordinal_position,
			c.column_default,
			c.is_nullable,
			c.data_type,
			c.udt_name,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			COALESCE(col_description(pc.oid, c.ordinal_position::int), '')
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_class pc
			ON pc.relname = c.table_name
		LEFT JOIN pg_catalog.pg_namespace pn
			ON pn.oid = pc.relnamespace AND pn.nspname = c.table_schema
		WHERE c.table_schema = $1
		ORDER BY c.table_name, c.ordinal_position`, schemaName)
	if err != nil {
		return fmt.Errorf("postgres: querying columns: %w", err)
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
			udtName    string
			charMaxLen sql.NullInt64
			numPrec    sql.NullInt64
			numScale   sql.NullInt64
			comment    string
		)
		if err := rows.Scan(&tableName, &colName, &ordinal, &defVal, &isNullable,
			&dataType, &udtName, &charMaxLen, &numPrec, &numScale, &comment); err != nil {
			return fmt.Errorf("postgres: scanning column row: %w", err)
		}

		col := schema.Column{
			Name:       colName,
			OrdinalPos: ordinal,
			DataType:   dataType,
			RawType:    udtName,
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

func (c *Connector) extractIndexes(s *schema.Schema, schemaName string) error {
	rows, err := c.db.Query(`
		SELECT
			t.relname AS table_name,
			i.relname AS index_name,
			ix.indisunique,
			ix.indisprimary,
			array_to_string(
				ARRAY(
					SELECT a.attname
					FROM pg_catalog.pg_attribute a
					JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n)
						ON a.attnum = k.attnum AND a.attrelid = t.oid
					ORDER BY k.n
				),
				','
			) AS columns
		FROM pg_catalog.pg_index ix
		JOIN pg_catalog.pg_class t ON t.oid = ix.indrelid
		JOIN pg_catalog.pg_class i ON i.oid = ix.indexrelid
		JOIN pg_catalog.pg_namespace ns ON ns.oid = t.relnamespace
		WHERE ns.nspname = $1
		ORDER BY t.relname, i.relname`, schemaName)
	if err != nil {
		return fmt.Errorf("postgres: querying indexes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, indexName, colList string
		var isUnique, isPrimary bool
		if err := rows.Scan(&tableName, &indexName, &isUnique, &isPrimary, &colList); err != nil {
			return fmt.Errorf("postgres: scanning index row: %w", err)
		}

		t, ok := s.Tables[tableName]
		if !ok {
			continue
		}

		cols := []string{}
		for _, c := range strings.Split(colList, ",") {
			if c = strings.TrimSpace(c); c != "" {
				cols = append(cols, c)
			}
		}

		t.Indexes = append(t.Indexes, schema.Index{
			Name:      indexName,
			Columns:   cols,
			Unique:    isUnique,
			IsPrimary: isPrimary,
		})
		s.Tables[tableName] = t
	}
	return rows.Err()
}

func (c *Connector) extractConstraints(s *schema.Schema, schemaName string) error {
	rows, err := c.db.Query(`
		SELECT
			tc.table_name,
			tc.constraint_name,
			tc.constraint_type,
			kcu.column_name,
			kcu.ordinal_position,
			COALESCE(ccu.table_name, ''),
			COALESCE(ccu.column_name, '')
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
			AND kcu.table_name = tc.table_name
		LEFT JOIN information_schema.referential_constraints rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		LEFT JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.table_schema = $1
			AND tc.constraint_type IN ('PRIMARY KEY', 'FOREIGN KEY', 'UNIQUE')
		ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position`, schemaName)
	if err != nil {
		return fmt.Errorf("postgres: querying constraints: %w", err)
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
			return fmt.Errorf("postgres: scanning constraint row: %w", err)
		}
		k := cKey{tableName, constraintName}
		if _, ok := entries[k]; !ok {
			entries[k] = &cEntry{cType: constraintType, refTable: refTable}
			order = append(order, k)
		}
		e := entries[k]
		e.columns = appendUnique(e.columns, colName)
		if refCol != "" {
			e.refColumns = appendUnique(e.refColumns, refCol)
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

func (c *Connector) extractViews(s *schema.Schema, schemaName string) error {
	rows, err := c.db.Query(`
		SELECT table_name, view_definition
		FROM information_schema.views
		WHERE table_schema = $1
		ORDER BY table_name`, schemaName)
	if err != nil {
		return fmt.Errorf("postgres: querying views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var def sql.NullString
		if err := rows.Scan(&name, &def); err != nil {
			return fmt.Errorf("postgres: scanning view row: %w", err)
		}
		s.Views[name] = schema.View{Name: name, Definition: def.String}
	}
	return rows.Err()
}

// sortTableSlices sorts index/constraint slices for deterministic diffs.
func sortTableSlices(t *schema.Table) {
	sort.Slice(t.Indexes, func(i, j int) bool {
		return t.Indexes[i].Name < t.Indexes[j].Name
	})
	sort.Slice(t.Constraints, func(i, j int) bool {
		return t.Constraints[i].Name < t.Constraints[j].Name
	})
}

// appendUnique appends s to slice only if not already present.
func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
