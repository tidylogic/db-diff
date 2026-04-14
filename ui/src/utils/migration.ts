import type {
  DiffResult,
  Direction,
  Dialect,
  SelectionState,
} from '../types'

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Posts the diff + selection to the Go /api/migrate endpoint and returns the
 * generated SQL string. Throws on network or server errors.
 *
 * Pass an AbortSignal to cancel in-flight requests when inputs change.
 */
export async function generateSQL(
  result: DiffResult,
  selection: SelectionState,
  direction: Direction,
  dialect: Dialect,
  signal?: AbortSignal,
): Promise<string> {
  const body = {
    diff: result,
    selection: {
      tables: Array.from(selection.tables),
      columns: Object.fromEntries(
        Object.entries(selection.columns).map(([k, v]) => [k, Array.from(v)]),
      ),
      indexes: Object.fromEntries(
        Object.entries(selection.indexes).map(([k, v]) => [k, Array.from(v)]),
      ),
      constraints: Object.fromEntries(
        Object.entries(selection.constraints).map(([k, v]) => [
          k,
          Array.from(v),
        ]),
      ),
      views: Array.from(selection.views),
    },
    direction,
    dialect,
  }

  const resp = await fetch('/api/migrate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal,
  })

  const data = await resp.json()
  if (!resp.ok) {
    throw new Error((data as { error?: string }).error ?? 'Server error')
  }
  return (data as { sql: string }).sql
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

  let viewsAdded = 0,
    viewsRemoved = 0,
    viewsModified = 0
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
