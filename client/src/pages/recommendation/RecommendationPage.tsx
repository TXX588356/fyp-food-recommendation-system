import {
  Accordion,
  Alert,
  Badge,
  Box,
  Button,
  SegmentedControl,
  Text,
  Title,
} from '@mantine/core'
import axios from 'axios'
import React, { useEffect, useMemo, useState } from 'react'
import { Moon, Sun } from 'lucide-react'

import { useAuth } from '@/auth/useAuth'
import './RecommendationPage.css'
import '@/App.css'
import type { LoggableMeal } from '@/pages/mealLog/mealLogTypes'
import LogMealModal from '@/pages/mealLog/LogMealModal'
import { FiCheck } from 'react-icons/fi'
import { Link, useNavigate } from 'react-router-dom'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

type MealCategory = 'breakfast' | 'lunch' | 'dinner' | 'snack'

type PriceRange = {
  min: number
  max: number
}

type GeneratedMeal = {
  name: string
  alternative_search_terms: string[]
  estimated_price_range: PriceRange
  sodium_level: string
  sugar_level: string
  purine_risk: string
  health_flags: Record<string, string>
}

type FoodSearchResult = {
  id: string
  name: string
  tags: string[]
  calories: number
  fat_g: number
  protein_g: number
  carbs_g: number
  image_url?: string
}

type MatchedMealCandidate = {
  generated_meal: GeneratedMeal
  food: FoodSearchResult
  matched_query: string
}

type FilteredMealCandidate = {
  candidate: MatchedMealCandidate
  reason: string
}

type GenerateRecommendationsResponse = {
  candidates: MatchedMealCandidate[]
  filtered_out: FilteredMealCandidate[]
  filtering_applied: boolean
}

type CategoryState<T> = Record<MealCategory, T>

type VisibleRecommendationItem = {
  candidate: MatchedMealCandidate
  filteredReason: string | null
}

type PersistedRecommendationState = {
  generatedDate: string
  candidatesByCategory: CategoryState<MatchedMealCandidate[]>
  filteredOutByCategory: CategoryState<FilteredMealCandidate[]>
  generatedByCategory: CategoryState<boolean>
}

const mealCategoryOptions: Array<{ value: MealCategory; label: string;}> = [
  { value: 'breakfast', label: 'Breakfast'},
  { value: 'lunch', label: 'Lunch'},
  { value: 'dinner', label: 'Dinner'},
  { value: 'snack', label: 'Snack'},
]

const emptyCandidates: CategoryState<MatchedMealCandidate[]> = {
  breakfast: [],
  lunch: [],
  dinner: [],
  snack: [],
}

const emptyFilteredOut: CategoryState<FilteredMealCandidate[]> = {
  breakfast: [],
  lunch: [],
  dinner: [],
  snack: [],
}

const emptyLoading: CategoryState<boolean> = {
  breakfast: false,
  lunch: false,
  dinner: false,
  snack: false,
}

const emptyGenerated: CategoryState<boolean> = {
  breakfast: false,
  lunch: false,
  dinner: false,
  snack: false,
}

const emptyErrors: CategoryState<string | null> = {
  breakfast: null,
  lunch: null,
  dinner: null,
  snack: null,
}

const formatRM = (value: number) => `RM ${value.toFixed(2)}`

const buildRecommendationStorageKey = (userKey: string | undefined) => {
  return `recommendation:${userKey ?? 'anonymous'}`
}

const createEmptyPersistedRecommendations = 
(): PersistedRecommendationState => ({
  generatedDate: getLocalDateKey(),
  candidatesByCategory: {
    breakfast: [],
    lunch: [],
    dinner: [],
    snack: [],
  },
  filteredOutByCategory: {
    breakfast: [],
    lunch: [],
    dinner: [],
    snack: [],
  },
  generatedByCategory: {
    breakfast: false,
    lunch: false,
    dinner: false,
    snack: false,
  },
})

