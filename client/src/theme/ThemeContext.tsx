import React, { createContext, useContext, useEffect, useMemo, useState } from 'react'

type ThemeMode = 'light' | 'dark'

type ThemeContextValue = {
  isDarkMode: boolean
  themeMode: ThemeMode
  toggleTheme: () => void
}

const storageKey = 'ui-theme-mode'

const ThemeContext = createContext<ThemeContextValue | null>(null)

const readInitialTheme = (): ThemeMode => {
  if (typeof window === 'undefined') {
    return 'light'
  }

  return localStorage.getItem(storageKey) === 'dark' ? 'dark' : 'light'
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [themeMode, setThemeMode] = useState<ThemeMode>(readInitialTheme)
  const isDarkMode = themeMode === 'dark'

  useEffect(() => {
    localStorage.setItem(storageKey, themeMode)
    document.documentElement.dataset.theme = themeMode
  }, [themeMode])

  const value = useMemo<ThemeContextValue>(
    () => ({
      isDarkMode,
      themeMode,
      toggleTheme: () => setThemeMode((current) => (current === 'dark' ? 'light' : 'dark')),
    }),
    [isDarkMode, themeMode],
  )

  return (
    <ThemeContext.Provider value={value}>
      <BoxlessThemeShell isDarkMode={isDarkMode}>{children}</BoxlessThemeShell>
    </ThemeContext.Provider>
  )
}

function BoxlessThemeShell({ children, isDarkMode }: { children: React.ReactNode; isDarkMode: boolean }) {
  return (
    <div className={`ui-theme-shell ${isDarkMode ? 'ui-theme-dark' : 'ui-theme-light'}`}>
      {children}
    </div>
  )
}

export const useThemeMode = () => {
  const value = useContext(ThemeContext)

  if (!value) {
    throw new Error('useThemeMode must be used inside ThemeProvider')
  }

  return value
}
