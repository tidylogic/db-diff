import { useCallback, useState } from 'react'
import type { DiffResult, SelectionState, Direction, Dialect } from './types'
import { buildInitialSelection } from './utils/migration'
import { useTheme } from './hooks/useTheme'
import { ThemeToggle } from './components/ThemeToggle'
import { StatsBar } from './components/StatsBar'
import { Sidebar } from './components/Sidebar'
import { DetailPanel } from './components/DetailPanel'
import { MigrationPanel } from './components/MigrationPanel'

// ── File parsing ──────────────────────────────────────────────────────────────

function parseJSON(text: string): DiffResult | string {
  try {
    const data = JSON.parse(text) as DiffResult
    if (!Array.isArray(data.Tables) && data.Tables != null) {
      return 'Invalid format: Tables field must be an array.'
    }
    data.Tables = (data.Tables ?? []).map((t) => ({
      ...t,
      Columns: t.Columns ?? [],
      Indexes: t.Indexes ?? [],
      Constraints: t.Constraints ?? [],
    }))
    data.Views = data.Views ?? []
    return data
  } catch {
    return 'Failed to parse JSON. Make sure this is a db-diff output file.'
  }
}

// ── Welcome / drop zone ───────────────────────────────────────────────────────

function WelcomeScreen({
  onLoad,
  theme,
  onThemeChange,
}: {
  onLoad: (r: DiffResult) => void
  theme: ReturnType<typeof useTheme>['theme']
  onThemeChange: ReturnType<typeof useTheme>['setTheme']
}) {
  const [error, setError] = useState<string | null>(null)
  const [dragging, setDragging] = useState(false)

  const handle = (file: File) => {
    setError(null)
    const reader = new FileReader()
    reader.onload = (e) => {
      const result = parseJSON(e.target?.result as string)
      if (typeof result === 'string') {
        setError(result)
      } else {
        onLoad(result)
      }
    }
    reader.readAsText(file)
  }

  const onFiles = (files: FileList | null) => {
    if (files?.[0]) handle(files[0])
  }

  return (
    <div className="min-h-screen bg-white dark:bg-gray-950 flex flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100 dark:border-gray-800">
        <span className="text-sm font-semibold text-gray-900 dark:text-gray-100 tracking-tight">
          db-diff
        </span>
        <ThemeToggle theme={theme} onChange={onThemeChange} />
      </div>

      {/* Center */}
      <div className="flex-1 flex flex-col items-center justify-center p-8">
        <div className="mb-8 text-center space-y-2">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Schema Diff Viewer
          </h1>
          <p className="text-sm text-gray-400 dark:text-gray-500 max-w-sm">
            Load a{' '}
            <code className="font-mono text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-1 py-0.5 rounded">
              db-diff compare --output json
            </code>{' '}
            result to inspect changes and generate migration SQL.
          </p>
        </div>

        <label
          className={`w-full max-w-md border border-dashed rounded-lg p-10 flex flex-col items-center gap-3 cursor-pointer transition-colors ${
            dragging
              ? 'border-gray-400 dark:border-gray-500 bg-gray-50 dark:bg-gray-900'
              : 'border-gray-200 dark:border-gray-800 hover:border-gray-400 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-900/60'
          }`}
          onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
          onDragLeave={() => setDragging(false)}
          onDrop={(e) => { e.preventDefault(); setDragging(false); onFiles(e.dataTransfer.files) }}
        >
          <input
            type="file"
            accept=".json,application/json"
            className="sr-only"
            onChange={(e) => onFiles(e.target.files)}
          />
          <svg className="w-8 h-8 text-gray-300 dark:text-gray-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <div className="text-center">
            <p className="text-sm text-gray-600 dark:text-gray-400">Drop JSON file here</p>
            <p className="text-xs text-gray-400 dark:text-gray-600 mt-0.5">or click to browse</p>
          </div>
        </label>

        {error && (
          <p className="mt-4 text-xs text-red-600 dark:text-red-400 max-w-md text-center">
            {error}
          </p>
        )}

        <p className="mt-8 text-xs text-gray-300 dark:text-gray-700 text-center">
          <code className="font-mono">
            db-diff compare --source ... --target ... --output json {'>'} diff.json
          </code>
        </p>
      </div>
    </div>
  )
}

// ── Header ────────────────────────────────────────────────────────────────────

