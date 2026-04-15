package migrate

import (
	"strings"
	"testing"

	"github.com/tidylogic/db-diff/internal/diff"
	"github.com/tidylogic/db-diff/internal/schema"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

func colPtr(name, rawType string, nullable bool, def *string) *schema.Column {
	return &schema.Column{
		Name:     name,
		DataType: rawType,
		RawType:  rawType,
		Nullable: nullable,
		Default:  def,
	}
}

func mustContain(t *testing.T, got string, wants []string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("want %q in output\ngot:\n%s", w, got)
		}
	}
}

func mustNotContain(t *testing.T, got string, absents []string) {
	t.Helper()
	for _, a := range absents {
		if strings.Contains(got, a) {
			t.Errorf("don't want %q in output\ngot:\n%s", a, got)
		}
	}
}

// ── TestGenerate ──────────────────────────────────────────────────────────────

func TestGenerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		direction    string
		dialect      string
		result       *diff.DiffResult
		wantContains []string
		wantAbsent   []string
	}{
		// ── MySQL: column operations ──────────────────────────────────────────
		{
			// Column exists only in target (Added); source_to_target propagates
			// source → target, so this column must be dropped from target.
			name:      "mysql_add_column",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "bio",
						Change: diff.Added,
						Target: colPtr("bio", "text", true, nil),
					}},
				}},
			},
			wantContains: []string{"DROP COLUMN `bio`"},
			wantAbsent:   []string{"ADD COLUMN"},
		},
		{
			// Column exists only in source (Removed); source_to_target propagates
			// source → target, so this column must be added to target.
			name:      "mysql_drop_column",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "bio",
						Change: diff.Removed,
						Source: colPtr("bio", "text", true, nil),
					}},
				}},
			},
			wantContains: []string{"ALTER TABLE `users` ADD COLUMN `bio` text NULL"},
			wantAbsent:   []string{"DROP COLUMN"},
		},
		{
			// source_to_target: target gets source's column type (int, not bigint).
			name:      "mysql_modify_column_type",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "bigint", false, nil),
					}},
				}},
			},
			wantContains: []string{"ALTER TABLE `users` MODIFY COLUMN `age` int NOT NULL"},
			wantAbsent:   []string{"bigint"},
		},
		{
			// source_to_target: target gets source's nullability (NOT NULL).
			name:      "mysql_modify_column_nullable",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "int", true, nil),
					}},
				}},
			},
			wantContains: []string{"MODIFY COLUMN `age` int NOT NULL"},
			wantAbsent:   []string{"int NULL"},
		},
		{
			// source_to_target: target gets source's definition (no default).
			name:      "mysql_modify_column_add_default",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "int", false, strPtr("0")),
					}},
				}},
			},
			wantContains: []string{"MODIFY COLUMN `age` int NOT NULL"},
			wantAbsent:   []string{"DEFAULT"},
		},
		{
			// source_to_target: target gets source's definition (includes default 0).
			name:      "mysql_modify_column_drop_default",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, strPtr("0")),
						Target: colPtr("age", "int", false, nil),
					}},
				}},
			},
			wantContains: []string{"MODIFY COLUMN `age` int NOT NULL", "DEFAULT 0"},
		},
		// ── MySQL: table operations ────────────────────────────────────────────
		{
			// Table exists only in source (Removed); source_to_target must add it
			// to target — emits a CREATE TABLE placeholder.
			name:      "mysql_drop_table",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Removed,
				}},
			},
			wantContains: []string{"-- CREATE TABLE users"},
			wantAbsent:   []string{"DROP TABLE"},
		},
		{
			// Table exists only in target (Added); source_to_target must drop it
			// from target.
			name:      "mysql_add_table",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Added,
				}},
			},
			wantContains: []string{"DROP TABLE `users`"},
			wantAbsent:   []string{"CREATE TABLE"},
		},
		// ── MySQL: index operations ────────────────────────────────────────────
		{
			// Index exists only in target (Added); source_to_target must drop it.
			name:      "mysql_create_index",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Indexes: []diff.IndexDiff{{
						Name:   "idx_users_age",
						Change: diff.Added,
						Target: &schema.Index{Name: "idx_users_age", Columns: []string{"age"}},
					}},
				}},
			},
			wantContains: []string{"ALTER TABLE `users` DROP INDEX `idx_users_age`"},
			wantAbsent:   []string{"CREATE INDEX"},
		},
		{
			// Unique index exists only in target (Added); source_to_target must drop it.
			name:      "mysql_create_unique_index",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Indexes: []diff.IndexDiff{{
						Name:   "uq_users_email",
						Change: diff.Added,
						Target: &schema.Index{Name: "uq_users_email", Columns: []string{"email"}, Unique: true},
					}},
				}},
			},
			wantContains: []string{"ALTER TABLE `users` DROP INDEX `uq_users_email`"},
			wantAbsent:   []string{"CREATE INDEX"},
		},
		{
			// Index exists only in source (Removed); source_to_target must create it in target.
			name:      "mysql_drop_index",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Indexes: []diff.IndexDiff{{
						Name:   "idx_users_age",
						Change: diff.Removed,
						Source: &schema.Index{Name: "idx_users_age", Columns: []string{"age"}},
					}},
				}},
			},
			wantContains: []string{"CREATE INDEX `idx_users_age` ON `users` (`age`)"},
			wantAbsent:   []string{"DROP INDEX"},
		},
		{
			// Modified index; source_to_target drops target version, creates source version.
			name:      "mysql_modify_index",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Indexes: []diff.IndexDiff{{
						Name:   "idx_users_age",
						Change: diff.Modified,
						Source: &schema.Index{Name: "idx_users_age", Columns: []string{"age"}},
						Target: &schema.Index{Name: "idx_users_age", Columns: []string{"age", "username"}},
					}},
				}},
			},
			wantContains: []string{
				"ALTER TABLE `users` DROP INDEX `idx_users_age`",
				"CREATE INDEX `idx_users_age` ON `users` (`age`)",
			},
			wantAbsent: []string{"`username`"},
		},
		// ── MySQL: constraint operations ───────────────────────────────────────
		{
			// FK exists only in target (Added); source_to_target must drop it.
			name:      "mysql_add_fk",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "orders",
					Change: diff.Modified,
					Constraints: []diff.ConstraintDiff{{
						Name:   "fk_orders_users",
						Change: diff.Added,
						Target: &schema.Constraint{
							Name:       "fk_orders_users",
							Type:       "FOREIGN KEY",
							Columns:    []string{"user_id"},
							RefTable:   "users",
							RefColumns: []string{"id"},
						},
					}},
				}},
			},
			wantContains: []string{
				"ALTER TABLE `orders` DROP CONSTRAINT `fk_orders_users`",
			},
			wantAbsent: []string{"ADD CONSTRAINT"},
		},
		{
			// FK exists only in source (Removed); source_to_target must add it to target.
			name:      "mysql_drop_constraint",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "orders",
					Change: diff.Modified,
					Constraints: []diff.ConstraintDiff{{
						Name:   "fk_orders_users",
						Change: diff.Removed,
						Source: &schema.Constraint{
							Name:       "fk_orders_users",
							Type:       "FOREIGN KEY",
							Columns:    []string{"user_id"},
							RefTable:   "users",
							RefColumns: []string{"id"},
						},
					}},
				}},
			},
			wantContains: []string{"ALTER TABLE `orders` ADD CONSTRAINT `fk_orders_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)"},
			wantAbsent:   []string{"DROP CONSTRAINT"},
		},
		{
			// Unique constraint exists only in target (Added); source_to_target must drop it.
			name:      "mysql_add_unique_constraint",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Constraints: []diff.ConstraintDiff{{
						Name:   "uq_email",
						Change: diff.Added,
						Target: &schema.Constraint{
							Name:    "uq_email",
							Type:    "UNIQUE",
							Columns: []string{"email"},
						},
					}},
				}},
			},
			wantContains: []string{"ALTER TABLE `users` DROP CONSTRAINT `uq_email`"},
			wantAbsent:   []string{"ADD CONSTRAINT"},
		},
		// ── MySQL: view operations ─────────────────────────────────────────────
		{
			// View exists only in target (Added); source_to_target must drop it.
			name:      "mysql_create_view",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Views: []diff.ViewDiff{{
					Name:   "user_orders",
					Change: diff.Added,
					Target: &schema.View{
						Name:       "user_orders",
						Definition: "SELECT u.username, o.amount FROM users u JOIN orders o ON u.id = o.user_id",
					},
				}},
			},
			wantContains: []string{"DROP VIEW `user_orders`"},
			wantAbsent:   []string{"CREATE VIEW"},
		},
		{
			// View exists only in source (Removed); source_to_target must create it in target.
			name:      "mysql_drop_view",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Views: []diff.ViewDiff{{
					Name:   "user_orders",
					Change: diff.Removed,
					Source: &schema.View{Name: "user_orders", Definition: "SELECT 1"},
				}},
			},
			wantContains: []string{"CREATE VIEW `user_orders` AS", "SELECT 1"},
			wantAbsent:   []string{"DROP VIEW"},
		},
		{
			// Modified view; source_to_target recreates it with source's definition.
			name:      "mysql_modify_view",
			direction: "source_to_target",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Views: []diff.ViewDiff{{
					Name:   "user_orders",
					Change: diff.Modified,
					Source: &schema.View{Name: "user_orders", Definition: "SELECT u.username FROM users u"},
					Target: &schema.View{
						Name:       "user_orders",
						Definition: "SELECT u.username, o.amount FROM users u JOIN orders o ON u.id = o.user_id",
					},
				}},
			},
			wantContains: []string{
				"DROP VIEW IF EXISTS `user_orders`",
				"CREATE VIEW `user_orders` AS",
				"SELECT u.username FROM users u",
			},
			wantAbsent: []string{"o.amount"},
		},
		// ── PostgreSQL: column operations ──────────────────────────────────────
		{
			// Column exists only in target (Added); source_to_target must drop it.
			name:      "pg_add_column",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "bio",
						Change: diff.Added,
						Target: colPtr("bio", "text", true, nil),
					}},
				}},
			},
			wantContains: []string{`DROP COLUMN "bio"`},
			wantAbsent:   []string{"ADD COLUMN"},
		},
		{
			// Column exists only in source (Removed); source_to_target must add it to target.
			name:      "pg_drop_column",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "bio",
						Change: diff.Removed,
						Source: colPtr("bio", "text", true, nil),
					}},
				}},
			},
			wantContains: []string{`ALTER TABLE "users" ADD COLUMN "bio" text NULL`},
			wantAbsent:   []string{"DROP COLUMN"},
		},
		{
			// source_to_target: target gets source's type (int, not int8).
			name:      "pg_modify_type_only",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "int8", false, nil),
					}},
				}},
			},
			wantContains: []string{`ALTER COLUMN "age" TYPE int`},
			wantAbsent:   []string{"int8", "SET NOT NULL", "DROP NOT NULL", "SET DEFAULT", "DROP DEFAULT"},
		},
		{
			// source_to_target: target gets source's nullability (NOT NULL).
			// Source is NOT NULL (false), target is NULL (true).
			name:      "pg_modify_nullable_to_null",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "int", true, nil),
					}},
				}},
			},
			wantContains: []string{`ALTER COLUMN "age" SET NOT NULL`},
			wantAbsent:   []string{"TYPE", "DROP NOT NULL", "SET DEFAULT", "DROP DEFAULT"},
		},
		{
			// source_to_target: target gets source's nullability (NULL).
			// Source is NULL (true), target is NOT NULL (false).
			name:      "pg_modify_nullable_to_not_null",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", true, nil),
						Target: colPtr("age", "int", false, nil),
					}},
				}},
			},
			wantContains: []string{`ALTER COLUMN "age" DROP NOT NULL`},
			wantAbsent:   []string{"TYPE", "SET NOT NULL", "SET DEFAULT", "DROP DEFAULT"},
		},
		{
			// source_to_target: target gets source's default (none → DROP DEFAULT).
			// Source has no default; target has default 0.
			name:      "pg_modify_set_default",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "int", false, strPtr("0")),
					}},
				}},
			},
			wantContains: []string{`ALTER COLUMN "age" DROP DEFAULT`},
			wantAbsent:   []string{"TYPE", "SET NOT NULL", "DROP NOT NULL", "SET DEFAULT"},
		},
		{
			// source_to_target: target gets source's default (0 → SET DEFAULT 0).
			// Source has default 0; target has no default.
			name:      "pg_modify_drop_default",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, strPtr("0")),
						Target: colPtr("age", "int", false, nil),
					}},
				}},
			},
			wantContains: []string{`ALTER COLUMN "age" SET DEFAULT 0`},
			wantAbsent:   []string{"TYPE", "SET NOT NULL", "DROP NOT NULL", "DROP DEFAULT"},
		},
		{
			// source_to_target: all three changes use source definition.
			name:      "pg_modify_all_three",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, strPtr("0")),
						Target: colPtr("age", "int8", true, strPtr("42")),
					}},
				}},
			},
			wantContains: []string{
				`ALTER COLUMN "age" TYPE int`,
				`ALTER COLUMN "age" SET NOT NULL`,
				`ALTER COLUMN "age" SET DEFAULT 0`,
			},
			wantAbsent: []string{"int8", "DROP NOT NULL", "SET DEFAULT 42"},
		},
		// ── PostgreSQL: index operations ───────────────────────────────────────
		{
			// Index exists only in target (Added); source_to_target must drop it.
			name:      "pg_create_index",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Indexes: []diff.IndexDiff{{
						Name:   "idx_users_age",
						Change: diff.Added,
						Target: &schema.Index{Name: "idx_users_age", Columns: []string{"age"}},
					}},
				}},
			},
			wantContains: []string{`DROP INDEX "idx_users_age"`},
			wantAbsent:   []string{"CREATE INDEX"},
		},
		{
			// Index exists only in source (Removed); source_to_target must create it in target.
			name:      "pg_drop_index",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Indexes: []diff.IndexDiff{{
						Name:   "idx_users_age",
						Change: diff.Removed,
						Source: &schema.Index{Name: "idx_users_age", Columns: []string{"age"}},
					}},
				}},
			},
			wantContains: []string{`CREATE INDEX "idx_users_age" ON "users" ("age")`},
			wantAbsent:   []string{"ALTER TABLE", "DROP INDEX"},
		},
		// ── PostgreSQL: constraint operations ──────────────────────────────────
		{
			// FK exists only in target (Added); source_to_target must drop it.
			name:      "pg_add_fk",
			direction: "source_to_target",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "orders",
					Change: diff.Modified,
					Constraints: []diff.ConstraintDiff{{
						Name:   "fk_orders_users",
						Change: diff.Added,
						Target: &schema.Constraint{
							Name:       "fk_orders_users",
							Type:       "FOREIGN KEY",
							Columns:    []string{"user_id"},
							RefTable:   "users",
							RefColumns: []string{"id"},
						},
					}},
				}},
			},
			wantContains: []string{
				`ALTER TABLE "orders" DROP CONSTRAINT "fk_orders_users"`,
			},
			wantAbsent: []string{"ADD CONSTRAINT"},
		},
		// ── Direction reversal ─────────────────────────────────────────────────
		{
			// Column Added in target; target_to_source propagates target → source,
			// so this column must be added to source.
			name:      "mysql_reverse_add_becomes_drop",
			direction: "target_to_source",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "bio",
						Change: diff.Added,
						Target: colPtr("bio", "text", true, nil),
					}},
				}},
			},
			wantContains: []string{"ADD COLUMN `bio` text NULL"},
			wantAbsent:   []string{"DROP COLUMN"},
		},
		{
			// Table Removed (only in source); target_to_source propagates target → source,
			// so this table must be dropped from source.
			name:      "mysql_reverse_table_removed_becomes_created",
			direction: "target_to_source",
			dialect:   "mysql",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Removed,
				}},
			},
			wantContains: []string{"DROP TABLE `users`"},
			wantAbsent:   []string{"CREATE TABLE"},
		},
		{
			// Source: not nullable, Target: nullable; target_to_source → use target def → DROP NOT NULL.
			name:      "pg_reverse_nullable",
			direction: "target_to_source",
			dialect:   "postgres",
			result: &diff.DiffResult{
				Tables: []diff.TableDiff{{
					Name:   "users",
					Change: diff.Modified,
					Columns: []diff.ColumnDiff{{
						Name:   "age",
						Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "int", true, nil),
					}},
				}},
			},
			wantContains: []string{`ALTER COLUMN "age" DROP NOT NULL`},
			wantAbsent:   []string{"SET NOT NULL"},
		},
		// ── Header content ─────────────────────────────────────────────────────
		{
			name:      "header_source_to_target",
			direction: "source_to_target",
			dialect:   "mysql",
			result:    &diff.DiffResult{SourceName: "src_db", TargetName: "tgt_db"},
			wantContains: []string{
				"-- Generated by db-diff: src_db \u2192 tgt_db",
				"-- Dialect: mysql",
				"-- Direction: source_to_target",
			},
		},
		{
			name:      "header_target_to_source",
			direction: "target_to_source",
			dialect:   "mysql",
			result:    &diff.DiffResult{SourceName: "src_db", TargetName: "tgt_db"},
			wantContains: []string{
				"-- Generated by db-diff: tgt_db \u2192 src_db",
				"-- Dialect: mysql",
				"-- Direction: target_to_source",
			},
		},
		// ── Edge cases ────────────────────────────────────────────────────────
		{
			name:         "no_sql_when_identical",
			direction:    "source_to_target",
			dialect:      "mysql",
			result:       &diff.DiffResult{SourceName: "src", TargetName: "tgt"},
			wantContains: []string{"-- Generated by db-diff:", "-- Dialect:", "-- Direction:"},
			wantAbsent:   []string{"ALTER TABLE", "DROP TABLE", "CREATE INDEX", "DROP VIEW", "CREATE VIEW"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Generate(tt.result, tt.direction, tt.dialect)
			if err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}
			mustContain(t, got, tt.wantContains)
			mustNotContain(t, got, tt.wantAbsent)
		})
	}
}

