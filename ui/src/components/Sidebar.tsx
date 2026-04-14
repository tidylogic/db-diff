import type { DiffResult, SelectionState, ChangeType } from '../types'
import { ChangeBadge, changeDotClass } from './ChangeBadge'

interface Props {
  result: DiffResult
  selection: SelectionState
  activeItem: { type: 'table' | 'view'; name: string } | null
  onSelectItem: (type: 'table' | 'view', name: string) => void
  onToggleTable: (name: string) => void
  onToggleView: (name: string) => void
  onSelectAll: () => void
  onDeselectAll: () => void
}

function isTablePartial(
  tableName: string,
  result: DiffResult,
  selection: SelectionState,
): boolean {
  if (!selection.tables.has(tableName)) return false
  const td = result.Tables.find((t) => t.Name === tableName)
  if (!td || td.Change !== 'modified') return false

  const selCols = selection.columns[tableName] ?? new Set<string>()
  const selIdxs = selection.indexes[tableName] ?? new Set<string>()
  const selConsts = selection.constraints[tableName] ?? new Set<string>()

  const allSelected =
    td.Columns.every((c) => selCols.has(c.Name)) &&
    td.Indexes.every((i) => selIdxs.has(i.Name)) &&
    td.Constraints.every((c) => selConsts.has(c.Name))

  const noneSelected =
    td.Columns.every((c) => !selCols.has(c.Name)) &&
    td.Indexes.every((i) => !selIdxs.has(i.Name)) &&
    td.Constraints.every((c) => !selConsts.has(c.Name))

  return !allSelected && !noneSelected
}

interface RowProps {
  name: string
  change: ChangeType
  isActive: boolean
  isChecked: boolean
  isIndeterminate?: boolean
  onClick: () => void
  onCheck: () => void
}

function ItemRow({ name, change, isActive, isChecked, isIndeterminate, onClick, onCheck }: RowProps) {
  return (
    <div
      className={`flex items-center gap-2 px-3 py-1.5 cursor-pointer transition-colors select-none ${
        isActive
          ? 'bg-gray-100 dark:bg-gray-800'
          : 'hover:bg-gray-50 dark:hover:bg-gray-800/60'
      }`}
      onClick={onClick}
    >
      <input
        type="checkbox"
        checked={isChecked}
        ref={(el) => { if (el) el.indeterminate = isIndeterminate ?? false }}
        onChange={(e) => { e.stopPropagation(); onCheck() }}
        onClick={(e) => e.stopPropagation()}
        className="w-3.5 h-3.5 flex-shrink-0 cursor-pointer accent-gray-700 dark:accent-gray-300"
      />
      <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${changeDotClass(change)}`} />
      <span
        className={`flex-1 text-xs truncate ${
          isActive
            ? 'text-gray-900 dark:text-gray-100 font-medium'
            : 'text-gray-600 dark:text-gray-400'
        }`}
        title={name}
      >
        {name}
      </span>
      <ChangeBadge change={change} size="sm" />
    </div>
  )
}

export function Sidebar({
  result,
  selection,
  activeItem,
  onSelectItem,
  onToggleTable,
  onToggleView,
  onSelectAll,
  onDeselectAll,
}: Props) {
  const total = result.Tables.length + result.Views.length
  const selected = selection.tables.size + selection.views.size

  return (
    <aside className="w-64 flex-shrink-0 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-800">
        <span className="text-xs text-gray-400 dark:text-gray-500">
          <span className="font-medium text-gray-600 dark:text-gray-300">{selected}</span>/{total}
        </span>
        <div className="flex gap-0.5">
          <button
            onClick={onSelectAll}
            className="text-xs px-2 py-0.5 rounded text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          >
            All
          </button>
          <button
            onClick={onDeselectAll}
            className="text-xs px-2 py-0.5 rounded text-gray-400 dark:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          >
            None
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto py-1">
        {result.Tables.length > 0 && (
          <>
            <div className="px-3 py-1.5 flex items-center gap-2">
              <span className="text-[10px] font-semibold text-gray-400 dark:text-gray-600 uppercase tracking-wider">
                Tables
              </span>
              <span className="text-[10px] text-gray-400 dark:text-gray-600">
                {result.Tables.length}
              </span>
            </div>
            {result.Tables.map((td) => (
              <ItemRow
                key={td.Name}
                name={td.Name}
                change={td.Change}
                isActive={activeItem?.type === 'table' && activeItem.name === td.Name}
                isChecked={selection.tables.has(td.Name)}
                isIndeterminate={isTablePartial(td.Name, result, selection)}
                onClick={() => onSelectItem('table', td.Name)}
                onCheck={() => onToggleTable(td.Name)}
              />
            ))}
          </>
        )}

        {result.Views.length > 0 && (
          <>
            <div className="px-3 py-1.5 mt-2 flex items-center gap-2">
              <span className="text-[10px] font-semibold text-gray-400 dark:text-gray-600 uppercase tracking-wider">
                Views
              </span>
              <span className="text-[10px] text-gray-400 dark:text-gray-600">
                {result.Views.length}
              </span>
            </div>
            {result.Views.map((vd) => (
              <ItemRow
                key={vd.Name}
                name={vd.Name}
                change={vd.Change}
                isActive={activeItem?.type === 'view' && activeItem.name === vd.Name}
                isChecked={selection.views.has(vd.Name)}
                onClick={() => onSelectItem('view', vd.Name)}
                onCheck={() => onToggleView(vd.Name)}
              />
            ))}
          </>
        )}
      </div>
    </aside>
  )
}
