import { useState, useRef, useCallback } from 'react'
import type { DiffResult, SelectionState, ChangeType } from '../types'
import { ChangeBadge, changeDotClass } from './ChangeBadge'

interface Props {
  result: DiffResult
  selection: SelectionState
  activeItem: { type: 'table' | 'view'; name: string } | null
  onSelectItem: (type: 'table' | 'view', name: string) => void
  onToggleTable: (name: string) => void
  onToggleView: (name: string) => void
  onSelectAll: (tableNames: string[], viewNames: string[]) => void
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

// ── Sub-components ────────────────────────────────────────────────────────────

interface RowProps {
  name: string
  change: ChangeType
  isActive: boolean
  isChecked: boolean
  isIndeterminate?: boolean
  onClick: () => void
  onCheck: () => void
  labels: Record<ChangeType, string>
}

function ItemRow({
  name,
  change,
  isActive,
  isChecked,
  isIndeterminate,
  onClick,
  onCheck,
  labels,
}: RowProps) {
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
      <ChangeBadge change={change} size="sm" labels={labels} />
    </div>
  )
}

function SectionHeader({
  title,
  visibleCount,
  totalCount,
  open,
  onToggle,
}: {
  title: string
  visibleCount: number
  totalCount: number
  open: boolean
  onToggle: () => void
}) {
  return (
    <button
      className="w-full px-3 py-1.5 flex items-center gap-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/60 transition-colors group"
      onClick={onToggle}
    >
      {/* Chevron */}
      <svg
        className={`w-3 h-3 flex-shrink-0 text-gray-400 dark:text-gray-600 transition-transform duration-150 ${
          open ? '' : '-rotate-90'
        }`}
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={2.5}
      >
        <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
      <span className="text-[10px] font-semibold text-gray-400 dark:text-gray-600 uppercase tracking-wider">
        {title}
      </span>
      <span className="text-[10px] text-gray-400 dark:text-gray-600">
        {visibleCount !== totalCount ? `${visibleCount}/${totalCount}` : totalCount}
      </span>
    </button>
  )
}

const FILTER_STYLES: Record<ChangeType, { on: string; off: string }> = {
  added: {
    on: 'bg-emerald-100 text-emerald-700 border-emerald-300 dark:bg-emerald-900/60 dark:text-emerald-400 dark:border-emerald-700',
    off: 'text-emerald-600 border-emerald-200 dark:text-emerald-600 dark:border-emerald-800/60 hover:bg-emerald-50 dark:hover:bg-emerald-950/40',
  },
  removed: {
    on: 'bg-red-100 text-red-700 border-red-300 dark:bg-red-900/60 dark:text-red-400 dark:border-red-700',
    off: 'text-red-600 border-red-200 dark:text-red-600 dark:border-red-800/60 hover:bg-red-50 dark:hover:bg-red-950/40',
  },
  modified: {
    on: 'bg-amber-100 text-amber-700 border-amber-300 dark:bg-amber-900/60 dark:text-amber-400 dark:border-amber-700',
    off: 'text-amber-600 border-amber-200 dark:text-amber-600 dark:border-amber-800/60 hover:bg-amber-50 dark:hover:bg-amber-950/40',
  },
}

