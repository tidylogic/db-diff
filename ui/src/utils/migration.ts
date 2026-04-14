/**
 * Client-side migration SQL generator.
 * Ports the logic from internal/migrate/generator.go.
 */

import type {
  DiffResult,
  TableDiff,
  ColumnDiff,
  IndexDiff,
  ConstraintDiff,
  ViewDiff,
  ChangeType,
  Direction,
  Dialect,
  SelectionState,
  Column,
  Index,
  Constraint,
} from '../types'

// ── Helpers ───────────────────────────────────────────────────────────────────

function quoteIdent(name: string, dialect: Dialect): string {
  if (dialect === 'postgres') {
    return '"' + name.replace(/"/g, '""') + '"'
  }
  return '`' + name.replace(/`/g, '``') + '`'
}

function invertChange(c: ChangeType): ChangeType {
  if (c === 'added') return 'removed'
  if (c === 'removed') return 'added'
  return c
}

function effectiveChange(c: ChangeType, direction: Direction): ChangeType {
  return direction === 'target_to_source' ? invertChange(c) : c
}

// ── Column SQL ────────────────────────────────────────────────────────────────

function colSpec(col: Column): string {
  const nullStr = col.Nullable ? ' NULL' : ' NOT NULL'
  const defStr = col.Default !== null ? ` DEFAULT ${col.Default}` : ''
  return `${col.RawType}${nullStr}${defStr}`
}

function columnStmts(
  table: string,
  cd: ColumnDiff,
  direction: Direction,
  dialect: Dialect,
): string[] {
  const change = effectiveChange(cd.Change, direction)
  const t = quoteIdent(table, dialect)
  const col = quoteIdent(cd.Name, dialect)

  switch (change) {
    case 'removed':
      return [`ALTER TABLE ${t} DROP COLUMN ${col}`]

    case 'added': {
      const def = direction === 'target_to_source' ? cd.Source : cd.Target
      if (!def) return []
      return [`ALTER TABLE ${t} ADD COLUMN ${col} ${colSpec(def)}`]
    }

    case 'modified': {
      const def = direction === 'target_to_source' ? cd.Source : cd.Target
      if (!def) return []

      if (dialect === 'mysql') {
        return [`ALTER TABLE ${t} MODIFY COLUMN ${col} ${colSpec(def)}`]
      }

      // postgres: split into granular statements
      const stmts: string[] = []
      const src = direction === 'target_to_source' ? cd.Target : cd.Source
      const tgt = def

      if (src && src.RawType !== tgt.RawType) {
        stmts.push(`ALTER TABLE ${t} ALTER COLUMN ${col} TYPE ${tgt.RawType}`)
      }
      if (src && src.Nullable !== tgt.Nullable) {
        stmts.push(
          tgt.Nullable
            ? `ALTER TABLE ${t} ALTER COLUMN ${col} DROP NOT NULL`
            : `ALTER TABLE ${t} ALTER COLUMN ${col} SET NOT NULL`,
        )
      }
      if (src && src.Default !== tgt.Default) {
        if (tgt.Default !== null) {
          stmts.push(
            `ALTER TABLE ${t} ALTER COLUMN ${col} SET DEFAULT ${tgt.Default}`,
          )
        } else {
          stmts.push(`ALTER TABLE ${t} ALTER COLUMN ${col} DROP DEFAULT`)
        }
      }
      return stmts
    }
  }
}

// ── Index SQL ─────────────────────────────────────────────────────────────────

function createIndexSQL(table: string, idx: Index, dialect: Dialect): string {
  const unique = idx.Unique && !idx.IsPrimary ? 'UNIQUE ' : ''
  const cols = idx.Columns.map((c) => quoteIdent(c, dialect)).join(', ')
  return `CREATE ${unique}INDEX ${quoteIdent(idx.Name, dialect)} ON ${quoteIdent(table, dialect)} (${cols})`
}

function dropIndexSQL(table: string, idx: Index, dialect: Dialect): string {
  if (dialect === 'mysql') {
    return `ALTER TABLE ${quoteIdent(table, dialect)} DROP INDEX ${quoteIdent(idx.Name, dialect)}`
  }
  return `DROP INDEX ${quoteIdent(idx.Name, dialect)}`
}

function indexStmts(
  table: string,
  id: IndexDiff,
  direction: Direction,
  dialect: Dialect,
): string[] {
  const change = effectiveChange(id.Change, direction)

  switch (change) {
    case 'added': {
      const idx = direction === 'target_to_source' ? id.Source : id.Target
      if (!idx) return []
      return [createIndexSQL(table, idx, dialect)]
    }
    case 'removed': {
      const idx = direction === 'target_to_source' ? id.Target : id.Source
      if (!idx) return []
      return [dropIndexSQL(table, idx, dialect)]
    }
    case 'modified': {
      const src = direction === 'target_to_source' ? id.Target : id.Source
      const tgt = direction === 'target_to_source' ? id.Source : id.Target
      const stmts: string[] = []
      if (src) stmts.push(dropIndexSQL(table, src, dialect))
      if (tgt) stmts.push(createIndexSQL(table, tgt, dialect))
      return stmts
    }
  }
}

// ── Constraint SQL ────────────────────────────────────────────────────────────

function addConstraintSQL(
  table: string,
  c: Constraint,
  dialect: Dialect,
): string {
  const t = quoteIdent(table, dialect)
  const name = quoteIdent(c.Name, dialect)
  const cols = c.Columns.map((col) => quoteIdent(col, dialect)).join(', ')

  switch (c.Type) {
    case 'FOREIGN KEY': {
      const refCols = c.RefColumns.map((rc) =>
        quoteIdent(rc, dialect),
      ).join(', ')
      return (
        `ALTER TABLE ${t} ADD CONSTRAINT ${name} FOREIGN KEY (${cols}) ` +
        `REFERENCES ${quoteIdent(c.RefTable, dialect)} (${refCols})`
      )
    }
    case 'UNIQUE':
      return `ALTER TABLE ${t} ADD CONSTRAINT ${name} UNIQUE (${cols})`
    default:
      return `ALTER TABLE ${t} ADD CONSTRAINT ${name} ${c.Type} (${cols})`
  }
}

function constraintStmts(
  table: string,
  cd: ConstraintDiff,
  direction: Direction,
  dialect: Dialect,
): string[] {
  const change = effectiveChange(cd.Change, direction)
  const t = quoteIdent(table, dialect)

  switch (change) {
    case 'added': {
      const c = direction === 'target_to_source' ? cd.Source : cd.Target
      if (!c) return []
      return [addConstraintSQL(table, c, dialect)]
    }
    case 'removed': {
      const c = direction === 'target_to_source' ? cd.Target : cd.Source
      if (!c) return []
      return [
        `ALTER TABLE ${t} DROP CONSTRAINT ${quoteIdent(c.Name, dialect)}`,
      ]
    }
    case 'modified': {
      const src = direction === 'target_to_source' ? cd.Target : cd.Source
      const tgt = direction === 'target_to_source' ? cd.Source : cd.Target
      const stmts: string[] = []
      if (src) {
        stmts.push(
          `ALTER TABLE ${t} DROP CONSTRAINT ${quoteIdent(src.Name, dialect)}`,
        )
      }
      if (tgt) stmts.push(addConstraintSQL(table, tgt, dialect))
      return stmts
    }
  }
}

// ── Table SQL ─────────────────────────────────────────────────────────────────

function tableStmts(
  td: TableDiff,
  selection: SelectionState,
  direction: Direction,
  dialect: Dialect,
): string[] {
  const change = effectiveChange(td.Change, direction)
  const stmts: string[] = []

  switch (change) {
    case 'added':
      stmts.push(
        `-- CREATE TABLE ${td.Name}  (full DDL not available in diff-only mode)`,
      )
      break

    case 'removed':
      stmts.push(`DROP TABLE ${quoteIdent(td.Name, dialect)}`)
      break

    case 'modified': {
      const selectedCols = selection.columns[td.Name] ?? new Set<string>()
      const selectedIdxs = selection.indexes[td.Name] ?? new Set<string>()
      const selectedConsts =
        selection.constraints[td.Name] ?? new Set<string>()

      for (const cd of td.Columns) {
        if (!selectedCols.has(cd.Name)) continue
        stmts.push(...columnStmts(td.Name, cd, direction, dialect))
      }
      for (const id of td.Indexes) {
        if (!selectedIdxs.has(id.Name)) continue
        stmts.push(...indexStmts(td.Name, id, direction, dialect))
      }
      for (const cd of td.Constraints) {
        if (!selectedConsts.has(cd.Name)) continue
        stmts.push(...constraintStmts(td.Name, cd, direction, dialect))
      }
      break
    }
  }

  return stmts
}

// ── View SQL ──────────────────────────────────────────────────────────────────

function viewStmts(vd: ViewDiff, direction: Direction, dialect: Dialect): string[] {
  const change = effectiveChange(vd.Change, direction)

  switch (change) {
    case 'added': {
      const v = direction === 'target_to_source' ? vd.Source : vd.Target
      if (!v) return []
      return [
        `CREATE VIEW ${quoteIdent(vd.Name, dialect)} AS\n${v.Definition}`,
      ]
    }
    case 'removed':
      return [`DROP VIEW ${quoteIdent(vd.Name, dialect)}`]

    case 'modified': {
      const tgt = direction === 'target_to_source' ? vd.Source : vd.Target
      if (!tgt) return []
      return [
        `DROP VIEW IF EXISTS ${quoteIdent(vd.Name, dialect)}`,
        `CREATE VIEW ${quoteIdent(vd.Name, dialect)} AS\n${tgt.Definition}`,
      ]
    }
  }
}

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Generates migration SQL from the diff result, filtered to selected items.
 */
export function generateSQL(
  result: DiffResult,
  selection: SelectionState,
  direction: Direction,
  dialect: Dialect,
): string {
  const src =
    direction === 'source_to_target' ? result.SourceName : result.TargetName
  const tgt =
    direction === 'source_to_target' ? result.TargetName : result.SourceName

  const now = new Date().toISOString()
  const lines: string[] = [
    `-- Generated by db-diff: ${src} → ${tgt}  (${now})`,
    `-- Dialect: ${dialect}`,
    `-- Direction: ${direction}`,
    '',
  ]

  for (const td of result.Tables) {
    if (!selection.tables.has(td.Name)) continue
    const stmts = tableStmts(td, selection, direction, dialect)
    if (stmts.length > 0) {
      lines.push(`-- TABLE: ${td.Name}`)
      for (const s of stmts) {
        lines.push(s + ';')
      }
      lines.push('')
    }
  }

  for (const vd of result.Views) {
    if (!selection.views.has(vd.Name)) continue
    const stmts = viewStmts(vd, direction, dialect)
    if (stmts.length > 0) {
      lines.push(`-- VIEW: ${vd.Name}`)
      for (const s of stmts) {
        lines.push(s + ';')
      }
      lines.push('')
    }
  }

  return lines.join('\n')
}

/**
 * Builds the initial all-selected SelectionState from a DiffResult.
 */
export function buildInitialSelection(result: DiffResult): SelectionState {
  const tables = new Set(result.Tables.map((t) => t.Name))
  const columns: Record<string, Set<string>> = {}
  const indexes: Record<string, Set<string>> = {}
  const constraints: Record<string, Set<string>> = {}

  for (const td of result.Tables) {
    if (td.Change === 'modified') {
      columns[td.Name] = new Set(td.Columns.map((c) => c.Name))
      indexes[td.Name] = new Set(td.Indexes.map((i) => i.Name))
      constraints[td.Name] = new Set(td.Constraints.map((c) => c.Name))
    }
  }

  const views = new Set(result.Views.map((v) => v.Name))
  return { tables, columns, indexes, constraints, views }
}

/**
 * Computes stats from a DiffResult.
 */
export function computeStats(result: DiffResult) {
  let tablesAdded = 0,
    tablesRemoved = 0,
    tablesModified = 0
  let columnsChanged = 0,
    indexesChanged = 0,
    constraintsChanged = 0

  for (const td of result.Tables) {
    if (td.Change === 'added') tablesAdded++
    else if (td.Change === 'removed') tablesRemoved++
    else {
      tablesModified++
      columnsChanged += td.Columns.length
      indexesChanged += td.Indexes.length
      constraintsChanged += td.Constraints.length
    }
  }

  let viewsAdded = 0, viewsRemoved = 0, viewsModified = 0
  for (const vd of result.Views) {
    if (vd.Change === 'added') viewsAdded++
    else if (vd.Change === 'removed') viewsRemoved++
    else viewsModified++
  }

  return {
    tablesAdded,
    tablesRemoved,
    tablesModified,
    viewsAdded,
    viewsRemoved,
    viewsModified,
    columnsChanged,
    indexesChanged,
    constraintsChanged,
  }
}
