import type {
  DiffResult,
  SelectionState,
  TableDiff,
  ViewDiff,
  ColumnDiff,
  IndexDiff,
  ConstraintDiff,
} from '../types'
import { ChangeBadge, changeRowClass } from './ChangeBadge'

// ── Props ─────────────────────────────────────────────────────────────────────

interface Props {
  result: DiffResult
  activeItem: { type: 'table' | 'view'; name: string } | null
  selection: SelectionState
  onToggleColumn: (table: string, col: string) => void
  onToggleIndex: (table: string, idx: string) => void
  onToggleConstraint: (table: string, con: string) => void
  onToggleAllColumns: (table: string, checked: boolean) => void
  onToggleAllIndexes: (table: string, checked: boolean) => void
  onToggleAllConstraints: (table: string, checked: boolean) => void
}

// ── Section header ────────────────────────────────────────────────────────────

function SectionHeader({
  title,
  count,
  allChecked,
  onToggleAll,
}: {
  title: string
  count: number
  allChecked: boolean
  onToggleAll: (v: boolean) => void
}) {
  if (count === 0) return null
  return (
    <div className="flex items-center gap-2 px-4 py-1.5 bg-gray-50 dark:bg-gray-800/60 border-y border-gray-100 dark:border-gray-800">
      <input
        type="checkbox"
        checked={allChecked}
        onChange={(e) => onToggleAll(e.target.checked)}
        className="w-3.5 h-3.5 accent-gray-700 dark:accent-gray-300"
      />
      <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
        {title}
      </span>
      <span className="text-[10px] text-gray-400 dark:text-gray-600">{count}</span>
    </div>
  )
}

// ── Column row ────────────────────────────────────────────────────────────────