// ── TestGenerateFiltered ──────────────────────────────────────────────────────

func TestGenerateFiltered(t *testing.T) {
	t.Parallel()

	// Base diff: users table modified with 3 columns + 1 index + 1 view
	baseResult := &diff.DiffResult{
		SourceName: "src",
		TargetName: "tgt",
		Tables: []diff.TableDiff{
			{
				Name:   "users",
				Change: diff.Modified,
				Columns: []diff.ColumnDiff{
					{Name: "bio", Change: diff.Added, Target: colPtr("bio", "text", true, nil)},
					{Name: "age", Change: diff.Modified,
						Source: colPtr("age", "int", false, nil),
						Target: colPtr("age", "bigint", false, nil)},
					{Name: "email", Change: diff.Modified,
						Source: colPtr("email", "varchar(100)", false, nil),
						Target: colPtr("email", "varchar(255)", false, nil)},
				},
				Indexes: []diff.IndexDiff{
					{Name: "idx_bio", Change: diff.Added,
						Target: &schema.Index{Name: "idx_bio", Columns: []string{"bio"}}},
				},
			},
			{
				Name:   "orders",
				Change: diff.Removed,
			},
		},
		Views: []diff.ViewDiff{
			{Name: "user_orders", Change: diff.Added,
				Target: &schema.View{Name: "user_orders", Definition: "SELECT 1"}},
		},
	}

	tests := []struct {
		name         string
		sel          Selection
		wantContains []string
		wantAbsent   []string
	}{
		{
			// source_to_target: bio Added in target → DROP COLUMN bio.
			// age/email unchanged in selection.
			name: "filter_one_column_of_three",
			sel: Selection{
				Tables:  []string{"users"},
				Columns: map[string][]string{"users": {"bio"}},
			},
			wantContains: []string{"DROP COLUMN `bio`"},
			wantAbsent:   []string{"`age`", "`email`", "ADD COLUMN"},
		},
		{
			name: "exclude_table_entirely",
			sel: Selection{
				Tables: []string{"users"},
				Views:  []string{"user_orders"},
			},
			// orders table excluded; users included (all columns since no column filter);
			// user_orders view included (Added in target → DROP VIEW with source_to_target).
			wantContains: []string{"`bio`", "`age`", "`email`", "user_orders"},
			wantAbsent:   []string{"DROP TABLE `orders`", "-- TABLE: orders"},
		},
		{
			name: "empty_selection_produces_only_header",
			sel:  Selection{},
			wantContains: []string{
				"-- Generated by db-diff:",
				"-- Dialect: mysql",
				"-- Direction: source_to_target",
			},
			wantAbsent: []string{"ALTER TABLE", "DROP TABLE", "CREATE VIEW"},
		},
		{
			// source_to_target: view Added in target → DROP VIEW.
			name:         "select_view_only",
			sel:          Selection{Views: []string{"user_orders"}},
			wantContains: []string{"DROP VIEW `user_orders`"},
			wantAbsent:   []string{"ALTER TABLE", "DROP TABLE", "CREATE VIEW"},
		},
		{
			// source_to_target: table Removed (only in source) → CREATE TABLE placeholder.
			name:         "select_removed_table",
			sel:          Selection{Tables: []string{"orders"}},
			wantContains: []string{"-- CREATE TABLE orders"},
			wantAbsent:   []string{"users", "user_orders", "DROP TABLE"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GenerateFiltered(baseResult, tt.sel, "source_to_target", "mysql")
			if err != nil {
				t.Fatalf("GenerateFiltered returned error: %v", err)
			}
			mustContain(t, got, tt.wantContains)
			mustNotContain(t, got, tt.wantAbsent)
		})
	}
}

// ── TestGenerateErrors ────────────────────────────────────────────────────────

func TestGenerateErrors(t *testing.T) {
	t.Parallel()
	t.Run("invalid_direction", func(t *testing.T) {
		t.Parallel()
		_, err := Generate(&diff.DiffResult{}, "invalid", "mysql")
		if err == nil {
			t.Fatal("expected error for invalid direction, got nil")
		}
		if !strings.Contains(err.Error(), "invalid direction") {
			t.Errorf("expected error to contain \"invalid direction\", got: %v", err)
		}
	})
}
