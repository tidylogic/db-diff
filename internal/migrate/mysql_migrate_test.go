//go:build integration

package migrate_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tc_mysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/tidylogic/db-diff/internal/config"
	"github.com/tidylogic/db-diff/internal/connector/mysql"
	"github.com/tidylogic/db-diff/internal/diff"
	"github.com/tidylogic/db-diff/internal/migrate"
	"github.com/tidylogic/db-diff/internal/schema"
)

// mysqlSrcStatements defines the source schema:
// users table with age INT NULL, no bio, has created_at.
// orders table unchanged between src and tgt.
// A view defined with "old" columns.
var mysqlSrcStatements = []string{
	`CREATE TABLE users (
		id         INT          NOT NULL AUTO_INCREMENT,
		username   VARCHAR(100) NOT NULL,
		email      VARCHAR(255) NOT NULL,
		age        INT,
		created_at DATETIME     NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uq_email (email)
	)`,
	`CREATE TABLE orders (
		id      INT           NOT NULL AUTO_INCREMENT,
		user_id INT           NOT NULL,
		amount  DECIMAL(10,2) NOT NULL,
		PRIMARY KEY (id),
		INDEX idx_user_id (user_id),
		CONSTRAINT fk_orders_users FOREIGN KEY (user_id) REFERENCES users(id)
	)`,
	`CREATE VIEW user_orders AS
		SELECT u.username, u.created_at FROM users u`,
}

// mysqlTgtStatements defines the target schema:
// users.age changed to BIGINT, bio added, created_at dropped,
// new index idx_users_username, new unique constraint uq_users_bio.
// orders unchanged.
// View redefined without created_at.
var mysqlTgtStatements = []string{
	`CREATE TABLE users (
		id       INT          NOT NULL AUTO_INCREMENT,
		username VARCHAR(100) NOT NULL,
		email    VARCHAR(255) NOT NULL,
		age      BIGINT,
		bio      TEXT,
		PRIMARY KEY (id),
		UNIQUE KEY uq_email (email),
		UNIQUE KEY uq_users_bio (bio),
		INDEX idx_users_username (username)
	)`,
	`CREATE TABLE orders (
		id      INT           NOT NULL AUTO_INCREMENT,
		user_id INT           NOT NULL,
		amount  DECIMAL(10,2) NOT NULL,
		PRIMARY KEY (id),
		INDEX idx_user_id (user_id),
		CONSTRAINT fk_orders_users FOREIGN KEY (user_id) REFERENCES users(id)
	)`,
	`CREATE VIEW user_orders AS
		SELECT u.username, u.email FROM users u`,
}

func TestMySQLMigrate(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	versions := []struct {
		name  string
		image string
	}{
		{name: "MySQL_5.7", image: "mysql:5.7"},
		{name: "MySQL_8.0", image: "mysql:8.0"},
		{name: "MySQL_8.4", image: "mysql:8.4"},
	}

	for _, v := range versions {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// Start source container
			srcCtr, err := tc_mysql.Run(ctx, v.image, tc_mysql.WithDatabase("srcdb"))
			if srcCtr != nil {
				t.Cleanup(func() { _ = srcCtr.Terminate(ctx) })
			}
			if err != nil {
				t.Fatalf("start src container %s: %v", v.image, err)
			}

			// Start target container
			tgtCtr, err := tc_mysql.Run(ctx, v.image, tc_mysql.WithDatabase("tgtdb"))
			if tgtCtr != nil {
				t.Cleanup(func() { _ = tgtCtr.Terminate(ctx) })
			}
			if err != nil {
				t.Fatalf("start tgt container %s: %v", v.image, err)
			}

			srcDSN, err := srcCtr.ConnectionString(ctx)
			if err != nil {
				t.Fatalf("src connection string: %v", err)
			}
			tgtDSN, err := tgtCtr.ConnectionString(ctx)
			if err != nil {
				t.Fatalf("tgt connection string: %v", err)
			}

			// Apply DDL to each container
			applySQL(t, "mysql", srcDSN, mysqlSrcStatements)
			applySQL(t, "mysql", tgtDSN, mysqlTgtStatements)

			// Extract schemas
			srcSchema := extractMySQL(t, srcDSN, "srcdb")
			tgtSchema := extractMySQL(t, tgtDSN, "tgtdb")

			// Diff and generate migration SQL
			result := diff.Compare(srcSchema, tgtSchema, config.IgnoreConfig{})
			if result.Identical {
				t.Fatal("expected schemas to differ, got identical")
			}

			sqlOut, err := migrate.Generate(result, "apply_to_source", "mysql")
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			// Apply migration to source container
			execSQL(t, "mysql", srcDSN, sqlOut)

			// Re-extract source schema and validate
			updated := extractMySQL(t, srcDSN, "srcdb")
			assertMySQLSchemasMatch(t, updated, tgtSchema)
		})
	}
}

