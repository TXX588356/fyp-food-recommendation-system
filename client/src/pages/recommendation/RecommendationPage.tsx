import {
  Alert,
  Box,
  Text,
} from '@mantine/core'
import axios from 'axios'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { ClipboardCheck, History, ListPlus, RotateCcw } from 'lucide-react'

import { useAuth } from '@/auth/useAuth'
import './RecommendationPage.css'
import '@/App.css'
import type { LoggableMeal, MealLogMonthResponse } from '@/pages/mealLog/mealLogTypes'
import { toMonthKey } from '@/pages/mealLog/mealLogHelpers'
import LogMealModal from '@/pages/mealLog/LogMealModal'
import { FiCheck } from 'react-icons/fi'
import { useLocation, useNavigate } from 'react-router-dom'
import { resolveWikipediaMealImage } from './wikiMealImages'
import { formatLocation, parseLocation } from '@/preferences/helpers'
import type { LocationValue, PreferenceData } from '@/preferences/types'
import DynamicLocationSelect from './components/DynamicLocationSelect'
import RecommendationAccordion from './components/RecommendationAccordion'
import RecommendationIcon from './components/RecommendationIcon'
import RecommendationMealCard from './components/RecommendationMealCard'
import RecommendationSpotlight from './components/RecommendationSpotlight'
import type {
  CategoryState,
  FilteredMealCandidate,
  GenerateRecommendationsResponse,
  MatchedMealCandidate,
  MealCategory,
  RecommendationLocationOption,
  RecommendationTourStep,
  SpotlightRect,
  VisibleRecommendationItem,
} from './recommendationTypes'
import {
  buildRecommendationStorageKey,
  emptyCandidates,
  emptyFilteredOut,
  emptyGenerated,
  getLocalDateKey,
  loadCurrentRecommendationLocation,
  loadPersistedRecommendations,
  saveCurrentRecommendationLocation,
  savePersistedRecommendations,
} from './recommendationStorage'
import MainNav from '@/theme/MainNav'
import { useThemeMode } from '@/theme/ThemeContext'
import {
  getCustomMealCreatedCategory,
  getCustomMealSuccessMessage,
} from '../customMeal/customMealNavigation'


const mealCategoryOptions: Array<{ value: MealCategory; label: string;}> = [
  { value: 'breakfast', label: 'Breakfast'},
  { value: 'lunch', label: 'Lunch'},
  { value: 'dinner', label: 'Dinner'},
  { value: 'snack', label: 'Snack'},
]

