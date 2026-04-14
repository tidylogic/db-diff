package diff

import "db-diff/internal/schema"

// ChangeType describes the kind of change between source and target.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Modified ChangeType = "modified"
)

// DiffResult is the top-level output of the comparison engine.
type DiffResult struct {
	SourceName string
	TargetName string
	Tables     []TableDiff
	Views      []ViewDiff
	Identical  bool
}

// TableDiff represents changes to a single table.
type TableDiff struct {
	Name        string
	Change      ChangeType
	Columns     []ColumnDiff
	Indexes     []IndexDiff
	Constraints []ConstraintDiff
}

// ColumnDiff represents a change to a single column.
type ColumnDiff struct {
	Name   string
	Change ChangeType
	Source *schema.Column // nil when Change == Added
	Target *schema.Column // nil when Change == Removed
}

// IndexDiff represents a change to a single index.
type IndexDiff struct {
	Name   string
	Change ChangeType
	Source *schema.Index // nil when Change == Added
	Target *schema.Index // nil when Change == Removed
}

// ConstraintDiff represents a change to a single constraint.
type ConstraintDiff struct {
	Name   string
	Change ChangeType
	Source *schema.Constraint // nil when Change == Added
	Target *schema.Constraint // nil when Change == Removed
}

// ViewDiff represents a change to a view.
type ViewDiff struct {
	Name   string
	Change ChangeType
	Source *schema.View // nil when Change == Added
	Target *schema.View // nil when Change == Removed
}
