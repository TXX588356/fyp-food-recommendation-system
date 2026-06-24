import {
  Alert,
  Badge,
  Box,
  Button,
  Checkbox,
  FileInput,
  Group,
  NumberInput,
  Select,
  SimpleGrid,
  Text,
  TextInput,
  Title,
} from '@mantine/core'
import axios from 'axios'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { dietaryRestrictionOptions, mealCategoryOptions } from '@/preferences/options'
import './CustomMealAddPage.css'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

type MealTime = 'breakfast' | 'lunch' | 'dinner' | 'snack'

type CustomMealResponse = {
  id: string
  name: string
  price: number
  calories: number
  fatG: number
  proteinG: number
  carbsG: number
  state: string
  district: string
  restaurantName: string
  imageURL: string
  dietaryRestrictionTags: string[]
  mealCategoryTags: string[]
  isOwner: boolean
  isShared: boolean
}

type MealSearchResult = {
  id: string
  name: string
  source: 'prebuilt' | 'custom'
  tags: string[]
  calories: number
  fat_g: number
  protein_g: number
  carbs_g: number
  price?: number
  image_url?: string
}

type ExistingMeal = {
  id: string
  name: string
  calories: number
  priceLabel: string
  tags: string[]
  source: 'prebuilt' | 'custom'
  imageUrl?: string
}

type CustomMealDraft = {
  name: string
  price: number | ''
  calories: number | ''
  carbsG: number | ''
  fatG: number | ''
  proteinG: number | ''
  state: string
  district: string
  restaurantName: string
  dietaryRestrictionTags: string[]
  mealCategoryTags: string[]
  mealURL: string
}

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

const customMealDietaryOptions = dietaryRestrictionOptions.filter((option) => option.value !== 'none')

const validateMealCategoryTags = (mealCategoryTags: string[]) =>
  mealCategoryTags.length === 0 ? 'Please select at least one meal category.' : null

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
  mealURL: '',
}

const isMealTime = (value: string | undefined): value is MealTime =>
  value === 'breakfast' || value === 'lunch' || value === 'dinner' || value === 'snack'

const formatRM = (value: number) => `RM ${value.toFixed(2)}`

