//go:build integration

package migrate_test

import (
	"context"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tc_pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/tidylogic/db-diff/internal/config"
	"github.com/tidylogic/db-diff/internal/connector/postgres"
	"github.com/tidylogic/db-diff/internal/diff"
	"github.com/tidylogic/db-diff/internal/migrate"
	"github.com/tidylogic/db-diff/internal/schema"
)

// pgSrcStatements defines the source schema.
// users: age INT NULL, no bio, has created_at.
// orders unchanged between src and tgt.
// View references created_at.
var pgSrcStatements = []string{
	`CREATE TABLE users (
		id         SERIAL       NOT NULL,
		username   VARCHAR(100) NOT NULL,
		email      VARCHAR(255) NOT NULL,
		age        INTEGER,
		created_at TIMESTAMP    NOT NULL,
		PRIMARY KEY (id),
		CONSTRAINT uq_email UNIQUE (email)
	)`,
	`CREATE TABLE orders (
		id      SERIAL        NOT NULL,
		user_id INTEGER       NOT NULL,
		amount  NUMERIC(10,2) NOT NULL,
		PRIMARY KEY (id),
		CONSTRAINT fk_orders_users FOREIGN KEY (user_id) REFERENCES users(id)
	)`,
	`CREATE INDEX idx_orders_user_id ON orders(user_id)`,
	`CREATE VIEW user_orders AS
		SELECT u.username, u.created_at FROM users u`,
}

// pgTgtStatements defines the target schema:
// users.age changed to BIGINT, bio TEXT added, created_at dropped,
// new index idx_users_username, new unique constraint uq_users_bio.
// orders unchanged. View redefined.
var pgTgtStatements = []string{
	`CREATE TABLE users (
		id       SERIAL       NOT NULL,
		username VARCHAR(100) NOT NULL,
		email    VARCHAR(255) NOT NULL,
		age      BIGINT,
		bio      TEXT,
		PRIMARY KEY (id),
		CONSTRAINT uq_email    UNIQUE (email),
		CONSTRAINT uq_users_bio UNIQUE (bio)
	)`,
	`CREATE TABLE orders (
		id      SERIAL        NOT NULL,
		user_id INTEGER       NOT NULL,
		amount  NUMERIC(10,2) NOT NULL,
		PRIMARY KEY (id),
		CONSTRAINT fk_orders_users FOREIGN KEY (user_id) REFERENCES users(id)
	)`,
	`CREATE INDEX idx_orders_user_id   ON orders(user_id)`,
	`CREATE INDEX idx_users_username ON users(username)`,
	`CREATE VIEW user_orders AS
		SELECT u.username, u.email FROM users u`,
}

func TestPostgresMigrate(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	versions := []struct {
		name  string
		image string
	}{
		{name: "PostgreSQL_13", image: "postgres:13"},
		{name: "PostgreSQL_14", image: "postgres:14"},
		{name: "PostgreSQL_15", image: "postgres:15"},
		{name: "PostgreSQL_16", image: "postgres:16"},
		{name: "PostgreSQL_17", image: "postgres:17"},
	}

	for _, v := range versions {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			waitStrategy := testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2),
			)

			// Start source container
			srcCtr, err := tc_pg.Run(ctx, v.image,
				tc_pg.WithDatabase("srcdb"),
				waitStrategy,
			)
			if srcCtr != nil {
				t.Cleanup(func() { _ = srcCtr.Terminate(ctx) })
			}
			if err != nil {
				t.Fatalf("start src container %s: %v", v.image, err)
			}

			// Start target container
			tgtCtr, err := tc_pg.Run(ctx, v.image,
				tc_pg.WithDatabase("tgtdb"),
				waitStrategy,
			)
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
			applySQL(t, "pgx", srcDSN, pgSrcStatements)
			applySQL(t, "pgx", tgtDSN, pgTgtStatements)

			// Extract schemas
			srcSchema := extractPostgres(t, srcDSN, "public")
			tgtSchema := extractPostgres(t, tgtDSN, "public")

			// Diff and generate migration SQL
			result := diff.Compare(srcSchema, tgtSchema, config.IgnoreConfig{})
			if result.Identical {
				t.Fatal("expected schemas to differ, got identical")
			}

			sqlOut, err := migrate.Generate(result, "apply_to_source", "postgres")
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			// Apply migration to source container
			execSQL(t, "pgx", srcDSN, sqlOut)

			// Re-extract source schema and validate
			updated := extractPostgres(t, srcDSN, "public")
			assertPostgresSchemasMatch(t, updated, tgtSchema)
		})
	}
}

// extractPostgres extracts the schema from a PostgreSQL DSN using our connector.
func extractPostgres(t *testing.T, dsn, schemaName string) *schema.Schema {
	t.Helper()
	c := postgres.New()
	if err := c.Connect(dsn); err != nil {
		t.Fatalf("postgres connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	s, err := c.ExtractSchema(schemaName)
	if err != nil {
		t.Fatalf("ExtractSchema(%s): %v", schemaName, err)
	}
	return s
}

// assertPostgresSchemasMatch validates the migrated source matches the target.
func assertPostgresSchemasMatch(t *testing.T, got, want *schema.Schema) {
	t.Helper()

	users, ok := got.Tables["users"]
	if !ok {
		t.Fatal("table 'users' not found after migration")
	}

	// users.age should be bigint
	ageCol := findPGCol(users.Columns, "age")
	if ageCol == nil {
		t.Error("column 'age' not found in migrated users table")
	} else if !strings.Contains(strings.ToLower(ageCol.RawType), "bigint") &&
		!strings.Contains(strings.ToLower(ageCol.DataType), "bigint") {
		t.Errorf("users.age: expected bigint type, got RawType=%q DataType=%q", ageCol.RawType, ageCol.DataType)
	}

	// users.bio should exist
	if findPGCol(users.Columns, "bio") == nil {
		t.Error("column 'bio' not found in migrated users table")
	}

	// users.created_at should be gone
	if findPGCol(users.Columns, "created_at") != nil {
		t.Error("column 'created_at' should have been dropped from users")
	}

	// idx_users_username index should exist
	if !hasIndex(users, "idx_users_username") {
		t.Error("index 'idx_users_username' not found in migrated users table")
	}

	// unique constraint on users.bio should exist
	if !hasUniqueOnCol(users, "bio") {
		t.Error("expected unique index or constraint on users.bio after migration")
	}

	// view user_orders should be redefined (PostgreSQL normalizes view defs,
	// so we assert absence of the old column reference rather than exact match)
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

func findPGCol(cols []schema.Column, name string) *schema.Column {
	for i := range cols {
		if cols[i].Name == name {
			return &cols[i]
		}
	}
	return nil
}