const loadPersistedRecommendations = (storageKey: string): PersistedRecommendationState => {
  const emptyState = createEmptyPersistedRecommendations()

  try {
    const rawValue = localStorage.getItem(storageKey)

    if (!rawValue) {
      return emptyState
    }

    const parsedValue = JSON.parse(rawValue) as Partial<PersistedRecommendationState>

    if (parsedValue.generatedDate !== getLocalDateKey()) {
      localStorage.removeItem(storageKey)
      return emptyState
    }

    return {
      generatedDate: parsedValue.generatedDate,
      candidatesByCategory: {
        ...emptyCandidates,
        ...parsedValue.candidatesByCategory,
      },
      filteredOutByCategory: {
        ...emptyFilteredOut,
        ...parsedValue.filteredOutByCategory,
      },
      generatedByCategory: {
        ...emptyGenerated,
        ...parsedValue.generatedByCategory,
      },
    }
  } catch {
    localStorage.removeItem(storageKey)
    return emptyState
  }
}

const savePersistedRecommendations = (
  storageKey: string,
  nextState: PersistedRecommendationState,
) => {
  localStorage.setItem(storageKey, JSON.stringify(nextState))
}

const getLocalDateKey = () => {
  const today = new Date()
  
  const year = today.getFullYear()
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')

  return `${year}-${month}-${day}`
}

type IconName = 'refresh' | 'bowl' | 'fork' | 'sparkle' | 'warning'

function RecommendationIcon({ name, size = 20 }: { name: IconName; size?: number }) {
  const commonProps = {
    width: size,
    height: size,
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 2,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  }

  if (name === 'refresh') {
    return (
      <svg {...commonProps}>
        <path d="M20 12a8 8 0 0 1-13.7 5.6" />
        <path d="M4 12A8 8 0 0 1 17.7 6.4" />
        <path d="M18 3v4h-4" />
        <path d="M6 21v-4h4" />
      </svg>
    )
  }

  if (name === 'bowl') {
    return (
      <svg {...commonProps}>
        <path d="M4 11h16" />
        <path d="M6 11c0 4 2.7 7 6 7s6-3 6-7" />
        <path d="M8 20h8" />
        <path d="M9 7c0-1 1-1 1-2s-1-1-1-2" />
        <path d="M14 7c0-1 1-1 1-2s-1-1-1-2" />
      </svg>
    )
  }

  if (name === 'fork') {
    return (
      <svg {...commonProps}>
        <path d="M7 3v8" />
        <path d="M5 3v4" />
        <path d="M9 3v4" />
        <path d="M7 11v10" />
        <path d="M15 3v18" />
        <path d="M15 3c3 2 4 5 1 8" />
      </svg>
    )
  }

  if (name === 'warning') {
    return (
      <svg {...commonProps}>
        <path d="M12 9v4" />
        <path d="M12 17h.01" />
        <path d="M10.3 4.4 2.7 18a2 2 0 0 0 1.7 3h15.2a2 2 0 0 0 1.7-3L13.7 4.4a2 2 0 0 0-3.4 0Z" />
      </svg>
    )
  }

  return (
    <svg {...commonProps}>
      <path d="M12 3l1.8 5.1L19 10l-5.2 1.9L12 17l-1.8-5.1L5 10l5.2-1.9L12 3Z" />
      <path d="M19 15l.8 2.2L22 18l-2.2.8L19 21l-.8-2.2L16 18l2.2-.8L19 15Z" />
    </svg>
  )
}

