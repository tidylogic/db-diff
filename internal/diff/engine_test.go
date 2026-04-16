package diff

import (
	"testing"

	"github.com/tidylogic/db-diff/internal/config"
	"github.com/tidylogic/db-diff/internal/schema"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

func col(name, rawType string) schema.Column {
	return schema.Column{Name: name, DataType: rawType, RawType: rawType, Nullable: false}
}

func colNullable(name, rawType string) schema.Column {
	c := col(name, rawType)
	c.Nullable = true
	return c
}

func colWithDefault(name, rawType, def string) schema.Column {
	c := col(name, rawType)
	c.Default = strPtr(def)
	return c
}

func makeSchema(name string, tables []schema.Table, views []schema.View) *schema.Schema {
	s := &schema.Schema{
		Name:   name,
		Tables: make(map[string]schema.Table),
		Views:  make(map[string]schema.View),
	}
	for _, t := range tables {
		s.Tables[t.Name] = t
	}
	for _, v := range views {
		s.Views[v.Name] = v
	}
	return s
}

func noIgnore() config.IgnoreConfig { return config.IgnoreConfig{} }

// findTableDiff returns the TableDiff with the given name, or nil.
func findTableDiff(result *DiffResult, name string) *TableDiff {
	for i := range result.Tables {
		if result.Tables[i].Name == name {
			return &result.Tables[i]
		}
	}
	return nil
}

// findColumnDiff returns the ColumnDiff with the given name, or nil.
func findColumnDiff(td *TableDiff, name string) *ColumnDiff {
	for i := range td.Columns {
		if td.Columns[i].Name == name {
			return &td.Columns[i]
		}
	}
	return nil
}

// findIndexDiff returns the IndexDiff with the given name, or nil.
func findIndexDiff(td *TableDiff, name string) *IndexDiff {
	for i := range td.Indexes {
		if td.Indexes[i].Name == name {
			return &td.Indexes[i]
		}
	}
	return nil
}

// findConstraintDiff returns the ConstraintDiff with the given name, or nil.
func findConstraintDiff(td *TableDiff, name string) *ConstraintDiff {
	for i := range td.Constraints {
		if td.Constraints[i].Name == name {
			return &td.Constraints[i]
		}
	}
	return nil
}

// ── TestCompare ───────────────────────────────────────────────────────────────

func TestCompare(t *testing.T) {
	t.Parallel()

	// Shared base table used across many cases.
	usersTable := schema.Table{
		Name:    "users",
		Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)")},
	}

	tests := []struct {
		name   string
		src    *schema.Schema
		tgt    *schema.Schema
		ignore config.IgnoreConfig
		check  func(t *testing.T, result *DiffResult)
	}{
		// ── Basic identical / empty ──────────────────────────────────────────

		{
			name: "identical_schemas",
			src:  makeSchema("src", []schema.Table{usersTable}, nil),
			tgt:  makeSchema("tgt", []schema.Table{usersTable}, nil),
			check: func(t *testing.T, r *DiffResult) {
				if !r.Identical {
					t.Error("expected Identical == true")
				}
				if len(r.Tables) != 0 {
					t.Errorf("expected 0 table diffs, got %d", len(r.Tables))
				}
				if len(r.Views) != 0 {
					t.Errorf("expected 0 view diffs, got %d", len(r.Views))
				}
			},
		},
		{
			name: "empty_vs_empty",
			src:  makeSchema("src", nil, nil),
			tgt:  makeSchema("tgt", nil, nil),
			check: func(t *testing.T, r *DiffResult) {
				if !r.Identical {
					t.Error("expected Identical == true")
				}
			},
		},

		// ── Table-level changes ──────────────────────────────────────────────

		{
			name: "table_added_in_target",
			src:  makeSchema("src", nil, nil),
			tgt:  makeSchema("tgt", []schema.Table{usersTable}, nil),
			check: func(t *testing.T, r *DiffResult) {
				if r.Identical {
					t.Fatal("expected Identical == false")
				}
				if len(r.Tables) != 1 {
					t.Fatalf("expected 1 table diff, got %d", len(r.Tables))
				}
				td := r.Tables[0]
				if td.Change != Added {
					t.Errorf("expected Added, got %s", td.Change)
				}
				if td.Name != "users" {
					t.Errorf("expected table name 'users', got %s", td.Name)
				}
				// Columns should be populated with Added diffs pointing to Target.
				if len(td.Columns) != len(usersTable.Columns) {
					t.Errorf("expected %d column diffs, got %d", len(usersTable.Columns), len(td.Columns))
				}
				for _, cd := range td.Columns {
					if cd.Change != Added {
						t.Errorf("column %q: expected Added, got %s", cd.Name, cd.Change)
					}
					if cd.Target == nil {
						t.Errorf("column %q: Target should not be nil for Added column", cd.Name)
					}
					if cd.Source != nil {
						t.Errorf("column %q: Source should be nil for Added column", cd.Name)
					}
				}
			},
		},
		{
			name: "table_removed_from_target",
			src:  makeSchema("src", []schema.Table{usersTable}, nil),
			tgt:  makeSchema("tgt", nil, nil),
			check: func(t *testing.T, r *DiffResult) {
				if r.Identical {
					t.Fatal("expected Identical == false")
				}
				if len(r.Tables) != 1 {
					t.Fatalf("expected 1 table diff, got %d", len(r.Tables))
				}
				td := r.Tables[0]
				if td.Change != Removed {
					t.Errorf("expected Removed, got %s", td.Change)
				}
				// Columns should be populated with Removed diffs pointing to Source.
				if len(td.Columns) != len(usersTable.Columns) {
					t.Errorf("expected %d column diffs, got %d", len(usersTable.Columns), len(td.Columns))
				}
				for _, cd := range td.Columns {
					if cd.Change != Removed {
						t.Errorf("column %q: expected Removed, got %s", cd.Name, cd.Change)
					}
					if cd.Source == nil {
						t.Errorf("column %q: Source should not be nil for Removed column", cd.Name)
					}
					if cd.Target != nil {
						t.Errorf("column %q: Target should be nil for Removed column", cd.Name)
					}
				}
			},
		},

		// ── Column-level changes ─────────────────────────────────────────────

		{
			name: "column_type_changed",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("age", "int")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("age", "bigint")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				if td.Change != Modified {
					t.Errorf("expected Modified, got %s", td.Change)
				}
				cd := findColumnDiff(td, "age")
				if cd == nil {
					t.Fatal("column 'age' not in diff")
				}
				if cd.Change != Modified {
					t.Errorf("expected Modified, got %s", cd.Change)
				}
				if cd.Source.RawType != "int" {
					t.Errorf("source RawType: want 'int', got %s", cd.Source.RawType)
				}
				if cd.Target.RawType != "bigint" {
					t.Errorf("target RawType: want 'bigint', got %s", cd.Target.RawType)
				}
			},
		},
		{
			name: "column_nullable_changed",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("email", "varchar(255)")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{colNullable("email", "varchar(255)")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				cd := findColumnDiff(td, "email")
				if cd == nil {
					t.Fatal("column 'email' not in diff")
				}
				if cd.Source.Nullable {
					t.Error("source: expected Nullable == false")
				}
				if !cd.Target.Nullable {
					t.Error("target: expected Nullable == true")
				}
			},
		},
		{
			name: "column_default_added",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("status", "varchar(20)")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{colWithDefault("status", "varchar(20)", "'active'")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				cd := findColumnDiff(td, "status")
				if cd == nil {
					t.Fatal("column 'status' not in diff")
				}
				if cd.Source.Default != nil {
					t.Errorf("source: expected nil Default, got %v", cd.Source.Default)
				}
				if cd.Target.Default == nil || *cd.Target.Default != "'active'" {
					t.Errorf("target: expected Default == \"'active'\", got %v", cd.Target.Default)
				}
			},
		},
		{
			name: "column_default_removed",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{colWithDefault("status", "varchar(20)", "'active'")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("status", "varchar(20)")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				cd := findColumnDiff(td, "status")
				if cd == nil {
					t.Fatal("column 'status' not in diff")
				}
				if cd.Source.Default == nil {
					t.Error("source: expected non-nil Default")
				}
				if cd.Target.Default != nil {
					t.Errorf("target: expected nil Default, got %v", cd.Target.Default)
				}
			},
		},
		{
			name: "column_added",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)"), colNullable("bio", "text")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				cd := findColumnDiff(td, "bio")
				if cd == nil {
					t.Fatal("column 'bio' not in diff")
				}
				if cd.Change != Added {
					t.Errorf("expected Added, got %s", cd.Change)
				}
				if cd.Source != nil {
					t.Error("Source should be nil for Added column")
				}
				if cd.Target == nil {
					t.Error("Target should not be nil for Added column")
				}
			},
		},
		{
			name: "column_removed",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), colNullable("bio", "text")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				cd := findColumnDiff(td, "bio")
				if cd == nil {
					t.Fatal("column 'bio' not in diff")
				}
				if cd.Change != Removed {
					t.Errorf("expected Removed, got %s", cd.Change)
				}
				if cd.Target != nil {
					t.Error("Target should be nil for Removed column")
				}
			},
		},

		// ── Index-level changes ──────────────────────────────────────────────

		{
			name: "index_added",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)")},
				Indexes: []schema.Index{{Name: "idx_name", Columns: []string{"name"}, Unique: false}},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				id := findIndexDiff(td, "idx_name")
				if id == nil {
					t.Fatal("index 'idx_name' not in diff")
				}
				if id.Change != Added {
					t.Errorf("expected Added, got %s", id.Change)
				}
				if id.Source != nil {
					t.Error("Source should be nil for Added index")
				}
			},
		},
		{
			name: "index_removed",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int")},
				Indexes: []schema.Index{{Name: "idx_name", Columns: []string{"name"}}},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				id := findIndexDiff(td, "idx_name")
				if id == nil {
					t.Fatal("index 'idx_name' not in diff")
				}
				if id.Change != Removed {
					t.Errorf("expected Removed, got %s", id.Change)
				}
				if id.Target != nil {
					t.Error("Target should be nil for Removed index")
				}
			},
		},
		{
			name: "index_modified_unique_flag",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)")},
				Indexes: []schema.Index{{Name: "idx_name", Columns: []string{"name"}, Unique: false}},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(100)")},
				Indexes: []schema.Index{{Name: "idx_name", Columns: []string{"name"}, Unique: true}},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				id := findIndexDiff(td, "idx_name")
				if id == nil {
					t.Fatal("index 'idx_name' not in diff")
				}
				if id.Change != Modified {
					t.Errorf("expected Modified, got %s", id.Change)
				}
				if id.Source.Unique {
					t.Error("source: expected Unique == false")
				}
				if !id.Target.Unique {
					t.Error("target: expected Unique == true")
				}
			},
		},
		{
			name: "index_modified_columns",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("a", "int"), col("b", "int")},
				Indexes: []schema.Index{{Name: "idx_ab", Columns: []string{"a"}}},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("a", "int"), col("b", "int")},
				Indexes: []schema.Index{{Name: "idx_ab", Columns: []string{"a", "b"}}},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "users")
				if td == nil {
					t.Fatal("table 'users' not in diff")
				}
				id := findIndexDiff(td, "idx_ab")
				if id == nil {
					t.Fatal("index 'idx_ab' not in diff")
				}
				if id.Change != Modified {
					t.Errorf("expected Modified, got %s", id.Change)
				}
			},
		},

		// ── Constraint-level changes ─────────────────────────────────────────

		{
			name: "constraint_added_fk",
			src: makeSchema("src", []schema.Table{
				{Name: "users", Columns: []schema.Column{col("id", "int")}},
				{Name: "orders", Columns: []schema.Column{col("user_id", "int")}},
			}, nil),
			tgt: makeSchema("tgt", []schema.Table{
				{Name: "users", Columns: []schema.Column{col("id", "int")}},
				{
					Name:    "orders",
					Columns: []schema.Column{col("user_id", "int")},
					Constraints: []schema.Constraint{{
						Name:       "fk_orders_users",
						Type:       "FOREIGN KEY",
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					}},
				},
			}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "orders")
				if td == nil {
					t.Fatal("table 'orders' not in diff")
				}
				cd := findConstraintDiff(td, "fk_orders_users")
				if cd == nil {
					t.Fatal("constraint 'fk_orders_users' not in diff")
				}
				if cd.Change != Added {
					t.Errorf("expected Added, got %s", cd.Change)
				}
				if cd.Source != nil {
					t.Error("Source should be nil for Added constraint")
				}
			},
		},
		{
			name: "constraint_removed",
			src: makeSchema("src", []schema.Table{{
				Name:    "orders",
				Columns: []schema.Column{col("user_id", "int")},
				Constraints: []schema.Constraint{{
					Name: "fk_orders_users", Type: "FOREIGN KEY",
					Columns: []string{"user_id"}, RefTable: "users", RefColumns: []string{"id"},
				}},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "orders",
				Columns: []schema.Column{col("user_id", "int")},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "orders")
				if td == nil {
					t.Fatal("table 'orders' not in diff")
				}
				cd := findConstraintDiff(td, "fk_orders_users")
				if cd == nil {
					t.Fatal("constraint 'fk_orders_users' not in diff")
				}
				if cd.Change != Removed {
					t.Errorf("expected Removed, got %s", cd.Change)
				}
			},
		},
		{
			name: "constraint_modified",
			src: makeSchema("src", []schema.Table{{
				Name:    "orders",
				Columns: []schema.Column{col("user_id", "int")},
				Constraints: []schema.Constraint{{
					Name: "fk_orders_users", Type: "FOREIGN KEY",
					Columns: []string{"user_id"}, RefTable: "users", RefColumns: []string{"id"},
				}},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "orders",
				Columns: []schema.Column{col("user_id", "int")},
				Constraints: []schema.Constraint{{
					Name: "fk_orders_users", Type: "FOREIGN KEY",
					Columns: []string{"user_id"}, RefTable: "accounts", RefColumns: []string{"id"},
				}},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				td := findTableDiff(r, "orders")
				if td == nil {
					t.Fatal("table 'orders' not in diff")
				}
				cd := findConstraintDiff(td, "fk_orders_users")
				if cd == nil {
					t.Fatal("constraint 'fk_orders_users' not in diff")
				}
				if cd.Change != Modified {
					t.Errorf("expected Modified, got %s", cd.Change)
				}
				if cd.Source.RefTable != "users" {
					t.Errorf("source RefTable: want 'users', got %s", cd.Source.RefTable)
				}
				if cd.Target.RefTable != "accounts" {
					t.Errorf("target RefTable: want 'accounts', got %s", cd.Target.RefTable)
				}
			},
		},

		// ── View-level changes ───────────────────────────────────────────────

		{
			name: "view_added",
			src:  makeSchema("src", nil, nil),
			tgt:  makeSchema("tgt", nil, []schema.View{{Name: "user_orders", Definition: "SELECT 1"}}),
			check: func(t *testing.T, r *DiffResult) {
				if r.Identical {
					t.Fatal("expected Identical == false")
				}
				if len(r.Views) != 1 {
					t.Fatalf("expected 1 view diff, got %d", len(r.Views))
				}
				if r.Views[0].Change != Added {
					t.Errorf("expected Added, got %s", r.Views[0].Change)
				}
				if r.Views[0].Source != nil {
					t.Error("Source should be nil for Added view")
				}
			},
		},
		{
			name: "view_removed",
			src:  makeSchema("src", nil, []schema.View{{Name: "user_orders", Definition: "SELECT 1"}}),
			tgt:  makeSchema("tgt", nil, nil),
			check: func(t *testing.T, r *DiffResult) {
				if len(r.Views) != 1 {
					t.Fatalf("expected 1 view diff, got %d", len(r.Views))
				}
				if r.Views[0].Change != Removed {
					t.Errorf("expected Removed, got %s", r.Views[0].Change)
				}
			},
		},
		{
			name: "view_modified",
			src:  makeSchema("src", nil, []schema.View{{Name: "v", Definition: "SELECT id FROM users"}}),
			tgt:  makeSchema("tgt", nil, []schema.View{{Name: "v", Definition: "SELECT id, name FROM users"}}),
			check: func(t *testing.T, r *DiffResult) {
				if len(r.Views) != 1 {
					t.Fatalf("expected 1 view diff, got %d", len(r.Views))
				}
				if r.Views[0].Change != Modified {
					t.Errorf("expected Modified, got %s", r.Views[0].Change)
				}
				if r.Views[0].Source == nil || r.Views[0].Target == nil {
					t.Error("expected both Source and Target to be set for Modified view")
				}
			},
		},
		{
			name: "view_whitespace_normalized",
			src:  makeSchema("src", nil, []schema.View{{Name: "v", Definition: "SELECT  id  FROM  users"}}),
			tgt:  makeSchema("tgt", nil, []schema.View{{Name: "v", Definition: "SELECT id FROM users"}}),
			check: func(t *testing.T, r *DiffResult) {
				if !r.Identical {
					t.Error("expected Identical == true (whitespace difference only)")
				}
			},
		},

		// ── Ignore config ────────────────────────────────────────────────────

		{
			name: "ignore_table",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(255)")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("name", "varchar(1000)")}, // different
			}}, nil),
			ignore: config.IgnoreConfig{Tables: []string{"users"}},
			check: func(t *testing.T, r *DiffResult) {
				if !r.Identical {
					t.Error("expected Identical == true (table is ignored)")
				}
			},
		},
		{
			name: "ignore_field",
			src: makeSchema("src", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("age", "int")},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name:    "users",
				Columns: []schema.Column{col("id", "int"), col("age", "bigint")}, // age differs
			}}, nil),
			ignore: config.IgnoreConfig{Fields: []string{"age"}},
			check: func(t *testing.T, r *DiffResult) {
				if !r.Identical {
					t.Error("expected Identical == true (field is ignored)")
				}
			},
		},

		// ── Deterministic output ─────────────────────────────────────────────

		{
			name: "multiple_tables_sorted",
			src:  makeSchema("src", nil, nil),
			tgt: makeSchema("tgt", []schema.Table{
				{Name: "zebra", Columns: []schema.Column{col("id", "int")}},
				{Name: "alpha", Columns: []schema.Column{col("id", "int")}},
			}, nil),
			check: func(t *testing.T, r *DiffResult) {
				if len(r.Tables) != 2 {
					t.Fatalf("expected 2 table diffs, got %d", len(r.Tables))
				}
				if r.Tables[0].Name != "alpha" {
					t.Errorf("expected first table 'alpha', got %s", r.Tables[0].Name)
				}
				if r.Tables[1].Name != "zebra" {
					t.Errorf("expected second table 'zebra', got %s", r.Tables[1].Name)
				}
			},
		},
		{
			name: "no_diff_when_identical_columns",
			src: makeSchema("src", []schema.Table{{
				Name: "users",
				Columns: []schema.Column{
					col("id", "int"),
					colNullable("bio", "text"),
					colWithDefault("status", "varchar(20)", "'active'"),
				},
				Indexes: []schema.Index{{Name: "idx_bio", Columns: []string{"bio"}}},
				Constraints: []schema.Constraint{{
					Name: "uq_id", Type: "UNIQUE", Columns: []string{"id"},
				}},
			}}, nil),
			tgt: makeSchema("tgt", []schema.Table{{
				Name: "users",
				Columns: []schema.Column{
					col("id", "int"),
					colNullable("bio", "text"),
					colWithDefault("status", "varchar(20)", "'active'"),
				},
				Indexes: []schema.Index{{Name: "idx_bio", Columns: []string{"bio"}}},
				Constraints: []schema.Constraint{{
					Name: "uq_id", Type: "UNIQUE", Columns: []string{"id"},
				}},
			}}, nil),
			check: func(t *testing.T, r *DiffResult) {
				if !r.Identical {
					t.Error("expected Identical == true")
				}
				if td := findTableDiff(r, "users"); td != nil {
					t.Error("expected no diff for identical table")
				}
			},
		},

		// ── Source/target name propagation ───────────────────────────────────

		{
			name: "result_carries_schema_names",
			src:  makeSchema("DEV", nil, nil),
			tgt:  makeSchema("PROD", nil, nil),
			check: func(t *testing.T, r *DiffResult) {
				if r.SourceName != "DEV" {
					t.Errorf("SourceName: want 'DEV', got %s", r.SourceName)
				}
				if r.TargetName != "PROD" {
					t.Errorf("TargetName: want 'PROD', got %s", r.TargetName)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Compare(tt.src, tt.tgt, tt.ignore)
			tt.check(t, result)
		})
	}
}
