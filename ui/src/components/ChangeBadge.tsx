import type { ChangeType } from '../types'

interface Props {
  change: ChangeType
  size?: 'sm' | 'md'
  labels?: Record<ChangeType, string>
}

const DEFAULT_LABELS: Record<ChangeType, string> = {
  added: 'Target Only',
  removed: 'Source Only',
  modified: 'Modified',
}

// Subtle, muted change colors — functional but not flashy
const STYLES: Record<ChangeType, string> = {
  added:
    'bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400 dark:border-emerald-800',
  removed:
    'bg-red-50 text-red-700 border border-red-200 dark:bg-red-950 dark:text-red-400 dark:border-red-800',
  modified:
    'bg-amber-50 text-amber-700 border border-amber-200 dark:bg-amber-950 dark:text-amber-400 dark:border-amber-800',
}

export function ChangeBadge({ change, size = 'sm', labels }: Props) {
  const label = (labels ?? DEFAULT_LABELS)[change]
  const pad = size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-0.5'
  return (
    <span className={`rounded font-medium whitespace-nowrap ${pad} ${STYLES[change]}`}>
      {label}
    </span>
  )
}

export function changeDotClass(change: ChangeType): string {
  return {
    added: 'bg-emerald-500',
    removed: 'bg-red-500',
    modified: 'bg-amber-500',
  }[change]
}

export function changeRowClass(change: ChangeType): string {
  return {
    added:
      'bg-emerald-50 border-l-2 border-emerald-300 dark:bg-emerald-950/40 dark:border-emerald-700',
    removed:
      'bg-red-50 border-l-2 border-red-300 dark:bg-red-950/40 dark:border-red-700',
    modified:
      'bg-amber-50 border-l-2 border-amber-300 dark:bg-amber-950/40 dark:border-amber-700',
  }[change]
}
