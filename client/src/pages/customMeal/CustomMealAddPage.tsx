import {
  Alert,
  Box,
  Button,
  Text,
  Title,
} from '@mantine/core'
import axios from 'axios'
import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { FiCheck } from 'react-icons/fi'

import { restrictedMealCategories } from '@/preferences/options'
import './CustomMealAddPage.css'
import '@/App.css'
import {
  customMealCreatedNavigationState,
  getCustomMealSuccessMessage,
} from './customMealNavigation'
import MainNav from '@/theme/MainNav'
import type { PreferenceData } from '@/preferences/types'
import LogMealModal from '../mealLog/LogMealModal'
import type { LoggableMeal } from '../mealLog/mealLogTypes'
import { resolveWikipediaMealImage } from '../recommendation/wikiMealImages'
import {
  appendPersistedRecommendationCandidate,
  buildRecommendationStorageKey,
  loadCurrentRecommendationLocation,
} from '../recommendation/recommendationStorage'
import type { MatchedMealCandidate } from '../recommendation/recommendationTypes'
import { useAuth } from '@/auth/useAuth'
import { replayInputShake } from '@/theme/inputTransitions'
import type {
  CustomMealAutocompleteResponse,
  CustomMealDraft,
  CustomMealField,
  CustomMealFieldErrors,
  CustomMealResponse,
  ExistingMeal,
  MealSearchResult,
  MealTime,
} from './customMealTypes'
import CustomMealBasicFields from './components/CustomMealBasicFields'
import CustomMealConsentModal from './components/CustomMealConsentModal'
import CustomMealFormActions from './components/CustomMealFormActions'
import CustomMealLocationFields from './components/CustomMealLocationFields'
import CustomMealTagFields from './components/CustomMealTagFields'
import ExistingMealCard from './components/ExistingMealCard'
import MealAddIcon from './components/MealAddIcon'

const mealTimeLabels: Record<MealTime, string> = {
  breakfast: 'Breakfast',
  lunch: 'Lunch',
  dinner: 'Dinner',
  snack: 'Snack',
}

const stateOptions = [
  'Johor',
  'Kedah',
  'Kelantan',
  'Kuala Lumpur',
  'Labuan',
  'Malacca',
  'Negeri Sembilan',
  'Pahang',
  'Penang',
  'Perak',
  'Perlis',
  'Putrajaya',
  'Sabah',
  'Sarawak',
  'Selangor',
  'Terengganu',
].map((state) => ({ value: state, label: state }))

const getRestrictedMealCategoryTags = (dietaryRestrictionTags: string[]) =>
  new Set(
    dietaryRestrictionTags.flatMap(
      (restriction) => restrictedMealCategories[restriction] ?? [],
    ),
  )

const filterRestrictedMealCategoryTags = (
  mealCategoryTags: string[],
  dietaryRestrictionTags: string[],
) => {
  const restrictedTags = getRestrictedMealCategoryTags(dietaryRestrictionTags)

  return mealCategoryTags.filter((tag) => !restrictedTags.has(tag))
}

const validateMealCategoryTags = (
  mealCategoryTags: string[],
  dietaryRestrictionTags: string[],
) => {
  if (mealCategoryTags.length === 0) {
    return 'Please select at least one meal category.'
  }

  const restrictedTags = getRestrictedMealCategoryTags(dietaryRestrictionTags)
  const restrictedSelectedTag = mealCategoryTags.find((tag) => restrictedTags.has(tag))

  if (restrictedSelectedTag) {
    return 'Please remove meal category tags that conflict with selected dietary restrictions.'
  }

  return null
}

const emptyDraft: CustomMealDraft = {
  name: '',
  price: '',
  calories: '',
  carbsG: '',
  fatG: '',
  proteinG: '',
  state: '',
  district: '',
  restaurantName: '',
  dietaryRestrictionTags: [],
  mealCategoryTags: [],
}

const isMealTime = (value: string | undefined): value is MealTime =>
  value === 'breakfast' || value === 'lunch' || value === 'dinner' || value === 'snack'

