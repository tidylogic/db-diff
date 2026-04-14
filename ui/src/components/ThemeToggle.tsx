import type { Theme } from '../hooks/useTheme'

interface Props {
  theme: Theme
  onChange: (t: Theme) => void
}

const OPTIONS: { value: Theme; icon: string; label: string }[] = [
  { value: 'light', icon: '☀', label: 'Light' },
  { value: 'system', icon: '⬤', label: 'System' },
  { value: 'dark', icon: '☽', label: 'Dark' },
]

export function ThemeToggle({ theme, onChange }: Props) {
  return (
    <div className="flex items-center border border-gray-200 dark:border-gray-700 rounded-md overflow-hidden">
      {OPTIONS.map((opt) => (
        <button
          key={opt.value}
          title={opt.label}
          onClick={() => onChange(opt.value)}
          className={`px-2 py-1 text-xs transition-colors ${
            theme === opt.value
              ? 'bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900'
              : 'text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
          }`}
        >
          {opt.icon}
        </button>
      ))}
    </div>
  )
}