function FilterChip({
  change,
  label,
  active,
  onClick,
}: {
  change: ChangeType
  label: string
  active: boolean
  onClick: () => void
}) {
  const s = FILTER_STYLES[change]
  return (
    <button
      onClick={onClick}
      className={`text-[10px] px-1.5 py-0.5 rounded border font-medium transition-colors truncate max-w-[80px] ${
        active ? s.on : s.off
      }`}
      title={label}
    >
      {label}
    </button>
  )
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

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
  const [tablesOpen, setTablesOpen] = useState(true)
  const [viewsOpen, setViewsOpen] = useState(true)
  const [activeFilters, setActiveFilters] = useState<Set<ChangeType>>(new Set())
  const [useNames, setUseNames] = useState(false)
  const [sidebarWidth, setSidebarWidth] = useState(256)
  const isDragging = useRef(false)
  const dragStartX = useRef(0)
  const dragStartWidth = useRef(0)

  const onDragMove = useCallback((e: MouseEvent) => {
    if (!isDragging.current) return
    const delta = e.clientX - dragStartX.current
    setSidebarWidth(Math.max(160, Math.min(480, dragStartWidth.current + delta)))
  }, [])

  const onDragEnd = useCallback(() => {
    isDragging.current = false
    document.removeEventListener('mousemove', onDragMove)
    document.removeEventListener('mouseup', onDragEnd)
  }, [onDragMove])

  const onDragStart = useCallback((e: React.MouseEvent) => {
    isDragging.current = true
    dragStartX.current = e.clientX
    dragStartWidth.current = sidebarWidth
    document.addEventListener('mousemove', onDragMove)
    document.addEventListener('mouseup', onDragEnd)
  }, [sidebarWidth, onDragMove, onDragEnd])

  const toggleFilter = (change: ChangeType) => {
    setActiveFilters((prev) => {
      const next = new Set(prev)
      next.has(change) ? next.delete(change) : next.add(change)
      return next
    })
  }

  // Labels: use actual DB names or generic terms
  const labels: Record<ChangeType, string> = {
    added: useNames ? `${result.TargetName} Only` : 'Target Only',
    removed: useNames ? `${result.SourceName} Only` : 'Source Only',
    modified: 'Modified',
  }

  // Apply change-type filter
  const filteredTables =
    activeFilters.size === 0
      ? result.Tables
      : result.Tables.filter((t) => activeFilters.has(t.Change))
  const filteredViews =
    activeFilters.size === 0
      ? result.Views
      : result.Views.filter((v) => activeFilters.has(v.Change))

  const total = result.Tables.length + result.Views.length
  const selected = selection.tables.size + selection.views.size
  const filteredTotal = filteredTables.length + filteredViews.length
  const filteredSelected =
    filteredTables.filter((t) => selection.tables.has(t.Name)).length +
    filteredViews.filter((v) => selection.views.has(v.Name)).length
  const isAllSelected = filteredTotal > 0 && filteredSelected === filteredTotal
  const isNoneSelected = selected === 0

  return (
    <aside
      className="relative flex-shrink-0 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col overflow-hidden"
      style={{ width: sidebarWidth }}
    >
      {/* Toolbar row: count + All/None + name toggle */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-800">
        <span className="text-xs text-gray-400 dark:text-gray-500">
          <span className="font-medium text-gray-600 dark:text-gray-300">{selected}</span>/{total}
        </span>
        <div className="flex items-center gap-1">
          <button
            onClick={() => onSelectAll(filteredTables.map((t) => t.Name), filteredViews.map((v) => v.Name))}
            className={`text-xs px-2 py-0.5 rounded transition-colors ${
              isAllSelected
                ? 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 font-medium'
                : 'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
            }`}
          >
            All
          </button>
          <button
            onClick={onDeselectAll}
            className={`text-xs px-2 py-0.5 rounded transition-colors ${
              isNoneSelected
                ? 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 font-medium'
                : 'text-gray-400 dark:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800'
            }`}
          >
            None
          </button>
          {/* Name replacement toggle */}
          <button
            onClick={() => setUseNames((v) => !v)}
            title={useNames ? 'Using database names — click to use generic labels' : 'Click to replace Source/Target with database names'}
            className={`text-[10px] ml-1 px-1.5 py-0.5 rounded border font-medium transition-colors ${
              useNames
                ? 'bg-gray-900 text-white border-gray-700 dark:bg-gray-100 dark:text-gray-900 dark:border-gray-300'
                : 'text-gray-400 dark:text-gray-500 border-gray-200 dark:border-gray-700 hover:border-gray-400 dark:hover:border-gray-500'
            }`}
          >
            Aa
          </button>
        </div>
      </div>

      {/* Filter chips row */}
      <div className="flex items-center gap-1 px-3 py-1.5 border-b border-gray-100 dark:border-gray-800/60 flex-wrap">
        {(['added', 'removed', 'modified'] as ChangeType[]).map((change) => (
          <FilterChip
            key={change}
            change={change}
            label={labels[change]}
            active={activeFilters.has(change)}
            onClick={() => toggleFilter(change)}
          />
        ))}
        {activeFilters.size > 0 && (
          <button
            onClick={() => setActiveFilters(new Set())}
            className="text-[10px] text-gray-400 dark:text-gray-600 hover:text-gray-600 dark:hover:text-gray-400 ml-0.5"
            title="Clear filters"
          >
            ✕
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto py-1">
        {/* Tables section */}
        {result.Tables.length > 0 && (
          <>
            <SectionHeader
              title="Tables"
              visibleCount={filteredTables.length}
              totalCount={result.Tables.length}
              open={tablesOpen}
              onToggle={() => setTablesOpen((v) => !v)}
            />
            {tablesOpen &&
              filteredTables.map((td) => (
                <ItemRow
                  key={td.Name}
                  name={td.Name}
                  change={td.Change}
                  isActive={activeItem?.type === 'table' && activeItem.name === td.Name}
                  isChecked={selection.tables.has(td.Name)}
                  isIndeterminate={isTablePartial(td.Name, result, selection)}
                  onClick={() => onSelectItem('table', td.Name)}
                  onCheck={() => onToggleTable(td.Name)}
                  labels={labels}
                />
              ))}
          </>
        )}

        {/* Views section */}
        {result.Views.length > 0 && (
          <>
            <div className={result.Tables.length > 0 ? 'mt-1' : ''}>
              <SectionHeader
                title="Views"
                visibleCount={filteredViews.length}
                totalCount={result.Views.length}
                open={viewsOpen}
                onToggle={() => setViewsOpen((v) => !v)}
              />
            </div>
            {viewsOpen &&
              filteredViews.map((vd) => (
                <ItemRow
                  key={vd.Name}
                  name={vd.Name}
                  change={vd.Change}
                  isActive={activeItem?.type === 'view' && activeItem.name === vd.Name}
                  isChecked={selection.views.has(vd.Name)}
                  onClick={() => onSelectItem('view', vd.Name)}
                  onCheck={() => onToggleView(vd.Name)}
                  labels={labels}
                />
              ))}
          </>
        )}
      </div>

      {/* Resize handle */}
      <div
        onMouseDown={onDragStart}
        className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors z-10"
      />
    </aside>
  )
}
