import type { DiffResult } from '../types'
import { computeStats } from '../utils/migration'

interface Props {
  result: DiffResult
}

function Dot({ color }: { color: string }) {
  return <span className={`inline-block w-1.5 h-1.5 rounded-full ${color}`} />
}

function Stat({ label, value, dot }: { label: string; value: number; dot: string }) {
  if (value === 0) return null
  return (
    <span className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
      <Dot color={dot} />
      <span className="font-medium text-gray-700 dark:text-gray-300">{value}</span>
      <span>{label}</span>
    </span>
  )
}

function Divider() {
  return <span className="text-gray-200 dark:text-gray-700 select-none">|</span>
}

export function StatsBar({ result }: Props) {
  const s = computeStats(result)
  const hasViews = s.viewsAdded + s.viewsRemoved + s.viewsModified > 0
  const hasChanges = s.columnsChanged + s.indexesChanged + s.constraintsChanged > 0

  return (
    <div className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 px-4 py-2 flex items-center gap-3 flex-wrap text-xs overflow-x-auto">
      <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500 mr-1">
        Tables
      </span>
      <Stat label="target only" value={s.tablesAdded} dot="bg-emerald-500" />
      <Stat label="source only" value={s.tablesRemoved} dot="bg-red-500" />
      <Stat label="modified" value={s.tablesModified} dot="bg-amber-500" />

      {hasViews && (
        <>
          <Divider />
          <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500 mr-1">
            Views
          </span>
          <Stat label="target only" value={s.viewsAdded} dot="bg-emerald-500" />
          <Stat label="source only" value={s.viewsRemoved} dot="bg-red-500" />
          <Stat label="modified" value={s.viewsModified} dot="bg-amber-500" />
        </>
      )}

      {hasChanges && (
        <>
          <Divider />
          {s.columnsChanged > 0 && (
            <span className="text-gray-500 dark:text-gray-400">
              <span className="font-medium text-gray-700 dark:text-gray-300">{s.columnsChanged}</span> col changes
            </span>
          )}
          {s.indexesChanged > 0 && (
            <span className="text-gray-500 dark:text-gray-400">
              <span className="font-medium text-gray-700 dark:text-gray-300">{s.indexesChanged}</span> idx changes
            </span>
          )}
          {s.constraintsChanged > 0 && (
            <span className="text-gray-500 dark:text-gray-400">
              <span className="font-medium text-gray-700 dark:text-gray-300">{s.constraintsChanged}</span> constraint changes
            </span>
          )}
        </>
      )}
    </div>
  )
}