const emptyLoading: CategoryState<boolean> = {
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

const generatedTourStorageKeyPrefix = 'recommendation-generated-tour-v2-dismissed'
const postLogTourStorageKeyPrefix = 'recommendation-post-log-tour-v2-dismissed'

const buildUserTourStorageKey = (prefix: string, userKey: string | undefined) =>
  `${prefix}:${userKey ?? 'anonymous'}`

const selectDefaultRecommendationLocation = (preference: PreferenceData) => {
  const today = new Date().getDay()

  if (today === 0 || today === 6) { 
    return preference.homeLocation
  }
  
  return preference.workSchoolLocation
}

const buildSavedLocationOptions = (preference: PreferenceData): RecommendationLocationOption[] => {
  const locationsByValue = new Map<string, string[]>()
  const savedLocations = [
    { value: preference.workSchoolLocation.trim(), label: 'Work / school' },
    { value: preference.homeLocation.trim(), label: 'Home' },
  ]

  savedLocations.forEach((location) => {
    if (!location.value) return

    locationsByValue.set(location.value, [
      ...(locationsByValue.get(location.value) ?? []),
      location.label,
    ])
  })

  return Array.from(locationsByValue.entries()).map(([value, labels]) => ({
    value,
    label: `${labels.join(' and ')}: ${value}`,
  }))
}

export default function RecommendationPage() {
  const { user } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const userKey = user?.id ?? user?.email

  const recommendationStorageKey = useMemo(
    () => buildRecommendationStorageKey(userKey),
    [userKey],
  )
  const generatedTourStorageKey = useMemo(
    () => buildUserTourStorageKey(generatedTourStorageKeyPrefix, userKey),
    [userKey],
  )
  const postLogTourStorageKey = useMemo(
    () => buildUserTourStorageKey(postLogTourStorageKeyPrefix, userKey),
    [userKey],
  )

  const persistedRecommendations = useMemo(
    () => loadPersistedRecommendations(recommendationStorageKey),
    [recommendationStorageKey],
  )

  const [activeCategory, setActiveCategory] = useState<MealCategory | null>(() => {
    const createdCategory = getCustomMealCreatedCategory(location.state)

    return mealCategoryOptions.some((option) => option.value === createdCategory)
      ? createdCategory as MealCategory
      : null
  })
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
  const [successMessage, setSuccessMessage] = useState<string | null>(() =>
    getCustomMealSuccessMessage(location.state),
  )

  const [tourGeneratedCategory, setTourGeneratedCategory] = useState<MealCategory | null>(null)
  const [activeTourStep, setActiveTourStep] = useState<RecommendationTourStep | null>(null)
  const [spotlightRect, setSpotlightRect] = useState<SpotlightRect | null>(null)
  const [hasLoggedRecommendation, setHasLoggedRecommendation] = useState(false)

  const finishGeneratedTour = useCallback(() => {
    localStorage.setItem(generatedTourStorageKey, 'true')
    setActiveTourStep(null)
    setTourGeneratedCategory(null)
    setSpotlightRect(null)
  }, [generatedTourStorageKey])

  const finishPostLogTour = useCallback(() => {
    localStorage.setItem(postLogTourStorageKey, 'true')
    setActiveTourStep(null)
    setSpotlightRect(null)
  }, [postLogTourStorageKey])

  const dismissActiveTour = useCallback(() => {
    if (activeTourStep === 'logsNav') {
      finishPostLogTour()
      return
    }

    finishGeneratedTour()
  }, [activeTourStep, finishGeneratedTour, finishPostLogTour])

  const advanceActiveTour = useCallback(() => {
    if (activeTourStep === 'logMeal') {
      setActiveTourStep('regenerate')
      return
    }

    if (activeTourStep === 'regenerate') {
      setActiveTourStep('addMeal')
      return
    }

    if (activeTourStep === 'addMeal') {
      finishGeneratedTour()
      return
    }

    if (activeTourStep === 'logsNav') {
      finishPostLogTour()
      navigate('/meal-logs')
    }
  }, [activeTourStep, finishGeneratedTour, finishPostLogTour, navigate])

  const { isDarkMode } = useThemeMode()
  const [savedLocations, setSavedLocations] = useState<RecommendationLocationOption[]>([])
  const [selectedLocation, setSelectedLocation] = useState<LocationValue>({
    state: '',
    district: '',
  })
  const [locationError, setLocationError] = useState<string | null>(null)

  useEffect(() => {
    if (!activeCategory || activeCategory !== tourGeneratedCategory) {
      return
    }

    const guideDismissed = localStorage.getItem(generatedTourStorageKey) === 'true'
    const hasVisibleCandidates = candidatesByCategory[activeCategory].length > 0

    if (
      !activeTourStep &&
      generatedByCategory[activeCategory] &&
      hasVisibleCandidates &&
      !guideDismissed &&
      !hasLoggedRecommendation
    ) {
      const guideTimer = window.setTimeout(() => setActiveTourStep('logMeal'), 0)

      return () => window.clearTimeout(guideTimer)
    }
  }, [
    activeCategory,
    activeTourStep,
    candidatesByCategory,
    generatedByCategory,
    generatedTourStorageKey,
    hasLoggedRecommendation,
    tourGeneratedCategory,
  ])

  useEffect(() => {
    if (!activeTourStep) {
      return
    }

    let hasScrolledTargetIntoView = false
    let scrollMeasureTimer: number | undefined

    const updateSpotlightRect = () => {
      const selectorByStep: Record<RecommendationTourStep, string> = {
        logMeal: '[data-recommendation-tour="log-meal"]',
        regenerate: '[data-recommendation-tour="regenerate"]',
        addMeal: '[data-recommendation-tour="add-meal"]',
        logsNav: '[data-main-nav-tour="logs"]',
      }
      const target = document.querySelector<HTMLElement>(selectorByStep[activeTourStep])

      if (!target) {
        setSpotlightRect(null)
        return
      }

      const rect = target.getBoundingClientRect()
      const viewportPadding = window.innerWidth <= 680 ? 120 : 96
      const isTargetAboveViewport = rect.top < viewportPadding
      const isTargetBelowViewport = rect.bottom > window.innerHeight - viewportPadding

      if (!hasScrolledTargetIntoView && (isTargetAboveViewport || isTargetBelowViewport)) {
        hasScrolledTargetIntoView = true

        const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
        target.scrollIntoView({
          behavior: prefersReducedMotion ? 'auto' : 'smooth',
          block: 'center',
          inline: 'nearest',
        })

        scrollMeasureTimer = window.setTimeout(updateSpotlightRect, prefersReducedMotion ? 0 : 280)
      }

      setSpotlightRect({
        top: rect.top,
        left: rect.left,
        width: rect.width,
        height: rect.height,
      })
    }

    updateSpotlightRect()

    window.addEventListener('resize', updateSpotlightRect)
    window.addEventListener('scroll', updateSpotlightRect, true)

    return () => {
      if (scrollMeasureTimer) {
        window.clearTimeout(scrollMeasureTimer)
      }
      window.removeEventListener('resize', updateSpotlightRect)
      window.removeEventListener('scroll', updateSpotlightRect, true)
    }
  }, [activeTourStep, activeCategory])

  useEffect(() => {
    if (!getCustomMealSuccessMessage(location.state)) {
      return
    }

    navigate('/recommendation', {
      replace: true,
      state: null,
    })
  }, [location.state, navigate])

  useEffect(() => {
    if (!successMessage) {
      return 
    }

    const dismissTimer = window.setTimeout(() => {
      setSuccessMessage(null)
    }, 4000)

    return () => window.clearTimeout(dismissTimer)
  }, [successMessage])

  const openLogModal = (candidate: MatchedMealCandidate) => {
    setMealToLog({
      source: candidate.food.source === 'custom' ? 'custom' : 'prebuilt',
      mealId: candidate.food.id,
      name: candidate.food.name,
      calories: candidate.food.calories,
    })
  }

  const token = localStorage.getItem('token')

  useEffect(() => {
    const loadPreferences = async () => {
      try {
        const response = await axios.get<PreferenceData>(
          "/preferences",
          {
            headers: {
              Authorization: `Bearer ${token}`,
            }
          }
        )

        const defaultLocation = selectDefaultRecommendationLocation(response.data)
        const currentLocation = loadCurrentRecommendationLocation(userKey)
        const initialLocation = currentLocation || defaultLocation
        const nextSavedLocations = buildSavedLocationOptions(response.data)

        setSavedLocations(nextSavedLocations)
        setSelectedLocation(parseLocation(initialLocation))
        if (!currentLocation) {
          saveCurrentRecommendationLocation(userKey, defaultLocation)
        }
      } catch (error) {
        console.error('Failed to load recommendation location preference', error)
        setLocationError('Could not load your saved recommendation location.')
      }
    }

    void loadPreferences()
  }, [token, userKey])

  const persistRecommendationCategory = (
  mealCategory: MealCategory,
  candidates: MatchedMealCandidate[],
  filteredOut: FilteredMealCandidate[],
  location: string,
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
    const nextLocationByCategory = {
      ...persistedRecommendations.locationByCategory,
      [mealCategory]: location,
    }

    savePersistedRecommendations(recommendationStorageKey, {
      generatedDate: getLocalDateKey(),
      candidatesByCategory: nextCandidatesByCategory,
      filteredOutByCategory: nextFilteredOutByCategory,
      generatedByCategory: nextGeneratedByCategory,
      locationByCategory: nextLocationByCategory,
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

    if (
      candidates.length > 0 &&
      localStorage.getItem(generatedTourStorageKey) !== 'true'
    ) {
      setTourGeneratedCategory(mealCategory)
    }
  }

  const generateRecommendations = async (mealCategory: MealCategory, force = false) => {
    if (loadingByCategory[mealCategory] || (generatedByCategory[mealCategory] && !force)) {
      return
    }

    const location = formatLocation(selectedLocation)
    if (!selectedLocation.state || !selectedLocation.district) {
      setErrorsByCategory((current) => ({
        ...current,
        [mealCategory]: 'Please select a recommendation location.'
      }))
      return
    }

    setLoadingByCategory((current) => ({ ...current, [mealCategory]: true }))
    saveCurrentRecommendationLocation(userKey, location)
    setErrorsByCategory((current) => ({ ...current, [mealCategory]: null }))

    try {
      const mealLogResponse = await axios.get<MealLogMonthResponse>(
        "/meal-logs",
        {
          params: { month: toMonthKey(new Date()) },
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )
      const currentMonthSpent = mealLogResponse.data.summary.totalSpent

      const response = await axios.post<GenerateRecommendationsResponse>(
        "/recommendations",
        {
          mealCategory,
          currentMonthSpent,
          location,
        },
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      persistRecommendationCategory(mealCategory, response.data.candidates ?? [], response.data.filtered_out ?? [], location)
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

  const updateCandidateImage = useCallback((
    mealCategory: MealCategory,
    mealId: string,
    imageUrl: string,
  ) => {
    if (!imageUrl) {
      return
    }

    setCandidatesByCategory((current) => ({
      ...current,
      [mealCategory]: current[mealCategory].map((candidate) => (
        candidate.food.id === mealId
          ? { ...candidate, food: { ...candidate.food, image_url: candidate.food.image_url || imageUrl } }
          : candidate
      )),
    }))

    setFilteredOutByCategory((current) => ({
      ...current,
      [mealCategory]: current[mealCategory].map((item) => (
        item.candidate.food.id === mealId
          ? {
              ...item,
              candidate: {
                ...item.candidate,
                food: {
                  ...item.candidate.food,
                  image_url: item.candidate.food.image_url || imageUrl,
                },
              },
            }
          : item
      )),
    }))

    const persistedRecommendations = loadPersistedRecommendations(recommendationStorageKey)
    savePersistedRecommendations(recommendationStorageKey, {
      ...persistedRecommendations,
      candidatesByCategory: {
        ...persistedRecommendations.candidatesByCategory,
        [mealCategory]: persistedRecommendations.candidatesByCategory[mealCategory].map((candidate) => (
          candidate.food.id === mealId
            ? { ...candidate, food: { ...candidate.food, image_url: candidate.food.image_url || imageUrl } }
            : candidate
        )),
      },
      filteredOutByCategory: {
        ...persistedRecommendations.filteredOutByCategory,
        [mealCategory]: persistedRecommendations.filteredOutByCategory[mealCategory].map((item) => (
          item.candidate.food.id === mealId
            ? {
                ...item,
                candidate: {
                  ...item.candidate,
                  food: {
                    ...item.candidate.food,
                    image_url: item.candidate.food.image_url || imageUrl,
                  },
                },
              }
            : item
        )),
      },
      locationByCategory: persistedRecommendations.locationByCategory,
    })
  }, [recommendationStorageKey])

  useEffect(() => {
    if (!activeCategory || loadingByCategory[activeCategory] || !generatedByCategory[activeCategory]) {
      return
    }

    const controller = new AbortController()
    const candidates = [
      ...candidatesByCategory[activeCategory],
      ...filteredOutByCategory[activeCategory].map((item) => item.candidate),
    ]
    const missingImageCandidates = candidates.filter((candidate) => (
      candidate.food.source !== 'custom' && !candidate.food.image_url
    ))

    const loadImages = async () => {
      for (const candidate of missingImageCandidates) {
        if (controller.signal.aborted) {
          return
        }

        const imageUrl = await resolveWikipediaMealImage(candidate.food.name, controller.signal)
        updateCandidateImage(activeCategory, candidate.food.id, imageUrl)

        await new Promise((resolve) => window.setTimeout(resolve, 350))
      }
    }

    void loadImages()

    return () => controller.abort()
    // The candidate arrays intentionally trigger lookups after recommendation state changes.
  }, [
    activeCategory,
    candidatesByCategory,
    filteredOutByCategory,
    generatedByCategory,
    loadingByCategory,
    recommendationStorageKey,
    updateCandidateImage,
  ])

  const handleCategoryChange = (value: string | null) => {
    const nextCategory = value as MealCategory | null
    setActiveCategory(nextCategory)

    if (nextCategory) {
      void generateRecommendations(nextCategory)
    }
  }

  const renderCandidateCard = (mealCategory: MealCategory, item: VisibleRecommendationItem, index: number) => {
    return (
      <RecommendationMealCard
        key={`${item.candidate.food.id}-${item.candidate.matched_query}-${index}`}
        mealCategory={mealCategory}
        item={item}
        index={index}
        isSpotlightTarget={
          activeTourStep === 'logMeal' &&
          mealCategory === activeCategory &&
          index === 0 &&
          item.filteredReason === null
        }
        onOpenDetail={(mealCategory, mealId) => navigate(`/recommendation/${mealCategory}/${mealId}`)}
        onLog={() => {
          openLogModal(item.candidate)
        }}
      />
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
  const activeCategoryLabel =
    mealCategoryOptions.find((option) => option.value === activeCategory)?.label ?? 'meal'
  const activeTourContent = activeTourStep
    ? {
        logMeal: {
          title: 'Log this meal',
          description: 'Save this meal to your history so it can be included in your spending and meal reports.',
          icon: <ClipboardCheck size={18} aria-hidden="true" />,
          primaryLabel: 'Next',
        },
        regenerate: {
          title: `Regenerate ${activeCategoryLabel.toLowerCase()}`,
          description: 'Not happy with these meals? Generate a new set using the same preferences and location.',
          icon: <RotateCcw size={18} aria-hidden="true" />,
          primaryLabel: 'Next',
        },
        addMeal: {
          title: `Add your own ${activeCategoryLabel.toLowerCase()}`,
          description: 'Can’t find the meal you ate? Add it manually so you can log it and use it in future recommendations.',
          icon: <ListPlus size={18} aria-hidden="true" />,
          primaryLabel: 'Got it',
        },
        logsNav: {
          title: 'View your meal logs',
          description: 'YSee your logged meals, update past entries, and generate weekly or monthly reports.',
          icon: <History size={18} aria-hidden="true" />,
          primaryLabel: 'Go to Logs',
        },
      }[activeTourStep]
    : null

  return (
    <Box className={`ui-settings-page ui-recommendation-page ${isDarkMode ? 'ui-recommendation-dark' : ''}`}>
      <Box component="main" className="ui-settings-frame">
        <MainNav
          active="recommendation"
          tourTarget={activeTourStep === 'logsNav' ? 'logs' : undefined}
        />

        <Box className="ui-recommendation-layout">
        <Text
          size="lg"
          fw={700}
          style={{
          textAlign: 'center',
          marginBottom: '10px',
        }}>{formattedDate}</Text>

        {locationError && (
          <Alert color='red' icon={< RecommendationIcon name="warning" size={20} />}>
            {locationError}
          </Alert>
        )}

        <DynamicLocationSelect 
          selectedLocation={selectedLocation}
          savedLocations={savedLocations}
          onChange={(location) => {
            saveCurrentRecommendationLocation(userKey, formatLocation(location))
            setSelectedLocation(location)
            setGeneratedByCategory(emptyGenerated)
            setCandidatesByCategory(emptyCandidates)
            setFilteredOutByCategory(emptyFilteredOut)
            setTourGeneratedCategory(null)
          }}
        />

          <Box component="section" className="ui-recommendation-results">
            <RecommendationAccordion
              activeCategory={activeCategory}
              mealCategoryOptions={mealCategoryOptions}
              candidatesByCategory={candidatesByCategory}
              filteredOutByCategory={filteredOutByCategory}
              loadingByCategory={loadingByCategory}
              generatedByCategory={generatedByCategory}
              errorsByCategory={errorsByCategory}
              displayMode={displayMode}
              selectedLocation={selectedLocation}
              activeTourStep={activeTourStep}
              onCategoryChange={handleCategoryChange}
              onDisplayModeChange={setDisplayMode}
              onGenerateRecommendations={generateRecommendations}
              renderCandidateCard={renderCandidateCard}
            />
          </Box>
        </Box>
      </Box>
      {activeTourStep && activeTourContent && (
        <RecommendationSpotlight
          targetRect={spotlightRect}
          title={activeTourContent.title}
          description={activeTourContent.description}
          icon={activeTourContent.icon}
          primaryLabel={activeTourContent.primaryLabel}
          onDismiss={dismissActiveTour}
          onPrimary={advanceActiveTour}
        />
      )}
      <LogMealModal
        opened={mealToLog !== null}
        meal={mealToLog}
        onClose={() => setMealToLog(null)}
        onLogged={() => {
          setSuccessMessage(`${mealToLog?.name ?? 'Meal'} logged successfully`)
          setHasLoggedRecommendation(true)
          if (localStorage.getItem(postLogTourStorageKey) !== 'true') {
            setActiveTourStep('logsNav')
          }
          setMealToLog(null)
        }}
      />
      {successMessage && (
        <Box
          className='ui-success-toast'
          role="status"
          aria-live='polite'
        >
          <FiCheck className="ui-success-toast-icon" aria-hidden="true" />
          <Text fw={900}>{successMessage}</Text>
        </Box>
      )}
    </Box>
  )
}
