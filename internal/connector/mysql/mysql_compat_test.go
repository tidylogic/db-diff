package mysql_test

import (
	"context"
	"database/sql"
	"testing"

	tc_mysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"db-diff/internal/connector/mysql"
	"db-diff/internal/schema"
)

// mysqlSetupStatements creates a representative test schema:
// two tables (with PK, FK, unique index) and a view.
var mysqlSetupStatements = []string{
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
		SELECT u.username, o.amount FROM users u JOIN orders o ON u.id = o.user_id`,
}

func TestCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		image string
	}{
		{name: "MySQL 5.7", image: "mysql:5.7"},
		{name: "MySQL 8.0", image: "mysql:8.0"},
		{name: "MySQL 8.4", image: "mysql:8.4"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			ctr, err := tc_mysql.Run(ctx, tt.image,
				tc_mysql.WithDatabase("testdb"),
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
			// driver is registered when the mysql package is imported above).
			setupDB, err := sql.Open("mysql", dsn)
			if err != nil {
				t.Fatalf("open setup connection: %v", err)
			}
			for _, stmt := range mysqlSetupStatements {
				if _, err := setupDB.ExecContext(ctx, stmt); err != nil {
					setupDB.Close()
					t.Fatalf("execute setup statement: %v", err)
				}
			}
			setupDB.Close()

			// Use our connector to extract the schema.
			c := mysql.New()
			if err := c.Connect(dsn); err != nil {
				t.Fatalf("connect: %v", err)
			}
			t.Cleanup(func() { _ = c.Close() })

			s, err := c.ExtractSchema("testdb")
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
