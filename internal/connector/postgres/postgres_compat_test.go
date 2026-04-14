package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tc_pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"db-diff/internal/connector/postgres"
	"db-diff/internal/schema"
)

// pgSetupStatements creates a representative test schema:
// two tables (with PK, FK, unique constraint) and a view.
var pgSetupStatements = []string{
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
		SELECT u.username, o.amount FROM users u JOIN orders o ON u.id = o.user_id`,
}

func TestCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		image string
	}{
		{name: "PostgreSQL 13", image: "postgres:13"},
		{name: "PostgreSQL 14", image: "postgres:14"},
		{name: "PostgreSQL 15", image: "postgres:15"},
		{name: "PostgreSQL 16", image: "postgres:16"},
		{name: "PostgreSQL 17", image: "postgres:17"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			ctr, err := tc_pg.Run(ctx, tt.image,
				tc_pg.WithDatabase("testdb"),
				testcontainers.WithWaitStrategy(
					wait.ForLog("database system is ready to accept connections").
						WithOccurrence(2),
				),
			)
			if ctr != nil {
				t.Cleanup(func() { _ = ctr.Terminate(ctx) })
			}
			if err != nil {
				t.Fatalf("start container %s: %v", tt.image, err)
			}

			dsn, err := ctr.ConnectionString(ctx)
			if err != nil {
				t.Fatalf("get connection string: %v", err)
			}

			// Set up the test schema via a raw sql.DB (our connector's
			// driver is registered when the postgres package is imported above).
			setupDB, err := sql.Open("pgx", dsn)
			if err != nil {
				t.Fatalf("open setup connection: %v", err)
			}
			for _, stmt := range pgSetupStatements {
				if _, err := setupDB.ExecContext(ctx, stmt); err != nil {
					setupDB.Close()
					t.Fatalf("execute setup statement: %v", err)
				}
			}
			setupDB.Close()

			// Use our connector to extract the schema.
			// PostgreSQL ExtractSchema takes the schema name ("public"), not the DB name.
			c := postgres.New()
			if err := c.Connect(dsn); err != nil {
				t.Fatalf("connect: %v", err)
			}
			t.Cleanup(func() { _ = c.Close() })

			s, err := c.ExtractSchema("public")
			if err != nil {
				t.Fatalf("ExtractSchema: %v", err)
			}

			assertSchema(t, s)
		})
	}
}

// assertSchema validates that the extracted schema contains the expected
// tables, columns, indexes, constraints, and views.
func assertSchema(t *testing.T, s *schema.Schema) {
	t.Helper()

	if len(s.Tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(s.Tables))
	}

	users, ok := s.Tables["users"]
	if !ok {
		t.Fatal("table 'users' not found")
	}
	if len(users.Columns) != 5 {
		t.Errorf("users: expected 5 columns, got %d", len(users.Columns))
	}
	if col := findColumn(users.Columns, "id"); col == nil {
		t.Error("users: column 'id' not found")
	} else if col.Nullable {
		t.Error("users.id: expected NOT NULL")
	}
	if col := findColumn(users.Columns, "email"); col == nil {
		t.Error("users: column 'email' not found")
	} else if col.Nullable {
		t.Error("users.email: expected NOT NULL")
	}
	if col := findColumn(users.Columns, "age"); col == nil {
		t.Error("users: column 'age' not found")
	} else if !col.Nullable {
		t.Error("users.age: expected nullable")
	}
	if !hasUniqueOn(users, "email") {
		t.Error("users: expected unique index or constraint on 'email'")
	}

	orders, ok := s.Tables["orders"]
	if !ok {
		t.Fatal("table 'orders' not found")
	}
	if !hasFKTo(orders, "users") {
		t.Error("orders: expected FOREIGN KEY constraint referencing 'users'")
	}

	if len(s.Views) != 1 {
		t.Errorf("expected 1 view, got %d", len(s.Views))
	}
	if _, ok := s.Views["user_orders"]; !ok {
		t.Error("view 'user_orders' not found")
	}
}

// findColumn returns the column with the given name, or nil if not found.
func findColumn(cols []schema.Column, name string) *schema.Column {
	for i := range cols {
		if cols[i].Name == name {
			return &cols[i]
		}
	}
	return nil
}

// hasUniqueOn reports whether tbl has a unique index or constraint that
// covers exactly the single column col.
func hasUniqueOn(tbl schema.Table, col string) bool {
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

// hasFKTo reports whether tbl has a FOREIGN KEY constraint pointing at refTable.
func hasFKTo(tbl schema.Table, refTable string) bool {
	for _, c := range tbl.Constraints {
		if c.Type == "FOREIGN KEY" && c.RefTable == refTable {
			return true
		}
	}
	return false
}
