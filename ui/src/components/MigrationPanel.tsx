import { useMemo, useRef, useState } from 'react'
import type { DiffResult, SelectionState, Direction, Dialect } from '../types'
import { generateSQL } from '../utils/migration'

interface Props {
  result: DiffResult
  selection: SelectionState
  direction: Direction
  dialect: Dialect
  onDirectionChange: (d: Direction) => void
  onDialectChange: (d: Dialect) => void
}

function Btn({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={`px-2.5 py-1 text-xs rounded transition-colors ${
        active
          ? 'bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900'
          : 'text-gray-500 dark:text-gray-400 border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800'
      }`}
    >
      {children}
    </button>
  )
}

export function MigrationPanel({
  result,
  selection,
  direction,
  dialect,
  onDirectionChange,
  onDialectChange,
}: Props) {
  const [copied, setCopied] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const sql = useMemo(
    () => generateSQL(result, selection, direction, dialect),
    [result, selection, direction, dialect],
  )

  const isEmpty = !sql.split('\n').some((l) => l.trim() && !l.startsWith('--'))

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(sql)
    } catch {
      textareaRef.current?.select()
      document.execCommand('copy')
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleDownload = () => {
    const blob = new Blob([sql], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const src = direction === 'source_to_target' ? result.SourceName : result.TargetName
    const tgt = direction === 'source_to_target' ? result.TargetName : result.SourceName
    a.download = `migrate_${src}_to_${tgt}.sql`.replace(/[^a-zA-Z0-9_.-]/g, '_')
    a.click()
    URL.revokeObjectURL(url)
  }

  const srcLabel = direction === 'source_to_target' ? result.SourceName : result.TargetName
  const tgtLabel = direction === 'source_to_target' ? result.TargetName : result.SourceName

  return (
    <div
      className="flex flex-col border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
      style={{ height: '320px' }}
    >
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-4 py-2 border-b border-gray-200 dark:border-gray-800 flex-shrink-0 flex-wrap gap-y-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-600">
          Migration SQL
        </span>

        <div className="flex items-center gap-1.5">
          <span className="text-[10px] text-gray-400 dark:text-gray-600">Direction</span>
          <div className="flex gap-1">
            <Btn active={direction === 'source_to_target'} onClick={() => onDirectionChange('source_to_target')}>
              {result.SourceName} → {result.TargetName}
            </Btn>
            <Btn active={direction === 'target_to_source'} onClick={() => onDirectionChange('target_to_source')}>
              {result.TargetName} → {result.SourceName}
            </Btn>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          <span className="text-[10px] text-gray-400 dark:text-gray-600">Dialect</span>
          <div className="flex gap-1">
            <Btn active={dialect === 'mysql'} onClick={() => onDialectChange('mysql')}>MySQL</Btn>
            <Btn active={dialect === 'postgres'} onClick={() => onDialectChange('postgres')}>PostgreSQL</Btn>
          </div>
        </div>

        <div className="ml-auto flex gap-1.5">
          <button
            onClick={handleCopy}
            className="flex items-center gap-1 px-2.5 py-1 text-xs border border-gray-200 dark:border-gray-700 rounded text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          >
            {copied ? '✓ Copied' : 'Copy'}
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-1 px-2.5 py-1 text-xs bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 rounded hover:bg-gray-700 dark:hover:bg-gray-300 transition-colors"
          >
            Download .sql
          </button>
        </div>
      </div>

      {/* Direction hint */}
      <div className="px-4 py-1 border-b border-gray-100 dark:border-gray-800/60 flex-shrink-0">
        <span className="text-[10px] text-gray-400 dark:text-gray-600">
          Make <span className="text-gray-600 dark:text-gray-400">{srcLabel}</span> match <span className="text-gray-600 dark:text-gray-400">{tgtLabel}</span>
        </span>
      </div>

      {/* SQL output */}
      <div className="flex-1 relative overflow-hidden">
        {isEmpty ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <p className="text-xs text-gray-300 dark:text-gray-700">
              No items selected — check tables or columns in the sidebar.
            </p>
          </div>
        ) : (
          <textarea
            ref={textareaRef}
            readOnly
            value={sql}
            className="absolute inset-0 w-full h-full resize-none border-0 bg-gray-950 text-gray-300 text-xs font-mono p-4 focus:outline-none leading-relaxed"
            spellCheck={false}
          />
        )}
      </div>
    </div>
  )
}