const formatRM = (value: number) => `RM ${value.toFixed(2)}`

function toExistingMeal(meal: MealSearchResult): ExistingMeal {
  return {
    id: meal.id,
    name: meal.name,
    calories: meal.calories,
    priceLabel: meal.price === undefined ? 'Prebuilt data' : formatRM(meal.price),
    tags: meal.tags,
    source: meal.source,
    imageUrl: meal.image_url,
  }
}

function toCustomRecommendationCandidate(meal: CustomMealResponse): MatchedMealCandidate {
  return {
    generated_meal: {
      name: meal.name,
      alternative_search_terms: [],
      estimated_price_range: {
        min: meal.price,
        max: meal.price,
      },
      sodium_level: '',
      sugar_level: '',
      purine_risk: '',
      health_flags: {},
    },
    food: {
      id: meal.id,
      name: meal.name,
      source: 'custom',
      tags: meal.mealCategoryTags,
      calories: meal.calories,
      fat_g: meal.fatG,
      protein_g: meal.proteinG,
      carbs_g: meal.carbsG,
      price: meal.price,
      image_url: meal.imageURL,
    },
    matched_query: meal.name,
  }
}

export function CustomMealSearchPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { mealCategory } = useParams()
  const { user } = useAuth()  
  const mealTime = isMealTime(mealCategory) ? mealCategory : 'breakfast'
  const recommendationLocation = new URLSearchParams(location.search).get('location') ?? ''
  const [query, setQuery] = useState('')
  const [visibleMeals, setVisibleMeals] = useState<ExistingMeal[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mealToLog, setMealToLog] = useState<LoggableMeal | null>(null)
  const [successMessage, setSuccessMessage] = useState(() =>
    getCustomMealSuccessMessage(location.state),
  )

  const token = localStorage.getItem('token')
  const userKey = user?.id ?? user?.email

  const updateSearchQuery = (nextQuery: string) => {
    setQuery(nextQuery)

    if (!nextQuery.trim()) {
      setVisibleMeals([])
      setIsLoading(false)
      setError(null)
    }
  }

  useEffect(() => {
    if (!successMessage) {
      return
    }

    navigate(`${location.pathname}${location.search}`, {
      replace: true,
      state: null,
    })

    const dismissTimer = window.setTimeout(() => {
      setSuccessMessage(null)
    }, 4000)

    return () => window.clearTimeout(dismissTimer)
  }, [location.pathname, location.search, navigate, successMessage])

  useEffect(() => {
    const normalizedQuery = query.trim()

    if (!normalizedQuery) {
      return
    }

    const request = new AbortController()

    const loadMeals = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await axios.get<MealSearchResult[]>(
          "/meals/search",
          {
            params: {
              q: normalizedQuery,
            },
            headers: {
              Authorization: `Bearer ${token}`,
            },
            signal: request.signal,  // to cancel previous network request tied to the previous query
          },
        )

        setVisibleMeals((response.data ?? []).map(toExistingMeal))
      } catch (error) {
        if (axios.isCancel(error)) {
          return
        }

        console.error('Failed to load meals for logging', error)
        setError('Could not load meals right now.')
      } finally {
        if (!request.signal.aborted) {
          setIsLoading(false)
        }
      }
    }

    void loadMeals()

    return () => request.abort()
  }, [query, token])

  useEffect(() => {
    const mealsWithoutImages = visibleMeals.filter((meal) => (
      meal.source === 'prebuilt' && !meal.imageUrl
    ))
    if (mealsWithoutImages.length === 0) {
      return
    }

    const controller = new AbortController()

    const loadImages = async () => {
      for (const meal of mealsWithoutImages) {
        if (controller.signal.aborted) {
          return
        }

        const imageUrl = await resolveWikipediaMealImage(meal.name, controller.signal)
        if (imageUrl) {
          setVisibleMeals((current) => current.map((currentMeal) => (
            currentMeal.source === 'prebuilt' && currentMeal.id === meal.id
              ? { ...currentMeal, imageUrl: currentMeal.imageUrl || imageUrl }
              : currentMeal
          )))
        }

        await new Promise((resolve) => window.setTimeout(resolve, 350))
      }
    }

    void loadImages()

    return () => controller.abort()
  }, [visibleMeals])

  const openLogModal = (meal: ExistingMeal) => {
    setMealToLog({
      source: meal.source,
      mealId: meal.id,
      name: meal.name,
      calories: meal.calories,
    })
  }

  const openMealDetail = (meal: ExistingMeal) => {
    const currentRecommendationLocation = loadCurrentRecommendationLocation(userKey)
    const detailLocation = currentRecommendationLocation || recommendationLocation
    const locationSearch = detailLocation
      ? `?location=${encodeURIComponent(detailLocation)}`
      : ''

    navigate(`/meals/${meal.source}/${meal.id}${locationSearch}`)
  }

  const handleMealCardKeyDown = (
    event: KeyboardEvent<HTMLDivElement>,
    meal: ExistingMeal,
  ) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return
    }

    event.preventDefault()
    openMealDetail(meal)
  }

  return (
    <Box className="ui-settings-page ui-meal-add-page">
      {successMessage && (
        <Box
          className="ui-success-toast"
          role="status"
          aria-live="polite"
        >
          <FiCheck className="ui-success-toast-icon" aria-hidden="true" />
          <Text fw={900}>{successMessage}</Text>
        </Box>
      )}

      <LogMealModal
        opened={mealToLog !== null}
        meal={mealToLog}
        onClose={() => setMealToLog(null)}
        onLogged={() => {
          setSuccessMessage(`${mealToLog?.name ?? 'Meal'} logged successfully`)
          setMealToLog(null)
        }}
      />

      <Box component="main" className="ui-settings-frame">
        <MainNav active="recommendation" />

        <Box className="ui-meal-add-header">
          <Title order={1}>Add to {mealTimeLabels[mealTime]}</Title>
        </Box>

        <Box component="section" className="ui-meal-add-panel ui-surface">
          <Box className="ui-meal-search-field">
            <MealAddIcon name="search" />
            <input
              aria-label="Search meals"
              placeholder="Search"
              value={query}
              onChange={(event) => updateSearchQuery(event.currentTarget.value)}
            />
          </Box>

          <button
            type="button"
            className="ui-custom-meal-entry"
            onClick={() => navigate(`/meals/add/${mealTime}/custom`)}
          >
            <span>
              <MealAddIcon name="plus" />
            </span>
            <Box className="ui-custom-meal-entry-copy">
              <strong>Create a custom meal</strong>
              <Text component="small">
                Cannot find it in search? Add your own meal details and save it to this {mealTimeLabels[mealTime].toLowerCase()}.
              </Text>
            </Box>
            <Box className="ui-custom-meal-entry-arrow" aria-hidden="true">
              →
            </Box>
          </button>

          {error && <Alert color="red">{error}</Alert>}

          {query.trim().length === 0 ? (
            <Box className="ui-meal-add-state ui-card">
              <Text fw={900}>Search for an existing meal.</Text>
              <Text className="ui-field-copy">Results will appear here after you enter a meal name.</Text>
            </Box>
          ) : isLoading ? (
            <Box className="ui-meal-add-state ui-card">
              <Text fw={900}>Loading meals...</Text>
            </Box>
          ) : visibleMeals.length === 0 ? (
            <Box className="ui-meal-add-state ui-card">
              <Text fw={900}>No meals found.</Text>
              <Text className="ui-field-copy">Create a custom meal item to add it to your list.</Text>
            </Box>
          ) : (
            <Box className="ui-meal-add-list">
              {visibleMeals.map((meal) => (
                <ExistingMealCard
                  key={`${meal.source}-${meal.id}`}
                  meal={meal}
                  onOpenDetail={openMealDetail}
                  onLog={openLogModal}
                  onKeyDown={handleMealCardKeyDown}
                />
              ))}
            </Box>
          )}

          <Button
              variant="subtle"
              className="ui-dark-button"
              leftSection={<span aria-hidden="true">←</span>}
              onClick={() => navigate('/recommendation')}
            >
              Back
            </Button>
        </Box>
      </Box>
    </Box>
  )
}

