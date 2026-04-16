package diff

import (
	"sort"
	"strings"

	"github.com/tidylogic/db-diff/internal/config"
	"github.com/tidylogic/db-diff/internal/schema"
)

// Compare produces a DiffResult describing all differences between source and target.
// Tables and columns listed in ignore are skipped.
func Compare(source, target *schema.Schema, ignore config.IgnoreConfig) *DiffResult {
	result := &DiffResult{
		SourceName: source.Name,
		TargetName: target.Name,
	}

	ignoreTableSet := toSet(ignore.Tables)
	ignoreFieldSet := toSet(ignore.Fields)

	// --- Tables ---
	for name, srcTable := range source.Tables {
		if ignoreTableSet[name] {
			continue
		}
		if tgtTable, ok := target.Tables[name]; ok {
			// Table exists in both — compare
			td := compareTable(name, srcTable, tgtTable, ignoreFieldSet)
			if td != nil {
				result.Tables = append(result.Tables, *td)
			}
		} else {
			// Table only in source → Removed (store full schema so CREATE TABLE can be generated)
			result.Tables = append(result.Tables, TableDiff{
				Name:        name,
				Change:      Removed,
				Columns:     tableColumnDiffs(srcTable.Columns, Removed),
				Indexes:     tableIndexDiffs(srcTable.Indexes, Removed),
				Constraints: tableConstraintDiffs(srcTable.Constraints, Removed),
			})
		}
	}
	for name, tgtTable := range target.Tables {
		if ignoreTableSet[name] {
			continue
		}
		if _, ok := source.Tables[name]; !ok {
			// Table only in target → Added (store full schema so CREATE TABLE can be generated)
			result.Tables = append(result.Tables, TableDiff{
				Name:        name,
				Change:      Added,
				Columns:     tableColumnDiffs(tgtTable.Columns, Added),
				Indexes:     tableIndexDiffs(tgtTable.Indexes, Added),
				Constraints: tableConstraintDiffs(tgtTable.Constraints, Added),
			})
		}
	}

	// --- Views ---
	for name, srcView := range source.Views {
		sv := srcView
		if tgtView, ok := target.Views[name]; ok {
			tv := tgtView
			if normalizeSQL(srcView.Definition) != normalizeSQL(tgtView.Definition) {
				result.Views = append(result.Views, ViewDiff{
					Name:   name,
					Change: Modified,
					Source: &sv,
					Target: &tv,
				})
			}
		} else {
			result.Views = append(result.Views, ViewDiff{
				Name:   name,
				Change: Removed,
				Source: &sv,
			})
		}
	}
	for name, tgtView := range target.Views {
		tv := tgtView
		if _, ok := source.Views[name]; !ok {
			result.Views = append(result.Views, ViewDiff{
				Name:   name,
				Change: Added,
				Target: &tv,
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(result.Tables, func(i, j int) bool {
		return result.Tables[i].Name < result.Tables[j].Name
	})
	sort.Slice(result.Views, func(i, j int) bool {
		return result.Views[i].Name < result.Views[j].Name
	})

	result.Identical = len(result.Tables) == 0 && len(result.Views) == 0

	return result
}

// compareTable returns a TableDiff for tables that exist in both schemas.
// Returns nil when the tables are identical.
func compareTable(name string, src, tgt schema.Table, ignoreFields map[string]bool) *TableDiff {
	td := &TableDiff{
		Name:        name,
		Change:      Modified,
		Columns:     []ColumnDiff{},
		Indexes:     []IndexDiff{},
		Constraints: []ConstraintDiff{},
	}

	// Columns
	srcCols := columnMap(src.Columns)
	tgtCols := columnMap(tgt.Columns)

	// Preserve source ordinal order, then append new target columns
	seen := map[string]bool{}
	for _, col := range src.Columns {
		if ignoreFields[col.Name] {
			continue
		}
		seen[col.Name] = true
		sc := col
		if tc, ok := tgtCols[col.Name]; ok {
			if !columnsEqual(sc, tc) {
				td.Columns = append(td.Columns, ColumnDiff{
					Name:   col.Name,
					Change: Modified,
					Source: &sc,
					Target: &tc,
				})
			}
		} else {
			td.Columns = append(td.Columns, ColumnDiff{
				Name:   col.Name,
				Change: Removed,
				Source: &sc,
			})
		}
	}
	// Columns only in target (added)
	for _, col := range tgt.Columns {
		if ignoreFields[col.Name] || seen[col.Name] {
			continue
		}
		tc := col
		if _, ok := srcCols[col.Name]; !ok {
			td.Columns = append(td.Columns, ColumnDiff{
				Name:   col.Name,
				Change: Added,
				Target: &tc,
			})
		}
	}

	// Indexes
	srcIdx := indexMap(src.Indexes)
	tgtIdx := indexMap(tgt.Indexes)
	for name, si := range srcIdx {
		si := si
		if ti, ok := tgtIdx[name]; ok {
			if !indexesEqual(si, ti) {
				td.Indexes = append(td.Indexes, IndexDiff{
					Name:   name,
					Change: Modified,
					Source: &si,
					Target: &ti,
				})
			}
		} else {
			td.Indexes = append(td.Indexes, IndexDiff{
				Name:   name,
				Change: Removed,
				Source: &si,
			})
		}
	}
	for name, ti := range tgtIdx {
		ti := ti
		if _, ok := srcIdx[name]; !ok {
			td.Indexes = append(td.Indexes, IndexDiff{
				Name:   name,
				Change: Added,
				Target: &ti,
			})
		}
	}

	// Constraints
	srcCon := constraintMap(src.Constraints)
	tgtCon := constraintMap(tgt.Constraints)
	for name, sc := range srcCon {
		sc := sc
		if tc, ok := tgtCon[name]; ok {
			if !constraintsEqual(sc, tc) {
				td.Constraints = append(td.Constraints, ConstraintDiff{
					Name:   name,
					Change: Modified,
					Source: &sc,
					Target: &tc,
				})
			}
		} else {
			td.Constraints = append(td.Constraints, ConstraintDiff{
				Name:   name,
				Change: Removed,
				Source: &sc,
			})
		}
	}
	for name, tc := range tgtCon {
		tc := tc
		if _, ok := srcCon[name]; !ok {
			td.Constraints = append(td.Constraints, ConstraintDiff{
				Name:   name,
				Change: Added,
				Target: &tc,
			})
		}
	}

	// Sort sub-diffs for deterministic output
	sort.Slice(td.Indexes, func(i, j int) bool { return td.Indexes[i].Name < td.Indexes[j].Name })
	sort.Slice(td.Constraints, func(i, j int) bool { return td.Constraints[i].Name < td.Constraints[j].Name })

	if len(td.Columns) == 0 && len(td.Indexes) == 0 && len(td.Constraints) == 0 {
		return nil
	}
	return td
}

// --- equality helpers ---

func columnsEqual(a, b schema.Column) bool {
	return a.DataType == b.DataType &&
		a.RawType == b.RawType &&
		a.Nullable == b.Nullable &&
		ptrStringEqual(a.Default, b.Default) &&
		a.Comment == b.Comment &&
		ptrInt64Equal(a.CharMaxLen, b.CharMaxLen) &&
		ptrInt64Equal(a.NumPrecision, b.NumPrecision) &&
		ptrInt64Equal(a.NumScale, b.NumScale)
}

func indexesEqual(a, b schema.Index) bool {
	return a.Unique == b.Unique &&
		a.IsPrimary == b.IsPrimary &&
		strings.Join(a.Columns, ",") == strings.Join(b.Columns, ",")
}

func constraintsEqual(a, b schema.Constraint) bool {
	return a.Type == b.Type &&
		strings.Join(a.Columns, ",") == strings.Join(b.Columns, ",") &&
		a.RefTable == b.RefTable &&
		strings.Join(a.RefColumns, ",") == strings.Join(b.RefColumns, ",")
}

func ptrStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrInt64Equal(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// --- map helpers ---

func columnMap(cols []schema.Column) map[string]schema.Column {
	m := make(map[string]schema.Column, len(cols))
	for _, c := range cols {
		m[c.Name] = c
	}
	return m
}

func indexMap(idxs []schema.Index) map[string]schema.Index {
	m := make(map[string]schema.Index, len(idxs))
	for _, i := range idxs {
		m[i.Name] = i
	}
	return m
}

func constraintMap(cons []schema.Constraint) map[string]schema.Constraint {
	m := make(map[string]schema.Constraint, len(cons))
	for _, c := range cons {
		m[c.Name] = c
	}
	return m
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

// normalizeSQL strips extra whitespace for view definition comparison.
func normalizeSQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// tableColumnDiffs converts a slice of schema.Column to ColumnDiff entries for
// a wholly Added or Removed table (all columns share the same change type).
func tableColumnDiffs(cols []schema.Column, change ChangeType) []ColumnDiff {
	diffs := make([]ColumnDiff, len(cols))
	for i, c := range cols {
		cc := c
		d := ColumnDiff{Name: c.Name, Change: change}
		if change == Added {
			d.Target = &cc
		} else {
			d.Source = &cc
		}
		diffs[i] = d
	}
	return diffs
}

// tableIndexDiffs converts a slice of schema.Index to IndexDiff entries for
// a wholly Added or Removed table.
func tableIndexDiffs(idxs []schema.Index, change ChangeType) []IndexDiff {
	diffs := make([]IndexDiff, len(idxs))
	for i, idx := range idxs {
		ii := idx
		d := IndexDiff{Name: idx.Name, Change: change}
		if change == Added {
			d.Target = &ii
		} else {
			d.Source = &ii
		}
		diffs[i] = d
	}
	return diffs
}

// tableConstraintDiffs converts a slice of schema.Constraint to ConstraintDiff
// entries for a wholly Added or Removed table.
func tableConstraintDiffs(cons []schema.Constraint, change ChangeType) []ConstraintDiff {
	diffs := make([]ConstraintDiff, len(cons))
	for i, c := range cons {
		cc := c
		d := ConstraintDiff{Name: c.Name, Change: change}
		if change == Added {
			d.Target = &cc
		} else {
			d.Source = &cc
		}
		diffs[i] = d
	}
	return diffs
}