// extractMySQL extracts the schema from a MySQL DSN using our connector.
func extractMySQL(t *testing.T, dsn, dbName string) *schema.Schema {
	t.Helper()
	c := mysql.New()
	if err := c.Connect(dsn); err != nil {
		t.Fatalf("mysql connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	s, err := c.ExtractSchema(dbName)
	if err != nil {
		t.Fatalf("ExtractSchema(%s): %v", dbName, err)
	}
	return s
}

// applySQL opens a raw DB connection and executes each statement.
func applySQL(t *testing.T, driver, dsn string, stmts []string) {
	t.Helper()
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("open db (%s): %v", driver, err)
	}
	defer db.Close()
	ctx := context.Background()
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
}

// execSQL splits generated SQL on ";\n" and executes each real statement.
// Comment lines (--) and blank lines are stripped from each chunk before
// execution, so section headers prepended to the first statement of a block
// do not cause the statement to be skipped.
func execSQL(t *testing.T, driver, dsn, sqlText string) {
	t.Helper()
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("open db (%s): %v", driver, err)
	}
	defer db.Close()
	ctx := context.Background()

	for _, chunk := range strings.Split(sqlText, ";\n") {
		// Strip comment and blank lines; preserve actual SQL lines.
		var sqlLines []string
		for _, line := range strings.Split(chunk, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
				sqlLines = append(sqlLines, trimmed)
			}
		}
		stmt := strings.Join(sqlLines, "\n")
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec migration stmt %q: %v", stmt, err)
		}
	}
}

// assertMySQLSchemasMatch validates that the migrated source matches the target.
func assertMySQLSchemasMatch(t *testing.T, got, want *schema.Schema) {
	t.Helper()

	// users.age should be bigint (MySQL 5.7 may return UPPERCASE)
	users, ok := got.Tables["users"]
	if !ok {
		t.Fatal("table 'users' not found after migration")
	}
	ageCol := findMySQLCol(users.Columns, "age")
	if ageCol == nil {
		t.Error("column 'age' not found in migrated users table")
	} else if !strings.Contains(strings.ToLower(ageCol.RawType), "bigint") {
		t.Errorf("users.age: expected bigint type, got %q", ageCol.RawType)
	}

	// users.bio should exist
	if findMySQLCol(users.Columns, "bio") == nil {
		t.Error("column 'bio' not found in migrated users table")
	}

	// users.created_at should be gone
	if findMySQLCol(users.Columns, "created_at") != nil {
		t.Error("column 'created_at' should have been dropped from users")
	}

	// idx_users_username index should exist
	if !hasIndex(users, "idx_users_username") {
		t.Error("index 'idx_users_username' not found in migrated users table")
	}

	// unique constraint/index on users.bio should exist
	if !hasUniqueOnCol(users, "bio") {
		t.Error("expected unique index or constraint on users.bio after migration")
	}

	// view user_orders should be redefined (no 'created_at' reference)
	v, ok := got.Views["user_orders"]
	if !ok {
		t.Error("view 'user_orders' not found after migration")
	} else if strings.Contains(strings.ToLower(v.Definition), "created_at") {
		t.Errorf("view 'user_orders' still references 'created_at': %s", v.Definition)
	}

	// orders table should be unchanged
	gotOrders, ok := got.Tables["orders"]
	if !ok {
		t.Fatal("table 'orders' missing after migration")
	}
	wantOrders, ok := want.Tables["orders"]
	if !ok {
		t.Fatal("table 'orders' not in target schema")
	}
	if len(gotOrders.Columns) != len(wantOrders.Columns) {
		t.Errorf("orders: expected %d columns, got %d", len(wantOrders.Columns), len(gotOrders.Columns))
	}
}

func findMySQLCol(cols []schema.Column, name string) *schema.Column {
	for i := range cols {
		if cols[i].Name == name {
			return &cols[i]
		}
	}
	return nil
}

func hasIndex(tbl schema.Table, name string) bool {
	for _, idx := range tbl.Indexes {
		if idx.Name == name {
			return true
		}
	}
	return false
}

func hasUniqueOnCol(tbl schema.Table, col string) bool {
	for _, idx := range tbl.Indexes {
		if idx.Unique && len(idx.Columns) == 1 && idx.Columns[0] == col {
			return true
		}
	}
	for _, c := range tbl.Constraints {
		if c.Type == "UNIQUE" && len(c.Columns) == 1 && c.Columns[0] == col {
			return true
		}
	}
	return false
}