export default function RecommendationPage() {
  const { user } = useAuth()
  const userKey = user?.id ?? user?.email

  const recommendationStorageKey = useMemo(
    () => buildRecommendationStorageKey(userKey),
    [userKey],
  )

  const persistedRecommendations = useMemo(
    () => loadPersistedRecommendations(recommendationStorageKey),
    [recommendationStorageKey],
  )

  const [activeCategory, setActiveCategory] = useState<MealCategory | null>(null)
  const [candidatesByCategory, setCandidatesByCategory] = useState<CategoryState<MatchedMealCandidate[]>>(
    persistedRecommendations.candidatesByCategory,
  )
  const [loadingByCategory, setLoadingByCategory] = useState<CategoryState<boolean>>(emptyLoading)
  const [generatedByCategory, setGeneratedByCategory] = useState<CategoryState<boolean>>(
    persistedRecommendations.generatedByCategory,
  )

  const [filteredOutByCategory, setFilteredOutByCategory] = useState<CategoryState<FilteredMealCandidate[]>>(
    persistedRecommendations.filteredOutByCategory,
  )

  const [displayMode, setDisplayMode] = useState<'filtered' | 'all'>('filtered')

  const [errorsByCategory, setErrorsByCategory] = useState<CategoryState<string | null>>(emptyErrors)

  const [mealToLog, setMealToLog] = useState<LoggableMeal | null>(null)
  const [successMeassage, setSuccessMessage] = useState<string | null>(null)
  const [isDarkMode, setIsDarkMode] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    if (!successMeassage) {
      return 
    }

    const dismissTimer = window.setTimeout(() => {
      setSuccessMessage(null)
    }, 4000)

    return () => window.clearTimeout(dismissTimer)
  }, [successMeassage])

  const openLogModal = (candidate: MatchedMealCandidate) => {
    setMealToLog({
      source: 'prebuilt',
      mealId: candidate.food.id,
      name: candidate.food.name,
      calories: candidate.food.calories,
    })
  }

  const token = localStorage.getItem('token')

  const persistRecommendationCategory = (
    mealCategory: MealCategory,
    candidates: MatchedMealCandidate[],
    filteredOut: FilteredMealCandidate[],
  ) => {
    const persistedRecommendations = loadPersistedRecommendations(recommendationStorageKey)

    // Create new updated copies of recommendation state for one meal category
    const nextCandidatesByCategory = {
      ...persistedRecommendations.candidatesByCategory,
      [mealCategory]: candidates,
    }
    const nextGeneratedByCategory = {
      ...persistedRecommendations.generatedByCategory,
      [mealCategory]: true,
    }
    const nextFilteredOutByCategory = {
      ...persistedRecommendations.filteredOutByCategory,
      [mealCategory]: filteredOut,
    }

    savePersistedRecommendations(recommendationStorageKey, {
      generatedDate: getLocalDateKey(),
      candidatesByCategory: nextCandidatesByCategory,
      filteredOutByCategory: nextFilteredOutByCategory,
      generatedByCategory: nextGeneratedByCategory,
    })

    setCandidatesByCategory((current) => ({
      ...current,
      [mealCategory]: candidates,
    }))
    setGeneratedByCategory((current) => ({
      ...current,
      [mealCategory]: true,
    }))
    setFilteredOutByCategory((current) => ({
      ...current,
      [mealCategory]: filteredOut,
    }))
  }

  const generateRecommendations = async (mealCategory: MealCategory, force = false) => {
    if (loadingByCategory[mealCategory] || (generatedByCategory[mealCategory] && !force)) {
      return
    }

    setLoadingByCategory((current) => ({ ...current, [mealCategory]: true }))
    setErrorsByCategory((current) => ({ ...current, [mealCategory]: null }))

    try {
      const response = await axios.post<GenerateRecommendationsResponse>(
        `${API_BASE_URL}/recommendations`,
        {
          mealCategory,
          currentMonthSpent: 0,
          perMealBudget: 0,
        },
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      persistRecommendationCategory(mealCategory, response.data.candidates ?? [], response.data.filtered_out ?? [])
    } catch (error) {
      console.error(`Failed to generate ${mealCategory} recommendations`, error)

      let message = 'Could not generate recommendations right now.'
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        message = 'Your session has expired. Please log in again.'
      } else if (axios.isAxiosError(error) && error.response?.status === 400) {
        message = error.response.data?.error ?? 'Please check the recommendation request.'
      }

      setErrorsByCategory((current) => ({ ...current, [mealCategory]: message }))
      setGeneratedByCategory((current) => ({ ...current, [mealCategory]: true }))
    } finally {
      setLoadingByCategory((current) => ({ ...current, [mealCategory]: false }))
    }
  }

  const handleCategoryChange = (value: string | null) => {
    const nextCategory = value as MealCategory | null
    setActiveCategory(nextCategory)

    if (nextCategory) {
      void generateRecommendations(nextCategory)
    }
  }

  const renderCandidateCard = (mealCategory: MealCategory, item: VisibleRecommendationItem, index: number) => {
    const { candidate, filteredReason } = item

    const openDetailPage = () => {
      if (filteredReason) { return }

      navigate(`/recommendation/${mealCategory}/${candidate.food.id}`)
    }

    const handleCardKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        openDetailPage()
      }
    }

    return (
    <Box 
      className={`ui-meal-card ui-card ${filteredReason ? 'ui-meal-card-filtered' : 'ui-meal-card-clickable'}`} 
      key={`${candidate.food.id}-${candidate.matched_query}-${index}`}
      role={filteredReason ? undefined : 'button'}
      tabIndex={filteredReason ? undefined : 0}
      onClick={openDetailPage}
      onKeyDown={handleCardKeyDown}
      >
      <Box className="ui-meal-photo" aria-hidden={!candidate.food.image_url}>
        {candidate.food.image_url ? (
          <img src={candidate.food.image_url} alt={candidate.food.name} loading="lazy" />
        ) : (
          <RecommendationIcon name="bowl" size={24} />
        )}
      </Box>
      {successMeassage && (
        <Box
          className='ui-success-toast'
          role="status"
          aria-live='polite'
        >
          <FiCheck className="ui-success-toast-icon" aria-hidden="true" />
          
          <Text fw={900}>{successMeassage}</Text>
        </Box>
      )}

      <Box className="ui-meal-summary">
        <Title order={2}>{candidate.food.name}</Title>
        <Text>{Math.round(candidate.food.calories)} kcal</Text>
        <Text className="ui-meal-price">
          {formatRM(candidate.generated_meal.estimated_price_range.min)} - {formatRM(candidate.generated_meal.estimated_price_range.max)}
        </Text>
        {filteredReason && (
          <Badge color="red" className="ui-meal-filtered-badge">
            Reason: {filteredReason}
          </Badge>
        )}
      </Box>

      <Button 
        className="ui-meal-log-button" 
        variant="subtle"
        disabled={filteredReason !== null}
        onClick={(event) => {
          event.stopPropagation()
          openLogModal(candidate)
        }}
        >
        Log
      </Button>
    </Box>
    )
  }

  const today = new Date()
  const options = {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  } as const
  const formattedDate = today.toLocaleDateString('en-US', options)

  return (
    <Box className={`ui-settings-page ui-recommendation-page ${isDarkMode ? 'ui-recommendation-dark' : ''}`}>
      <Box component="main" className="ui-settings-frame">
        <nav className="ui-settings-nav ui-surface" aria-label="Main navigation">
          <Link to="/recommendation" aria-current="page">Recommendation</Link>
          <Link to="/meal-logs">Logs</Link>
          <Link to="/preferences">Preferences</Link>
          <button
            className="ui-theme-toggle"
            type="button"
            aria-label={isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'}
            aria-pressed={isDarkMode}
            onClick={() => setIsDarkMode((current) => !current)}
          >
            {isDarkMode ? (
              <Sun size={19} strokeWidth={2.35} aria-hidden="true" />
            ) : (
              <Moon size={19} strokeWidth={2.35} aria-hidden="true" />
            )}
          </button>
        </nav>

        <Box className="ui-recommendation-layout">
        <Text 
          size="lg" 
          fw={700} 
          style={{
          textAlign: 'center',
          marginBottom: '10px',
        }}>{formattedDate}</Text>

          <Box component="section" className="ui-recommendation-results">
            <Accordion
              value={activeCategory}
              onChange={handleCategoryChange}
              className="ui-meal-accordion"
              chevronPosition="right"
            >
              {mealCategoryOptions.map((option) => {
                const candidates = candidatesByCategory[option.value]
                const filteredOut = filteredOutByCategory[option.value]
                const isLoading = loadingByCategory[option.value]
                const hasGenerated = generatedByCategory[option.value]
                const error = errorsByCategory[option.value]
                const visibleItems =
                  displayMode === 'filtered'
                    ? candidates.map((candidate) => ({
                        candidate,
                        filteredReason: null,
                      }))
                    : [
                        ...candidates.map((candidate) => ({
                          candidate,
                          filteredReason: null,
                        })),
                        ...filteredOut.map((item) => ({
                          candidate: item.candidate,
                          filteredReason: item.reason,
                        })),
                      ]

                

                return (
                  <Accordion.Item value={option.value} key={option.value} className="ui-meal-accordion-item">
                    <Accordion.Control className="ui-meal-accordion-control">
                      <Box className="ui-meal-accordion-heading">
                        <Box className="ui-meal-accordion-icon">
                          <RecommendationIcon name="fork" size={20} />
                        </Box>
                        <Title order={2}>{option.label}</Title>
                        <Badge className="ui-meal-accordion-count">
                          {isLoading ? 'Generating' : hasGenerated ? `${candidates.length} matches` : 'Folded'}
                        </Badge>
                      </Box>
                    </Accordion.Control>

                    <Accordion.Panel className="ui-meal-accordion-panel">
                      {error && (
                        <Alert color="red" icon={<RecommendationIcon name="warning" size={20} />}>
                          {error}
                        </Alert>
                      )}
                      
                      <Box className="ui-recommendation-filter-row">
                        <SegmentedControl
                          className="ui-recommendation-filter-toggle"
                          value={displayMode}
                          onChange={(value) => setDisplayMode(value as 'filtered' | 'all')}
                          data={[
                            { label: 'Show recommended only', value: 'filtered' },
                            { label: 'Show all results', value: 'all' },
                          ]}
                        />
                      </Box>
                      {isLoading && (
                        <Box className="ui-recommendation-state ui-card">
                          <RecommendationIcon name="sparkle" size={28} />
                          <Text fw={900}>Generating and sorting {option.label.toLowerCase()} meals...</Text>
                        </Box>
                      )}

                      {!isLoading && hasGenerated && visibleItems.length === 0 && (
                        <Box className="ui-recommendation-state ui-card">
                          <RecommendationIcon name="bowl" size={30} />
                          <Text fw={900}>No dataset matches found.</Text>
                          <Text className="ui-field-copy">
                            Try regenerating or add your custom meal.
                          </Text>
                        </Box>
                      )}

                      {!isLoading && !hasGenerated && (
                        <Box className="ui-recommendation-state ui-card">
                          <RecommendationIcon name="bowl" size={30} />
                          <Text fw={900}>Preparing this mealtime.</Text>
                        </Box>
                      )}

                      {!isLoading && visibleItems.length > 0 && (
                        <Box className="ui-recommendation-grid">
                          {visibleItems.map((item, index) => renderCandidateCard(option.value, item, index))}
                        </Box>
                      )}

                      <Button
                        className="ui-primary-button ui-recommendation-regenerate"
                        leftSection={<RecommendationIcon name="refresh" size={18} />}
                        onClick={() => generateRecommendations(option.value, true)}
                        loading={isLoading}
                      >
                        Regenerate {option.label.toLowerCase()}
                      </Button>

                      <Button
                        component={Link}
                        to={`/meals/add/${option.value}`}
                        className="ui-ghost-button ui-recommendation-add-meal"
                        variant="subtle"
                      >
                        Add other meal to {option.label.toLowerCase()}
                      </Button>
                    </Accordion.Panel>
                  </Accordion.Item>
                )
              })}
            </Accordion>
          </Box>
        </Box>
      </Box>
      <LogMealModal
        opened={mealToLog !== null}
        meal={mealToLog}
        onClose={() => setMealToLog(null)}
        onLogged={() => {
          setSuccessMessage(`${mealToLog?.name ?? 'Meal'} logged successfully`)
          setMealToLog(null)
        }}
      />
    </Box>
  )
}