function Header({
  result,
  onReset,
  onLoad,
  theme,
  onThemeChange,
}: {
  result: DiffResult
  onReset: () => void
  onLoad: (r: DiffResult) => void
  theme: ReturnType<typeof useTheme>['theme']
  onThemeChange: ReturnType<typeof useTheme>['setTheme']
}) {
  const handleFile = (file: File) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      const r = parseJSON(e.target?.result as string)
      if (typeof r !== 'string') onLoad(r)
    }
    reader.readAsText(file)
  }

  return (
    <header className="h-11 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 flex items-center gap-3 px-4 flex-shrink-0">
      <button
        onClick={onReset}
        className="text-sm font-semibold text-gray-900 dark:text-gray-100 hover:text-gray-600 dark:hover:text-gray-400 transition-colors"
      >
        db-diff
      </button>

      <span className="text-gray-200 dark:text-gray-700">/</span>

      <div className="flex items-center gap-1.5 text-xs">
        <span className="text-gray-700 dark:text-gray-300 font-medium">{result.SourceName}</span>
        <span className="text-gray-300 dark:text-gray-700">↔</span>
        <span className="text-gray-700 dark:text-gray-300 font-medium">{result.TargetName}</span>
      </div>

      {result.Identical && (
        <span className="text-[10px] border border-gray-200 dark:border-gray-700 text-gray-400 dark:text-gray-600 rounded px-1.5 py-0.5">
          identical
        </span>
      )}

      <div className="ml-auto flex items-center gap-2">
        <ThemeToggle theme={theme} onChange={onThemeChange} />

        <label className="flex items-center gap-1 text-xs px-2.5 py-1 rounded border border-gray-200 dark:border-gray-700 text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors cursor-pointer">
          Load JSON
          <input
            type="file"
            accept=".json,application/json"
            className="sr-only"
            onChange={(e) => { if (e.target.files?.[0]) handleFile(e.target.files[0]) }}
          />
        </label>
      </div>
    </header>
  )
}

// ── Main App ──────────────────────────────────────────────────────────────────

