import { Moon, Sun } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef } from 'react'
import { Link } from 'react-router-dom'

import { useThemeMode } from './ThemeContext'

type MainNavSection = 'recommendation' | 'logs' | 'preferences'

type MainNavProps = {
  active?: MainNavSection
  tourTarget?: MainNavSection
}

let previousActiveSection: MainNavSection | undefined

export default function MainNav({ active, tourTarget }: MainNavProps) {
  const { isDarkMode, toggleTheme } = useThemeMode()
  const pillRef = useRef<HTMLSpanElement | null>(null)
  const tabRefs = useRef<Record<MainNavSection, HTMLAnchorElement | null>>({
    recommendation: null,
    logs: null,
    preferences: null,
  })

  const positionPill = useCallback((section: MainNavSection | undefined, withTransition: boolean) => {
    const pill = pillRef.current

    if (!pill || !section) {
      return
    }

    const activeTab = tabRefs.current[section]

    if (!activeTab) {
      return
    }

    if (!withTransition) {
      pill.style.transition = 'none'
    }

    pill.style.transform = `translateX(${activeTab.offsetLeft}px)`
    pill.style.width = `${activeTab.offsetWidth}px`

    if (!withTransition) {
      void pill.offsetWidth
      pill.style.transition = ''
    }
  }, [])

  useLayoutEffect(() => {
    if (!active) {
      return
    }

    const previousActiveTabExists =
      previousActiveSection &&
      previousActiveSection !== active &&
      tabRefs.current[previousActiveSection]

    if (previousActiveTabExists) {
      positionPill(previousActiveSection, false)
      const animationFrame = window.requestAnimationFrame(() => {
        positionPill(active, true)
        previousActiveSection = active
      })

      return () => window.cancelAnimationFrame(animationFrame)
    }

    positionPill(active, false)
    previousActiveSection = active
  }, [active, positionPill])

  useEffect(() => {
    const handleResize = () => positionPill(active, false)

    window.addEventListener('resize', handleResize)

    return () => window.removeEventListener('resize', handleResize)
  }, [active, positionPill])

  const getLinkClassName = (section: MainNavSection) => (
    ['t-tab', tourTarget === section ? 'ui-spotlight-target' : '']
      .filter(Boolean)
      .join(' ')
  )

  return (
    <nav className="ui-settings-nav ui-surface t-tabs" aria-label="Main navigation">
      <span ref={pillRef} className="t-tabs-pill" aria-hidden="true" />
      <Link
        ref={(node) => {
          tabRefs.current.recommendation = node
        }}
        to="/recommendation"
        aria-current={active === 'recommendation' ? 'page' : undefined}
        aria-selected={active === 'recommendation'}
        className={getLinkClassName('recommendation')}
        data-main-nav-tour={tourTarget === 'recommendation' ? 'recommendation' : undefined}
      >
        Recommendation
      </Link>
      <Link
        ref={(node) => {
          tabRefs.current.logs = node
        }}
        to="/meal-logs"
        aria-current={active === 'logs' ? 'page' : undefined}
        aria-selected={active === 'logs'}
        className={getLinkClassName('logs')}
        data-main-nav-tour={tourTarget === 'logs' ? 'logs' : undefined}
      >
        Logs
      </Link>
      <Link
        ref={(node) => {
          tabRefs.current.preferences = node
        }}
        to="/preferences"
        aria-current={active === 'preferences' ? 'page' : undefined}
        aria-selected={active === 'preferences'}
        className={getLinkClassName('preferences')}
        data-main-nav-tour={tourTarget === 'preferences' ? 'preferences' : undefined}
      >
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