function MealAddIcon({ name }: { name: 'search' | 'plus' | 'bowl' }) {
  const commonProps = {
    width: 22,
    height: 22,
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 2,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  }

  if (name === 'search') {
    return (
      <svg {...commonProps}>
        <circle cx="11" cy="11" r="7" />
        <path d="m20 20-3.5-3.5" />
      </svg>
    )
  }

  if (name === 'plus') {
    return (
      <svg {...commonProps}>
        <path d="M12 5v14" />
        <path d="M5 12h14" />
      </svg>
    )
  }

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

function MainNav({ active }: { active: 'recommendation' | 'log' | 'preferences' }) {
  return (
    <nav className="ui-settings-nav ui-surface" aria-label="Main navigation">
      <Link to="/recommendation" aria-current={active === 'recommendation' ? 'page' : undefined}>
        Recommendations
      </Link>
      <Link to="/meal-logs" aria-current={active === 'log' ? 'page' : undefined}>
        Log
      </Link>
      <Link to="/preferences" aria-current={active === 'preferences' ? 'page' : undefined}>
        Preferences
      </Link>
    </nav>
  )
}

function toExistingMeal(meal: MealSearchResult): ExistingMeal {
  return {
    id: `${meal.source}-${meal.id}`,
    name: meal.name,
    calories: meal.calories,
    priceLabel: meal.price === undefined ? 'Prebuilt data' : formatRM(meal.price),
    tags: meal.tags,
    source: meal.source,
    imageUrl: meal.image_url,
  }
}

export function CustomMealSearchPage() {
  const navigate = useNavigate()
  const { mealCategory } = useParams()
  const mealTime = isMealTime(mealCategory) ? mealCategory : 'breakfast'
  const [query, setQuery] = useState('')
  const [visibleMeals, setVisibleMeals] = useState<ExistingMeal[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const token = localStorage.getItem('token')

  useEffect(() => {
    const normalizedQuery = query.trim()

    if (!normalizedQuery) {
      setVisibleMeals([])
      setIsLoading(false)
      setError(null)
      return
    }

    const request = new AbortController()

    const loadMeals = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await axios.get<MealSearchResult[]>(
          `${API_BASE_URL}/meals/search`,
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

  return (
    <Box className="ui-settings-page ui-meal-add-page">
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
              onChange={(event) => setQuery(event.currentTarget.value)}
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
            <strong>Add my custom meal item</strong>
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
                <Box className="ui-meal-card ui-card" key={meal.id}>
                  <Box className="ui-meal-photo" aria-hidden={!meal.imageUrl}>
                    {meal.imageUrl ? (
                      <img src={meal.imageUrl} alt={meal.name} loading="lazy" />
                    ) : (
                      <MealAddIcon name="bowl" />
                    )}
                  </Box>

                  <Box className="ui-meal-summary">
                    <Group gap="xs" align="center">
                      <Title order={2}>{meal.name}</Title>
                      {meal.source === 'custom' && (
                        <Badge className="ui-meal-source-badge">Community</Badge>
                      )}
                    </Group>
                    <Text>{Math.round(meal.calories)} kcal</Text>
                    <Text className="ui-meal-price">{meal.priceLabel}</Text>
                  </Box>

                  <Button className="ui-meal-log-button" variant="subtle">
                    Log
                  </Button>
                </Box>
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
  const mealTime = isMealTime(mealCategory) ? mealCategory : 'breakfast'
  const [draft, setDraft] = useState<CustomMealDraft>(emptyDraft)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mealImage, setMealImage] = useState<File | null>(null)

  const token = localStorage.getItem('token')

  const updateDraft = <Key extends keyof CustomMealDraft>(key: Key, value: CustomMealDraft[Key]) => {
    setDraft((current) => ({
      ...current,
      [key]: value,
    }))
  }

  const toggleDietaryTag = (tag: string) => {
    setDraft((current) => {
      const hasTag = current.dietaryRestrictionTags.includes(tag)

      return {
        ...current,
        dietaryRestrictionTags: hasTag
          ? current.dietaryRestrictionTags.filter((currentTag) => currentTag !== tag)
          : [...current.dietaryRestrictionTags, tag],
      }
    })
  }

  const toggleMealCategoryTag = (tag: string) => {
    setDraft((current) => {
      const hasTag = current.mealCategoryTags.includes(tag)

      return {
        ...current,
        mealCategoryTags: hasTag
          ? current.mealCategoryTags.filter((currentTag) => currentTag !== tag)
          : [...current.mealCategoryTags, tag],
      }
    })
  }

  const saveCustomMeal = async () => {
    setError(null)

    if (!draft.name.trim()) {
      setError('Please enter a meal name.')
      return
    }

    if (
      draft.price === '' ||
      draft.calories === '' ||
      draft.carbsG === '' ||
      draft.fatG === '' ||
      draft.proteinG === '' ||
      !draft.state ||
      !draft.district.trim() ||
      !draft.restaurantName.trim()
    ) {
      setError('Please complete the custom meal details.')
      return
    }

    const mealCategoryError = validateMealCategoryTags(draft.mealCategoryTags)

    if (mealCategoryError) {
      setError(mealCategoryError)
      return
    }

    setIsSaving(true)

    try {
      await axios.post<CustomMealResponse>(
        `${API_BASE_URL}/custom-meals`,
        {
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
          mealURL: draft.mealURL,
        },
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      navigate(`/meals/add/${mealTime}`)
    } catch (error) {
      console.error('Failed to save custom meal', error)

      if (axios.isAxiosError(error) && error.response?.data?.error) {
        setError(error.response.data.error)
      } else {
        setError('Could not save this custom meal.')
      }
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Box className="ui-settings-page ui-meal-add-page">
      <Box component="main" className="ui-settings-frame">
        <MainNav active="recommendation" />

        <Box className="ui-meal-add-header">
          <Title order={1}>Enter the details for your custom meal item</Title>
        </Box>

        <Box component="section" className="ui-custom-meal-form ui-surface">
          
          <Box className="ui-custom-meal-fields">
          {error && <Alert color="red" className="ui-custom-meal-error">{error}</Alert>}
            <TextInput
              classNames={{ input: 'ui-input' }}
              placeholder="Meal Name"
              value={draft.name}
              onChange={(event) => updateDraft('name', event.currentTarget.value)}
            />

            <NumberInput
              classNames={{ input: 'ui-input' }}
              min={0}
              decimalScale={2}
              placeholder="Price (RM)"
              value={draft.price}
              onChange={(value) => updateDraft('price', value === '' ? '' : Number(value))}
            />

            <Box className="ui-serving-size">
              <Text fw={900}>Nutritional info</Text>
              <Group gap="sm" align="center">
                <NumberInput
                  hideControls
                  classNames={{ input: 'ui-input ui-short-input' }}
                  min={0}
                  decimalScale={0}
                  value={draft.calories}
                  onChange={(value) => updateDraft('calories', value === '' ? '' : Number(value))}
                />
                <Text fw={900}>kcal</Text>
              </Group>

              <SimpleGrid cols={{ base: 1, xs: 3 }} spacing="sm">
                <NumberInput
                  classNames={{ input: 'ui-input' }}
                  min={0}
                  decimalScale={1}
                  placeholder="Carbs"
                  rightSection={<Text fw={900}>g</Text>}
                  value={draft.carbsG}
                  onChange={(value) => updateDraft('carbsG', value === '' ? '' : Number(value))}
                />
                <NumberInput
                  classNames={{ input: 'ui-input' }}
                  min={0}
                  decimalScale={1}
                  placeholder="Fat"
                  rightSection={<Text fw={900}>g</Text>}
                  value={draft.fatG}
                  onChange={(value) => updateDraft('fatG', value === '' ? '' : Number(value))}
                />
                <NumberInput
                  classNames={{ input: 'ui-input' }}
                  min={0}
                  decimalScale={1}
                  placeholder="Protein"
                  rightSection={<Text fw={900}>g</Text>}
                  value={draft.proteinG}
                  onChange={(value) => updateDraft('proteinG', value === '' ? '' : Number(value))}
                />
              </SimpleGrid>
              <Text>Not sure? Let AI generate for you!</Text>
            </Box>
          </Box>

          <Box className="ui-custom-meal-fields">
            <Box className="ui-dietary-tag-group">
              <Text fw={900}>Dietary tags (optional)</Text>
              <SimpleGrid cols={{ base: 1, xs: 2, sm: 3 }} spacing="sm">
                {customMealDietaryOptions.map((option) => (
                  <Checkbox
                    key={option.value}
                    label={option.label}
                    checked={draft.dietaryRestrictionTags.includes(option.value)}
                    onChange={() => toggleDietaryTag(option.value)}
                  />
                ))}
              </SimpleGrid>
            </Box>

            <Box
              className="ui-dietary-tag-group"
              role="group"
              aria-labelledby="meal-category-tags-label"
              aria-required="true"
            >
              <Box>
                <Text id="meal-category-tags-label" fw={900}>
                  Meal category tags <Text component="span" c="red">*</Text>
                </Text>
                <Text size="sm" c="dimmed">Select at least one.</Text>
              </Box>
              <SimpleGrid cols={{ base: 1, xs: 2, sm: 3 }} spacing="sm">
                {mealCategoryOptions.map((option) => (
                  <Checkbox
                    key={option.value}
                    label={option.label}
                    checked={draft.mealCategoryTags.includes(option.value)}
                    onChange={() => toggleMealCategoryTag(option.value)}
                  />
                ))}
              </SimpleGrid>
            </Box>

            <Box className="ui-location-fields">
              <Text fw={900}>Enter location where you had this meal</Text>
              <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="sm">
                <Select
                  label="State"
                  classNames={{ input: 'ui-input' }}
                  data={stateOptions}
                  value={draft.state}
                  onChange={(value) => updateDraft('state', value ?? '')}
                />
                <TextInput
                  label="District"
                  classNames={{ input: 'ui-input' }}
                  value={draft.district}
                  onChange={(event) => updateDraft('district', event.currentTarget.value)}
                />
              </SimpleGrid>
              <TextInput
                label="Restaurant Name"
                classNames={{ input: 'ui-input' }}
                value={draft.restaurantName}
                onChange={(event) => updateDraft('restaurantName', event.currentTarget.value)}
              />
              <FileInput
                classNames={{ input: 'ui-input' }}
                clearable 
                accept="image/png,image/jpeg" 
                label="Upload meal image" 
                value={mealImage}
                onChange={setMealImage}
              />
            </Box>
          </Box>

          <Group justify="space-between" className="ui-custom-meal-actions">
            <Button
              className="ui-dark-button"
              onClick={() => navigate(`/meals/add/${mealTime}`)}
            >
              Back
            </Button>

            <Button
              className="ui-dark-button"
              loading={isSaving}
              onClick={saveCustomMeal}
            >
              Save
            </Button>
          </Group>
        </Box>
      </Box>
    </Box>
  )
}