export default function App() {
  const { theme, setTheme } = useTheme()

  const [diffResult, setDiffResult] = useState<DiffResult | null>(null)
  const [selection, setSelection] = useState<SelectionState>({
    tables: new Set(),
    columns: {},
    indexes: {},
    constraints: {},
    views: new Set(),
  })
  const [activeItem, setActiveItem] = useState<{
    type: 'table' | 'view'
    name: string
  } | null>(null)
  const [direction, setDirection] = useState<Direction>('source_to_target')
  const [dialect, setDialect] = useState<Dialect>('mysql')

  const handleLoad = useCallback((result: DiffResult) => {
    setDiffResult(result)
    setSelection(buildInitialSelection(result))
    setActiveItem(
      result.Tables.length > 0
        ? { type: 'table', name: result.Tables[0].Name }
        : result.Views.length > 0
          ? { type: 'view', name: result.Views[0].Name }
          : null,
    )
  }, [])

  const handleReset = useCallback(() => {
    setDiffResult(null)
    setActiveItem(null)
  }, [])

  // ── Selection ──────────────────────────────────────────────────────────────

  const toggleTable = useCallback((name: string) => {
    setSelection((p) => {
      const tables = new Set(p.tables)
      tables.has(name) ? tables.delete(name) : tables.add(name)
      return { ...p, tables }
    })
  }, [])

  const toggleView = useCallback((name: string) => {
    setSelection((p) => {
      const views = new Set(p.views)
      views.has(name) ? views.delete(name) : views.add(name)
      return { ...p, views }
    })
  }, [])

  const selectAll = useCallback((tableNames: string[], viewNames: string[]) => {
    if (!diffResult) return
    setSelection((prev) => {
      const tables = new Set(prev.tables)
      const views = new Set(prev.views)
      const columns = { ...prev.columns }
      const indexes = { ...prev.indexes }
      const constraints = { ...prev.constraints }

      for (const name of tableNames) {
        tables.add(name)
        const td = diffResult.Tables.find((t) => t.Name === name)
        if (td?.Change === 'modified') {
          columns[name] = new Set(td.Columns.map((c) => c.Name))
          indexes[name] = new Set(td.Indexes.map((i) => i.Name))
          constraints[name] = new Set(td.Constraints.map((c) => c.Name))
        }
      }
      for (const name of viewNames) {
        views.add(name)
      }

      return { tables, columns, indexes, constraints, views }
    })
  }, [diffResult])

  const deselectAll = useCallback(() => {
    setSelection({ tables: new Set(), columns: {}, indexes: {}, constraints: {}, views: new Set() })
  }, [])

  const toggleColumn = useCallback((table: string, col: string) => {
    setSelection((p) => {
      const set = new Set(p.columns[table] ?? [])
      set.has(col) ? set.delete(col) : set.add(col)
      const tables = new Set(p.tables)
      if (set.size > 0) tables.add(table)
      return { ...p, tables, columns: { ...p.columns, [table]: set } }
    })
  }, [])

  const toggleIndex = useCallback((table: string, idx: string) => {
    setSelection((p) => {
      const set = new Set(p.indexes[table] ?? [])
      set.has(idx) ? set.delete(idx) : set.add(idx)
      const tables = new Set(p.tables)
      if (set.size > 0) tables.add(table)
      return { ...p, tables, indexes: { ...p.indexes, [table]: set } }
    })
  }, [])

  const toggleConstraint = useCallback((table: string, con: string) => {
    setSelection((p) => {
      const set = new Set(p.constraints[table] ?? [])
      set.has(con) ? set.delete(con) : set.add(con)
      const tables = new Set(p.tables)
      if (set.size > 0) tables.add(table)
      return { ...p, tables, constraints: { ...p.constraints, [table]: set } }
    })
  }, [])

  const toggleAllColumns = useCallback((table: string, checked: boolean) => {
    if (!diffResult) return
    const td = diffResult.Tables.find((t) => t.Name === table)
    if (!td) return
    setSelection((p) => {
      const set = checked ? new Set(td.Columns.map((c) => c.Name)) : new Set<string>()
      const tables = new Set(p.tables)
      if (checked) tables.add(table)
      return { ...p, tables, columns: { ...p.columns, [table]: set } }
    })
  }, [diffResult])

  const toggleAllIndexes = useCallback((table: string, checked: boolean) => {
    if (!diffResult) return
    const td = diffResult.Tables.find((t) => t.Name === table)
    if (!td) return
    setSelection((p) => {
      const set = checked ? new Set(td.Indexes.map((i) => i.Name)) : new Set<string>()
      const tables = new Set(p.tables)
      if (checked) tables.add(table)
      return { ...p, tables, indexes: { ...p.indexes, [table]: set } }
    })
  }, [diffResult])

  const toggleAllConstraints = useCallback((table: string, checked: boolean) => {
    if (!diffResult) return
    const td = diffResult.Tables.find((t) => t.Name === table)
    if (!td) return
    setSelection((p) => {
      const set = checked ? new Set(td.Constraints.map((c) => c.Name)) : new Set<string>()
      const tables = new Set(p.tables)
      if (checked) tables.add(table)
      return { ...p, tables, constraints: { ...p.constraints, [table]: set } }
    })
  }, [diffResult])

  // ── Render ─────────────────────────────────────────────────────────────────

  if (!diffResult) {
    return <WelcomeScreen onLoad={handleLoad} theme={theme} onThemeChange={setTheme} />
  }

  return (
    <div className="h-screen flex flex-col overflow-hidden bg-white dark:bg-gray-950">
      <Header
        result={diffResult}
        onReset={handleReset}
        onLoad={handleLoad}
        theme={theme}
        onThemeChange={setTheme}
      />
      <StatsBar result={diffResult} />

      <div className="flex flex-1 overflow-hidden">
        <Sidebar
          result={diffResult}
          selection={selection}
          activeItem={activeItem}
          onSelectItem={(type, name) => setActiveItem({ type, name })}
          onToggleTable={toggleTable}
          onToggleView={toggleView}
          onSelectAll={selectAll}
          onDeselectAll={deselectAll}
        />

        <main className="flex-1 flex flex-col overflow-hidden bg-white dark:bg-gray-950">
          <DetailPanel
            result={diffResult}
            activeItem={activeItem}
            selection={selection}
            onToggleColumn={toggleColumn}
            onToggleIndex={toggleIndex}
            onToggleConstraint={toggleConstraint}
            onToggleAllColumns={toggleAllColumns}
            onToggleAllIndexes={toggleAllIndexes}
            onToggleAllConstraints={toggleAllConstraints}
          />

          <MigrationPanel
            result={diffResult}
            selection={selection}
            direction={direction}
            dialect={dialect}
            onDirectionChange={setDirection}
            onDialectChange={setDialect}
          />
        </main>
      </div>
    </div>
  )
}
