import {
  Alert,
  Badge,
  Box,
  Button,
  Group,
  Loader,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core'
import axios from 'axios'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { FiCheck } from 'react-icons/fi'
import { ChefHat } from 'lucide-react'

import '@/App.css'
import '../recommendation/MealDetailPage.css'
import LogMealModal from '../mealLog/LogMealModal'
import type { LoggableMeal } from '../mealLog/mealLogTypes'
import { resolveWikipediaMealImage } from '../recommendation/wikiMealImages'
import { RestaurantCard } from '../recommendation/MealDetailPage'
import type { RestaurantResult } from '../recommendation/recommendationTypes'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

type MealSource = 'prebuilt' | 'custom'

type SearchMealDetailResponse = {
  meal: {
    id: string
    name: string
    source: MealSource
    imageUrl?: string
    servingDescription?: string
    price?: number
    tags: string[]
    nutrition: {
      calories: number
      fatG: number
      proteinG: number
      carbsG: number
    }
  }
  location: {
    query: string
    basis: string
  }
  restaurants: RestaurantResult[]
  restaurantLookupStatus: 'ok' | 'no_results' | 'unavailable'
}

const isMealSource = (value: string | undefined): value is MealSource =>
  value === 'prebuilt' || value === 'custom'

const formatMacro = (value: number) => `${Number(value.toFixed(1))}g`
const formatRM = (value: number) => `RM ${value.toFixed(2)}`

export default function SearchMealDetailPage() {
  const navigate = useNavigate()
  const { source, mealId } = useParams()
  const token = localStorage.getItem('token')

  const [detail, setDetail] = useState<SearchMealDetailResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [mealToLog, setMealToLog] = useState<LoggableMeal | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  const loadMealDetail = async () => {
    if (!isMealSource(source) || !mealId) {
      setError('Meal detail unavailable.')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const response = await axios.get<SearchMealDetailResponse>(
        `${API_BASE_URL}/meals/${source}/${mealId}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      )

      setDetail(response.data)
    } catch (requestError) {
      console.error('Failed to load searched meal detail', requestError)

      let message = 'Could not load this meal detail right now.'
      if (axios.isAxiosError(requestError) && requestError.response?.status === 404) {
        message = 'Meal not found.'
      }
      if (axios.isAxiosError(requestError) && requestError.response?.status === 401) {
        message = 'Your session has expired. Please log in again.'
      }

      setError(message)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void loadMealDetail()

    // Route params drive this page load.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source, mealId])

  useEffect(() => {
    if (!successMessage) {
      return
    }

    const timer = window.setTimeout(() => setSuccessMessage(null), 4000)
    return () => window.clearTimeout(timer)
  }, [successMessage])

  useEffect(() => {
    if (!detail || detail.meal.source !== 'prebuilt' || detail.meal.imageUrl) {
      return
    }

    const controller = new AbortController()

    const loadImage = async () => {
      const imageUrl = await resolveWikipediaMealImage(detail.meal.name, controller.signal)
      if (!imageUrl || controller.signal.aborted) {
        return
      }

      setDetail((current) => (
        current
          ? {
              ...current,
              meal: {
                ...current.meal,
                imageUrl,
              },
            }
          : current
      ))
    }

    void loadImage()

    return () => controller.abort()
  }, [detail])

  const openLogModal = () => {
    if (!detail) {
      return
    }

    setMealToLog({
      source: detail.meal.source,
      mealId: detail.meal.id,
      name: detail.meal.name,
      calories: detail.meal.nutrition.calories,
    })
  }

  return (
    <Box className="ui-settings-page ui-meal-detail-page">
      <Box component="main" className="ui-settings-frame">
        <nav className="ui-settings-nav ui-surface" aria-label="Main navigation">
          <Link to="/recommendation">Recommendation</Link>
          <Link to="/meal-logs">Logs</Link>
          <Link to="/preferences">Preferences</Link>
        </nav>

        {successMessage && (
          <Box className="ui-success-toast" role="status" aria-live="polite">
            <FiCheck className="ui-success-toast-icon" aria-hidden="true" />
            <Text fw={900}>{successMessage}</Text>
          </Box>
        )}

        {isLoading && (
          <Box className="ui-meal-detail-loading ui-card">
            <Loader color="green" />
            <Text fw={900}>Loading meal detail...</Text>
          </Box>
        )}

        {error && !isLoading && (
          <Alert color="red" className="ui-meal-detail-alert">
            <Text fw={900}>{error}</Text>
            <Button className="ui-primary-button" onClick={loadMealDetail}>
              Retry
            </Button>
          </Alert>
        )}

        {detail && !isLoading && !error && (
          <Box className="ui-meal-detail-layout ui-manual-meal-detail-layout">
            <Box className="ui-meal-detail-left">
              <Text
                component="button"
                type="button"
                onClick={() => navigate(-1)}
                className="ui-meal-detail-back ui-meal-detail-back-button"
              >
                Back to search
              </Text>

              <Title order={1}>Meal Detail</Title>

              <Box className="ui-meal-detail-photo" aria-hidden={!detail.meal.imageUrl}>
                {detail.meal.imageUrl ? (
                  <img
                    src={detail.meal.imageUrl}
                    alt={detail.meal.name}
                    referrerPolicy="no-referrer"
                  />
                ) : (
                  <ChefHat size={38} />
                )}
              </Box>

              <Box className="ui-meal-detail-meta ui-card">
                <Text fw={900}>Per serving</Text>
                {detail.meal.servingDescription && (
                  <Text className="ui-field-copy">{detail.meal.servingDescription}</Text>
                )}
                <SimpleGrid cols={2} spacing="sm">
                  <Text>{Math.round(detail.meal.nutrition.calories)} kcal</Text>
                  <Text>Fat {formatMacro(detail.meal.nutrition.fatG)}</Text>
                  <Text>Protein {formatMacro(detail.meal.nutrition.proteinG)}</Text>
                  <Text>Carbs {formatMacro(detail.meal.nutrition.carbsG)}</Text>
                </SimpleGrid>

                {typeof detail.meal.price === 'number' && detail.meal.price > 0 && (
                  <Text className="ui-meal-detail-price">{formatRM(detail.meal.price)}</Text>
                )}

                {detail.meal.tags.length > 0 && (
                  <Group gap="xs">
                    {detail.meal.tags.slice(0, 8).map((tag) => (
                      <Badge variant="light" key={tag}>{tag}</Badge>
                    ))}
                  </Group>
                )}
              </Box>
            </Box>

            <Box className="ui-meal-detail-panel ui-surface">
              <Stack gap="lg">
                <Box>
                  <Title order={2}>{detail.meal.name}</Title>
                </Box>

                <Button className="ui-primary-button ui-meal-detail-log" onClick={openLogModal}>
                  Log
                </Button>

                <Box>
                  <Text className="ui-restaurant-heading">
                    Nearby restaurants that may serve this food
                  </Text>

                  {detail.location.query && (
                    <Text className="ui-restaurant-location">
                      Near {detail.location.query}
                    </Text>
                  )}

                  {detail.restaurantLookupStatus === 'ok' && detail.restaurants.length > 0 && (
                    <Stack gap="sm">
                      {detail.restaurants.map((restaurant) => (
                        <RestaurantCard
                          restaurant={restaurant}
                          key={`${restaurant.name}-${restaurant.address}`}
                        />
                      ))}
                    </Stack>
                  )}

                  {detail.restaurantLookupStatus === 'no_results' && (
                    <Text className="ui-field-copy">No nearby restaurant matches found.</Text>
                  )}

                  {detail.restaurantLookupStatus === 'unavailable' && (
                    <Text className="ui-field-copy">Restaurant lookup is unavailable right now.</Text>
                  )}
                </Box>
              </Stack>
            </Box>
          </Box>
        )}
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
