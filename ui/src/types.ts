// ── Change types ─────────────────────────────────────────────────────────────

export type ChangeType = 'added' | 'removed' | 'modified'
export type Direction = 'apply_to_target' | 'apply_to_source'
export type Dialect = 'mysql' | 'postgres'

// ── Schema primitives (matching internal/schema/types.go) ────────────────────

export interface Column {
  Name: string
  OrdinalPos: number
  DataType: string
  RawType: string
  Nullable: boolean
  Default: string | null
  Comment: string
  CharMaxLen: number | null
  NumPrecision: number | null
  NumScale: number | null
}

export interface Index {
  Name: string
  Columns: string[]
  Unique: boolean
  IsPrimary: boolean
}

export interface Constraint {
  Name: string
  Type: string // 'PRIMARY KEY' | 'FOREIGN KEY' | 'UNIQUE' | 'CHECK'
  Columns: string[]
  RefTable: string
  RefColumns: string[]
}

export interface View {
  Name: string
  Definition: string
}

// ── Diff types (matching internal/diff/types.go) ─────────────────────────────

export interface ColumnDiff {
  Name: string
  Change: ChangeType
  Source: Column | null
  Target: Column | null
}

export interface IndexDiff {
  Name: string
  Change: ChangeType
  Source: Index | null
  Target: Index | null
}

export interface ConstraintDiff {
  Name: string
  Change: ChangeType
  Source: Constraint | null
  Target: Constraint | null
}

export interface TableDiff {
  Name: string
  Change: ChangeType
  Columns: ColumnDiff[]
  Indexes: IndexDiff[]
  Constraints: ConstraintDiff[]
}

export interface ViewDiff {
  Name: string
  Change: ChangeType
  Source: View | null
  Target: View | null
}

export interface DiffResult {
  SourceName: string
  TargetName: string
  Tables: TableDiff[]
  Views: ViewDiff[]
  Identical: boolean
}

// ── Selection state ───────────────────────────────────────────────────────────

/**
 * Tracks which items are selected for migration generation.
 *
 * - For added/removed tables: presence in `tables` controls inclusion.
 * - For modified tables: presence in `tables` controls inclusion, plus
 *   per-item sets for columns/indexes/constraints.
 * - Views: presence in `views` controls inclusion.
 */
export interface SelectionState {
  tables: Set<string>
  columns: Record<string, Set<string>>    // tableName → set of column names
  indexes: Record<string, Set<string>>    // tableName → set of index names
  constraints: Record<string, Set<string>> // tableName → set of constraint names
  views: Set<string>
}

// ── Statistics ────────────────────────────────────────────────────────────────

export interface DiffStats {
  tablesAdded: number    // target-only tables
  tablesRemoved: number  // source-only tables
  tablesModified: number
  viewsAdded: number
  viewsRemoved: number
  viewsModified: number
  columnsChanged: number
  indexesChanged: number
  constraintsChanged: number
}