function ColumnRow({ cd, checked, onToggle }: { cd: ColumnDiff; checked: boolean; onToggle: () => void }) {
  const src = cd.Source
  const tgt = cd.Target

  return (
    <div className={`border-b border-gray-100 dark:border-gray-800 last:border-b-0 ${changeRowClass(cd.Change)}`}>
      <div className="flex items-start gap-3 px-4 py-2.5">
        <input
          type="checkbox"
          checked={checked}
          onChange={onToggle}
          className="mt-0.5 w-3.5 h-3.5 flex-shrink-0 accent-gray-700 dark:accent-gray-300"
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-mono text-xs font-semibold text-gray-800 dark:text-gray-200">
              {cd.Name}
            </span>
            <ChangeBadge change={cd.Change} size="sm" />
          </div>

          <div className="mt-1 text-xs space-y-0.5">
            {cd.Change === 'modified' && src && tgt && (
              <>
                {src.RawType !== tgt.RawType && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-gray-400 dark:text-gray-600 w-12 text-[10px]">type</span>
                    <span className="font-mono line-through text-red-500 dark:text-red-400 opacity-80">{src.RawType}</span>
                    <span className="text-gray-300 dark:text-gray-600">→</span>
                    <span className="font-mono text-emerald-600 dark:text-emerald-400">{tgt.RawType}</span>
                  </div>
                )}
                {src.Nullable !== tgt.Nullable && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-gray-400 dark:text-gray-600 w-12 text-[10px]">null</span>
                    <span className="font-mono line-through text-red-500 dark:text-red-400 opacity-80">
                      {src.Nullable ? 'NULL' : 'NOT NULL'}
                    </span>
                    <span className="text-gray-300 dark:text-gray-600">→</span>
                    <span className="font-mono text-emerald-600 dark:text-emerald-400">
                      {tgt.Nullable ? 'NULL' : 'NOT NULL'}
                    </span>
                  </div>
                )}
                {src.Default !== tgt.Default && (
                  <div className="flex items-center gap-1.5">
                    <span className="text-gray-400 dark:text-gray-600 w-12 text-[10px]">default</span>
                    <span className="font-mono line-through text-red-500 dark:text-red-400 opacity-80">
                      {src.Default ?? 'none'}
                    </span>
                    <span className="text-gray-300 dark:text-gray-600">→</span>
                    <span className="font-mono text-emerald-600 dark:text-emerald-400">
                      {tgt.Default ?? 'none'}
                    </span>
                  </div>
                )}
                {src.Comment !== tgt.Comment && src.Comment !== '' || tgt.Comment !== '' && src.Comment !== tgt.Comment ? (
                  <div className="flex items-center gap-1.5">
                    <span className="text-gray-400 dark:text-gray-600 w-12 text-[10px]">comment</span>
                    <span className="line-through text-red-500 dark:text-red-400 opacity-80">{src.Comment || '—'}</span>
                    <span className="text-gray-300 dark:text-gray-600">→</span>
                    <span className="text-emerald-600 dark:text-emerald-400">{tgt.Comment || '—'}</span>
                  </div>
                ) : null}
              </>
            )}
            {cd.Change === 'removed' && src && (
              <span className="font-mono text-red-600 dark:text-red-400 opacity-80">{src.RawType} · {src.Nullable ? 'NULL' : 'NOT NULL'}</span>
            )}
            {cd.Change === 'added' && tgt && (
              <span className="font-mono text-emerald-600 dark:text-emerald-400">{tgt.RawType} · {tgt.Nullable ? 'NULL' : 'NOT NULL'}</span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ── Index row ─────────────────────────────────────────────────────────────────

function IndexRow({ id, checked, onToggle }: { id: IndexDiff; checked: boolean; onToggle: () => void }) {
  const repr = id.Source ?? id.Target
  const srcCols = id.Source?.Columns
  const tgtCols = id.Target?.Columns

  return (
    <div className={`border-b border-gray-100 dark:border-gray-800 last:border-b-0 ${changeRowClass(id.Change)}`}>
      <div className="flex items-start gap-3 px-4 py-2.5">
        <input
          type="checkbox"
          checked={checked}
          onChange={onToggle}
          className="mt-0.5 w-3.5 h-3.5 flex-shrink-0 accent-gray-700 dark:accent-gray-300"
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-mono text-xs font-semibold text-gray-800 dark:text-gray-200">{id.Name}</span>
            <ChangeBadge change={id.Change} size="sm" />
            {repr?.Unique && !repr?.IsPrimary && (
              <span className="text-[10px] border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 rounded px-1 py-0.5">UNIQUE</span>
            )}
            {repr?.IsPrimary && (
              <span className="text-[10px] border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 rounded px-1 py-0.5">PRIMARY</span>
            )}
          </div>
          <div className="mt-1 text-xs font-mono text-gray-500 dark:text-gray-500">
            {id.Change === 'modified' && srcCols && tgtCols ? (
              <span>
                <span className="line-through text-red-500 dark:text-red-400 opacity-80">({srcCols.join(', ')})</span>
                <span className="text-gray-300 dark:text-gray-600 mx-1">→</span>
                <span className="text-emerald-600 dark:text-emerald-400">({tgtCols.join(', ')})</span>
              </span>
            ) : (
              <span>({repr?.Columns.join(', ')})</span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ── Constraint row ────────────────────────────────────────────────────────────

function ConstraintRow({ cd, checked, onToggle }: { cd: ConstraintDiff; checked: boolean; onToggle: () => void }) {
  const info = cd.Source ?? cd.Target
  return (
    <div className={`border-b border-gray-100 dark:border-gray-800 last:border-b-0 ${changeRowClass(cd.Change)}`}>
      <div className="flex items-start gap-3 px-4 py-2.5">
        <input
          type="checkbox"
          checked={checked}
          onChange={onToggle}
          className="mt-0.5 w-3.5 h-3.5 flex-shrink-0 accent-gray-700 dark:accent-gray-300"
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-mono text-xs font-semibold text-gray-800 dark:text-gray-200">{cd.Name}</span>
            <ChangeBadge change={cd.Change} size="sm" />
            {info && (
              <span className="text-[10px] border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 rounded px-1 py-0.5">
                {info.Type}
              </span>
            )}
          </div>
          {info && (
            <div className="mt-1 text-xs font-mono text-gray-500 dark:text-gray-500">
              ({info.Columns.join(', ')})
              {info.Type === 'FOREIGN KEY' && info.RefTable && (
                <span> → {info.RefTable}({info.RefColumns.join(', ')})</span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ── Table detail ──────────────────────────────────────────────────────────────

function TableDetail({
  td, selection,
  onToggleColumn, onToggleIndex, onToggleConstraint,
  onToggleAllColumns, onToggleAllIndexes, onToggleAllConstraints,
}: {
  td: TableDiff
  selection: SelectionState
  onToggleColumn: (c: string) => void
  onToggleIndex: (i: string) => void
  onToggleConstraint: (c: string) => void
  onToggleAllColumns: (v: boolean) => void
  onToggleAllIndexes: (v: boolean) => void
  onToggleAllConstraints: (v: boolean) => void
}) {
  const selCols = selection.columns[td.Name] ?? new Set<string>()
  const selIdxs = selection.indexes[td.Name] ?? new Set<string>()
  const selConsts = selection.constraints[td.Name] ?? new Set<string>()

  if (td.Change !== 'modified') {
    return (
      <div className="flex-1 flex items-center justify-center p-12">
        <div className="text-center space-y-2 max-w-xs">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {td.Change === 'added'
              ? 'This table exists only in the target database.'
              : 'This table exists only in the source database.'}
          </p>
          <p className="text-xs text-gray-400 dark:text-gray-600">
            {td.Change === 'added'
              ? 'Migration will generate a CREATE TABLE placeholder.'
              : 'Migration will generate a DROP TABLE statement.'}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto">
      <SectionHeader
        title="Columns"
        count={td.Columns.length}
        allChecked={td.Columns.length > 0 && td.Columns.every((c) => selCols.has(c.Name))}
        onToggleAll={onToggleAllColumns}
      />
      {td.Columns.map((cd) => (
        <ColumnRow key={cd.Name} cd={cd} checked={selCols.has(cd.Name)} onToggle={() => onToggleColumn(cd.Name)} />
      ))}

      {td.Indexes.length > 0 && (
        <>
          <SectionHeader
            title="Indexes"
            count={td.Indexes.length}
            allChecked={td.Indexes.every((i) => selIdxs.has(i.Name))}
            onToggleAll={onToggleAllIndexes}
          />
          {td.Indexes.map((id) => (
            <IndexRow key={id.Name} id={id} checked={selIdxs.has(id.Name)} onToggle={() => onToggleIndex(id.Name)} />
          ))}
        </>
      )}

      {td.Constraints.length > 0 && (
        <>
          <SectionHeader
            title="Constraints"
            count={td.Constraints.length}
            allChecked={td.Constraints.every((c) => selConsts.has(c.Name))}
            onToggleAll={onToggleAllConstraints}
          />
          {td.Constraints.map((cd) => (
            <ConstraintRow key={cd.Name} cd={cd} checked={selConsts.has(cd.Name)} onToggle={() => onToggleConstraint(cd.Name)} />
          ))}
        </>
      )}
    </div>
  )
}

// ── View detail ───────────────────────────────────────────────────────────────

function ViewDetail({ vd }: { vd: ViewDiff }) {
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {vd.Change === 'modified' && vd.Source && vd.Target && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-600 mb-1.5">Source</p>
            <pre className="text-xs font-mono bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-900 rounded p-3 overflow-x-auto whitespace-pre-wrap text-red-800 dark:text-red-300">
              {vd.Source.Definition}
            </pre>
          </div>
          <div>
            <p className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-600 mb-1.5">Target</p>
            <pre className="text-xs font-mono bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-900 rounded p-3 overflow-x-auto whitespace-pre-wrap text-emerald-800 dark:text-emerald-300">
              {vd.Target.Definition}
            </pre>
          </div>
        </div>
      )}
      {vd.Change === 'added' && vd.Target && (
        <pre className="text-xs font-mono bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-900 rounded p-3 overflow-x-auto whitespace-pre-wrap text-emerald-800 dark:text-emerald-300">
          {vd.Target.Definition}
        </pre>
      )}
      {vd.Change === 'removed' && vd.Source && (
        <pre className="text-xs font-mono bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-900 rounded p-3 overflow-x-auto whitespace-pre-wrap text-red-800 dark:text-red-300">
          {vd.Source.Definition}
        </pre>
      )}
    </div>
  )
}

// ── Empty state ───────────────────────────────────────────────────────────────

function EmptyState() {
  return (
    <div className="flex-1 flex items-center justify-center">
      <p className="text-xs text-gray-300 dark:text-gray-700">
        Select a table or view
      </p>
    </div>
  )
}

// ── Main export ───────────────────────────────────────────────────────────────

export function DetailPanel({
  result, activeItem, selection,
  onToggleColumn, onToggleIndex, onToggleConstraint,
  onToggleAllColumns, onToggleAllIndexes, onToggleAllConstraints,
}: Props) {
  if (!activeItem) return <EmptyState />

  if (activeItem.type === 'table') {
    const td = result.Tables.find((t) => t.Name === activeItem.name)
    if (!td) return <EmptyState />
    return (
      <div className="flex flex-col flex-1 overflow-hidden">
        <div className="px-4 py-2.5 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 flex items-center gap-2 flex-shrink-0">
          <span className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">{td.Name}</span>
          <ChangeBadge change={td.Change} size="md" />
          {td.Change === 'modified' && (
            <span className="ml-auto text-[10px] text-gray-400 dark:text-gray-600">
              {td.Columns.length}c · {td.Indexes.length}i · {td.Constraints.length}k
            </span>
          )}
        </div>
        <TableDetail
          td={td} selection={selection}
          onToggleColumn={(c) => onToggleColumn(td.Name, c)}
          onToggleIndex={(i) => onToggleIndex(td.Name, i)}
          onToggleConstraint={(c) => onToggleConstraint(td.Name, c)}
          onToggleAllColumns={(v) => onToggleAllColumns(td.Name, v)}
          onToggleAllIndexes={(v) => onToggleAllIndexes(td.Name, v)}
          onToggleAllConstraints={(v) => onToggleAllConstraints(td.Name, v)}
        />
      </div>
    )
  }

  const vd = result.Views.find((v) => v.Name === activeItem.name)
  if (!vd) return <EmptyState />
  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <div className="px-4 py-2.5 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 flex items-center gap-2 flex-shrink-0">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-600">View</span>
        <span className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">{vd.Name}</span>
        <ChangeBadge change={vd.Change} size="md" />
      </div>
      <ViewDetail vd={vd} />
    </div>
  )
}