export function CustomMealFormPage() {
  const navigate = useNavigate()
  const { mealCategory } = useParams()
  const { user } = useAuth()
  const mealTime = isMealTime(mealCategory) ? mealCategory : 'breakfast'
  const [draft, setDraft] = useState<CustomMealDraft>(emptyDraft)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mealImage, setMealImage] = useState<File | null>(null)
  const [fieldErrors, setFieldErrors] = useState<CustomMealFieldErrors>({})
  const [isConsentModalOpen, setIsConsentModalOpen] = useState(false)
  const [isCheckingConsent, setIsCheckingConsent] = useState(false)
  const [consentChoiceBeingSaved, setConsentChoiceBeingSaved] = useState<boolean | null>(null)
  const [isAutocompleting, setIsAutocompleting] = useState(false)
  const [errorShakeSequence, setErrorShakeSequence] = useState(0)
  const formRef = useRef<HTMLDivElement | null>(null)
  const fieldErrorsRef = useRef(fieldErrors)

  const token = localStorage.getItem('token')
  const userKey = user?.id ?? user?.email
  const restrictedMealCategoryTags = getRestrictedMealCategoryTags(draft.dietaryRestrictionTags)

  const inputClassNames = (field: CustomMealField) => ({
    label: 'ui-input-label',
    input: `ui-input t-input ${fieldErrors[field] ? 'is-error' : ''}`,
    wrapper: `ui-input-wrapper t-input-wrap ${fieldErrors[field] ? 'is-error' : ''}`,
  })

  const clearFieldError = (field: CustomMealField) => {
    setFieldErrors((current) => {
      if (!current[field]) {
        return current
      }

      const next = { ...current }
      delete next[field]
      return next
    })
  }

  const showFieldErrors = (errors: CustomMealFieldErrors) => {
    setFieldErrors(errors)
    setErrorShakeSequence((current) => current + 1)
  }

  useEffect(() => {
    fieldErrorsRef.current = fieldErrors
  }, [fieldErrors])

  useEffect(() => {
    if (errorShakeSequence === 0) {
      return
    }

    const firstInvalidField = Object.keys(fieldErrorsRef.current)[0]

    if (!firstInvalidField) {
      return
    }

    const animationFrame = window.requestAnimationFrame(() => {
      replayInputShake('.ui-custom-meal-form .t-input.is-error')

      const firstInvalidElement = formRef.current?.querySelector<HTMLElement>(
        `[data-custom-meal-field="${firstInvalidField}"]`,
      )

      firstInvalidElement?.scrollIntoView({ behavior: 'smooth', block: 'center' })

      const focusTarget = firstInvalidElement?.matches('input, button, [tabindex]')
        ? firstInvalidElement
        : firstInvalidElement?.querySelector<HTMLElement>('input, button, [tabindex]')

      focusTarget?.focus({ preventScroll: true })
    })

    return () => window.cancelAnimationFrame(animationFrame)
  }, [errorShakeSequence])

  const updateDraft = <Key extends keyof CustomMealDraft>(key: Key, value: CustomMealDraft[Key]) => {
    if (key !== 'dietaryRestrictionTags') {
      clearFieldError(key)
    }
    setError(null)
    setDraft((current) => ({
      ...current,
      [key]: value,
    }))
  }

  const validateAutocompleteMealName = (name: string) => {
    if (name.trim().length < 2) {
      return 'Enter a meaningful name before using AI autocomplete'
    }

    return null
  }

  const autocompleteCustomMeal = async () => {
    const validationError = validateAutocompleteMealName(draft.name)

    if (validationError) {
      showFieldErrors({ name: validationError })
      return
    }

    setIsAutocompleting(true)
    setError(null)
    setFieldErrors({})

    try {
      const response = await axios.post<CustomMealAutocompleteResponse>(
        "/custom-meals/autocomplete",
        { name: draft.name },
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      setDraft((current) => ({
        ...current,
        calories: response.data.calories,
        fatG: response.data.fatG,
        proteinG: response.data.proteinG,
        carbsG: response.data.carbsG,
        dietaryRestrictionTags: response.data.dietaryRestrictionTags,
        mealCategoryTags: filterRestrictedMealCategoryTags(
          response.data.mealCategoryTags,
          response.data.dietaryRestrictionTags,
        )
      }))
    } catch (error) {
      console.error("Failed to generate meal details: ", error)
      setError('Could not generate meal details.')
      formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } finally {
      setIsAutocompleting(false)
    }
  }

  const toggleDietaryTag = (tag: string) => {
    setError(null)
    setDraft((current) => {
      const hasTag = current.dietaryRestrictionTags.includes(tag)
      const nextDietaryRestrictionTags = hasTag
        ? current.dietaryRestrictionTags.filter((currentTag) => currentTag !== tag)
        : [...current.dietaryRestrictionTags, tag]

      return {
        ...current,
        dietaryRestrictionTags: nextDietaryRestrictionTags,
        mealCategoryTags: filterRestrictedMealCategoryTags(
          current.mealCategoryTags,
          nextDietaryRestrictionTags,
        ),
      }
    })
  }

  const toggleMealCategoryTag = (tag: string) => {
    clearFieldError('mealCategoryTags')
    setError(null)
    setDraft((current) => {
      const restrictedTags = getRestrictedMealCategoryTags(current.dietaryRestrictionTags)

      if (restrictedTags.has(tag)) {
        return current
      }

      const hasTag = current.mealCategoryTags.includes(tag)

      return {
        ...current,
        mealCategoryTags: hasTag
          ? current.mealCategoryTags.filter((currentTag) => currentTag !== tag)
          : [...current.mealCategoryTags, tag],
      }
    })
  }

  const validateDraft = (): CustomMealFieldErrors => {
    const errors: CustomMealFieldErrors = {}

    if (!draft.name.trim()) {
      errors.name = 'Please enter a meal name.'
    }

    if (draft.price === '') {
      errors.price = 'Please enter the meal price.'
    }

    if (draft.calories === '') {
      errors.calories = 'Please enter calories.'
    }

    if (draft.carbsG === '') {
      errors.carbsG = 'Please enter carbs.'
    }

    if (draft.fatG === '') {
      errors.fatG = 'Please enter fat.'
    }

    if (draft.proteinG === '') {
      errors.proteinG = 'Please enter protein.'
    }

    if (!draft.state) {
      errors.state = 'Please select a state.'
    }

    if (!draft.district.trim()) {
      errors.district = 'Please enter the district.'
    }

    if (!draft.restaurantName.trim()) {
      errors.restaurantName = 'Please enter the restaurant name.'
    }

    const categoryError = validateMealCategoryTags(
      draft.mealCategoryTags,
      draft.dietaryRestrictionTags,
    )

    if (categoryError) {
      errors.mealCategoryTags = categoryError
    }

    if (mealImage && mealImage.size > 5 * 1024 * 1024) {
      errors.image = 'Image must not exceed 5 MB.'
    }

    return errors
  }

  const createCustomMeal = async () => {
    const payload = {
      name: draft.name,
      price: draft.price,
      calories: draft.calories,
      fatG: draft.fatG,
      proteinG: draft.proteinG,
      carbsG: draft.carbsG,
      state: draft.state,
      district: draft.district,
      restaurantName: draft.restaurantName,
      dietaryRestrictionTags: draft.dietaryRestrictionTags,
      mealCategoryTags: draft.mealCategoryTags,
    }

    const formData = new FormData()

    formData.append('payload', JSON.stringify(payload))
    if (mealImage) {
      formData.append('image', mealImage)
    }

    setIsSaving(true)
    setError(null)
    setFieldErrors({})

    try {
      const response = await axios.post<CustomMealResponse>(
        "/custom-meals",
        formData,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      appendPersistedRecommendationCandidate(
        buildRecommendationStorageKey(userKey),
        mealTime,
        toCustomRecommendationCandidate(response.data),
        loadCurrentRecommendationLocation(userKey),
      )

      navigate('/recommendation', {
        state: customMealCreatedNavigationState(mealTime),
      })
    } catch (error) {
      console.error('Failed to save custom meal', error)
      if (axios.isAxiosError(error) && error.response?.data?.error) {
        setError(error.response.data.error)
      } else {
        setError('Could not save this custom meal.')
      }
      formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } finally {
      setIsSaving(false)
    }
  }

  const saveCustomMeal = async () => {
    setError(null)

    const validationErrors = validateDraft()

    if (Object.keys(validationErrors).length > 0) {
      showFieldErrors(validationErrors)
      return
    }

    setIsCheckingConsent(true)
    setFieldErrors({})

    try {
      const response = await axios.get<PreferenceData>(
        "/preferences",
        {
          headers: {
            Authorization: `Bearer ${token}`
          },
        },
      )

      const latestConsent = response.data.dataSharingConsent

      if (latestConsent === null) {
        setIsConsentModalOpen(true)
        return 
      }

      await createCustomMeal()
    } catch (error) {
      console.error('Failed to check data sharing consent', error)
      setError('Could not check your data sharing preference.')
      formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } finally {
      setIsCheckingConsent(false)
    }
  }

  const saveConsentAndCreateMeal = async (consent: boolean) => {
    setError(null)
    setConsentChoiceBeingSaved(consent)

    try {
      await axios.put(
        "/preferences/data-sharing",
        {
          dataSharingConsent: consent,
        },
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      setIsConsentModalOpen(false)

      await createCustomMeal()
    } catch (error) {
      console.error('Failed to save data sharing consent', error)
      setError('Could not save your data sharing preference.')
      formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } finally {
      setConsentChoiceBeingSaved(null)
    }
  }
  

  return (
    <Box className="ui-settings-page ui-meal-add-page">
      <CustomMealConsentModal
        opened={isConsentModalOpen}
        consentChoiceBeingSaved={consentChoiceBeingSaved}
        onClose={() => setIsConsentModalOpen(false)}
        onSaveConsent={(consent) => {
          void saveConsentAndCreateMeal(consent)
        }}
      />
      <Box component="main" className="ui-settings-frame">
        <MainNav active="recommendation" />

        <Box className="ui-meal-add-header">
          <Title order={1}>Enter the details for your custom meal item</Title>
        </Box>

        <Box ref={formRef} component="section" className="ui-custom-meal-form ui-surface">
          {error && <Alert color="red" className="ui-custom-meal-error">{error}</Alert>}

          <CustomMealBasicFields
            draft={draft}
            fieldErrors={fieldErrors}
            inputClassNames={inputClassNames}
            isAutocompleting={isAutocompleting}
            onUpdateDraft={updateDraft}
            onAutocomplete={autocompleteCustomMeal}
          />

          <Box className="ui-custom-meal-fields">
            <CustomMealTagFields
              draft={draft}
              fieldErrors={fieldErrors}
              restrictedMealCategoryTags={restrictedMealCategoryTags}
              onToggleDietaryTag={toggleDietaryTag}
              onToggleMealCategoryTag={toggleMealCategoryTag}
            />

            <CustomMealLocationFields
              draft={draft}
              fieldErrors={fieldErrors}
              inputClassNames={inputClassNames}
              mealImage={mealImage}
              stateOptions={stateOptions}
              onUpdateDraft={updateDraft}
              onMealImageChange={(file) => {
                clearFieldError('image')
                setError(null)
                setMealImage(file)
              }}
            />
          </Box>

          <CustomMealFormActions
            isSaving={isSaving}
            isCheckingConsent={isCheckingConsent}
            onBack={() => navigate(`/meals/add/${mealTime}`)}
            onSave={saveCustomMeal}
          />
        </Box>
      </Box>
    </Box>
  )
}
