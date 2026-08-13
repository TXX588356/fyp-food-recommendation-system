import { Moon, Sun } from 'lucide-react'
import { Link } from 'react-router-dom'

import { useThemeMode } from './ThemeContext'

type MainNavSection = 'recommendation' | 'logs' | 'preferences'

type MainNavProps = {
  active?: MainNavSection
}

export default function MainNav({ active }: MainNavProps) {
  const { isDarkMode, toggleTheme } = useThemeMode()

  return (
    <nav className="ui-settings-nav ui-surface" aria-label="Main navigation">
      <Link to="/recommendation" aria-current={active === 'recommendation' ? 'page' : undefined}>
        Recommendation
      </Link>
      <Link to="/meal-logs" aria-current={active === 'logs' ? 'page' : undefined}>
        Logs
      </Link>
      <Link to="/preferences" aria-current={active === 'preferences' ? 'page' : undefined}>
        Preferences
      </Link>
      <button
        className="ui-theme-toggle"
        type="button"
        aria-label={isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'}
        aria-pressed={isDarkMode}
        onClick={toggleTheme}
      >
        {isDarkMode ? (
          <Sun size={19} strokeWidth={2.35} aria-hidden="true" />
        ) : (
          <Moon size={19} strokeWidth={2.35} aria-hidden="true" />
        )}
      </button>
    </nav>
  )
}
