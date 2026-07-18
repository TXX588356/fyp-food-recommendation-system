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
import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import '@/App.css'
import LogMealModal from '../mealLog/LogMealModal'
import type { LoggableMeal } from '../mealLog/mealLogTypes'
import {
  buildRecommendationStorageKey,
  findPersistedRecommendationCandidate,
} from './recommendationStorage'
import { resolveWikipediaMealImage } from './wikiMealImages'
import type {
  MatchedMealCandidate,
  MealCategory,
  MealDetailResponse,
  RestaurantResult,
} from './recommendationTypes.ts'
import './MealDetailPage.css'
import { ChefHat, Soup } from 'lucide-react'
import { FiCheck } from 'react-icons/fi'


const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

const validMealCategories: MealCategory[] = ['breakfast', 'lunch', 'dinner', 'snack']

const isMealCategory = (value: string | undefined): value is MealCategory => {
    return validMealCategories.includes(value as MealCategory)
}

const formatRM = (value: number) => `RM${value.toFixed(2)}`
const formatMacro = (value: number) => `${Number(value.toFixed(1))}g`

// Custom meals can have a fixed price, while generated meals usually return a range.
const formatPriceRange = (min: number, max: number) => {
	if (min === max) {
		return formatRM(min)
	}

	return `${formatRM(min)} - ${formatRM(max)}`
}

function RestaurantCard({ restaurant }: { restaurant: RestaurantResult }) {
	const [imageFailed, setImageFailed] = useState(false)
	const shouldShowThumbnail = Boolean(restaurant.thumbnailUrl) && !imageFailed

	return (
		<Box className='ui-restaurant-row'>
			<Box className="ui-restaurant-thumb" aria-hidden={!shouldShowThumbnail}>
				{shouldShowThumbnail ? (
						<img
							src={restaurant.thumbnailUrl}
							alt=""
							loading="lazy"
							referrerPolicy="no-referrer"
							onError={() => setImageFailed(true)}
						/>
				) : (
						<Soup size={18}/>
				)}
		</Box>

		<Box className='ui-restaurant-copy'>
				<Text fw={900}>{restaurant.name}</Text>
				{restaurant.address && <Text className='ui-restaurant-address'>{restaurant.address}</Text>}

				<Group gap="xs" className='ui-restaurant-meta'>
					{restaurant.rating > 0 && <Text color='#FFEA00'>{restaurant.rating.toFixed(1)} ★</Text>}
					{restaurant.reviewCount > 0 && <Text>{restaurant.reviewCount} reviews</Text>}
          {restaurant.price && <Text>{restaurant.price}</Text>}
					{restaurant.openNow !== undefined && (
						<Badge color={restaurant.openNow ? 'green' : 'red'} variant="light">
							{restaurant.openNow ? 'Open' : 'Closed'}
						</Badge>
					)}
				</Group>

				{restaurant.sourceUrl && (
					<Button
						component='a'
						href={restaurant.sourceUrl}
						target='_blank'
						rel="noreferrer"
						variant="subtle"
						className='ui-restaurant-link'
						>
						View map
					</Button>
				)}
			</Box>
		</Box>
	)
}

export default function MealDetailPage() {
	const { user } = useAuth()
	const { mealCategory, mealId } = useParams()
	const token = localStorage.getItem('token')
	const userKey = user?.id ?? user?.email

	const [detail, setDetail] = useState<MealDetailResponse | null>(null)
	const [error, setError] = useState<string | null>(null)
	const [isLoading, setIsLoading] = useState(false)
	const [mealToLog, setMealToLog] = useState<LoggableMeal | null>(null)
	const [successMessage, setSuccessMessage] = useState<string | null>(null)

	const storageKey = useMemo(() => buildRecommendationStorageKey(userKey), [userKey])

	const candidate = useMemo<MatchedMealCandidate | null>(() => {
		if(!isMealCategory(mealCategory) || !mealId) {
			return null
		} 

		return findPersistedRecommendationCandidate(storageKey, mealCategory, mealId)
	}, [mealCategory, mealId, storageKey])

	useEffect(() => {
		if(!successMessage) {
			return
		}

		const timer = window.setTimeout(() => setSuccessMessage(null), 4000)
		return () => window.clearTimeout(timer)
	}, [successMessage])

	const loadMealDetail = async () => {
		if(!isMealCategory(mealCategory) || !candidate) {
			return
		}

		setIsLoading(true)
		setError(null)

		try {
			const response = await axios.post<MealDetailResponse>(
				`${API_BASE_URL}/recommendations/meal-detail`,
				{
					mealCategory,
					candidate,
				},
				{
					headers: {
						Authorization: `Bearer ${token}`
					}
				}
			)

			setDetail({
				...response.data,
				meal: {
					...response.data.meal,
					imageUrl: response.data.meal.imageUrl || candidate.food.image_url,
				},
			})
		} catch (requestError) {
			console.error('Failed to load meal detail', requestError)

			let message = 'Could not load this meal detail right now.'
			if (axios.isAxiosError(requestError) && requestError.response?.status === 401) {
				message = 'Your session has expired, Please log in again'
			}

			setError(message)
		} finally {
			setIsLoading(false)
		}
	}

	useEffect(() => {
		void loadMealDetail()

		// candidate is memoized from storage and should trigger reload when route changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
	}, [candidate, mealCategory])

	useEffect(() => {
		if (!detail || detail.meal.imageUrl) {
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
			source: 'prebuilt',
			mealId: detail.meal.id,
			name: detail.meal.name,
			calories: detail.meal.nutrition.calories,
		})
	}

	if (!isMealCategory(mealCategory) || !mealId || !candidate) {
		return (
			<Box className='ui-settings-page ui-meal-detail-page'>
				<Box component="main" className='ui-settings-frame'>
					<nav className='ui-settings-nav ui-surface' aria-label='Main navigation'>
						<Link to="/recommendation">Recommendation</Link>
						<Link to="/meal-logs">Logs</Link>
						<Link to="/preferences">Preferences</Link>
					</nav>

					<Box className='ui-meal-detail-empty ui-card'>
						<ChefHat size={24} />
						<Title order={1}>Meal detail unavailable</Title>
						<Text className='ui-field-copy'>
							This recommendation is not in today&apos;s saved recommendations.
						</Text>
						<Button component={Link} to="/recommendation" className='ui-primary-button'>
							Back to recommendation
						</Button>
					</Box>
				</Box>
			</Box>
		)
	}

	return (
		<Box className='ui-settings-page ui-meal-detail-page'>
			<Box component="main" className='ui-settings-frame'>
			 	<nav className='ui-settings-nav ui-surface' aria-label='Main navigation'>
					<Link to="/recommendation">Recommendation</Link>
					<Link to="/meal-logs">Logs</Link>
					<Link to="/preferences">Preferences</Link>
				</nav>

				{successMessage && (
					<Box className='ui-success-toast' role='status' aria-live='polite'>
						<FiCheck className="ui-success-toast-icon" aria-hidden="true" />
						<Text fw={900}>{successMessage}</Text>
					</Box>
				)}

				{isLoading && (
					<Box className='ui-meal-detail-loading ui-card'>
						<Loader color='green'/>
						<Text fw={900}>Loading meal detail...</Text>
					</Box>
				)}

				{error && !isLoading && (
					<Alert color='red' className='ui-meal-detail-alert'>
						<Text fw={900}>{error}</Text>
						<Button className='ui-primary-button' onClick={loadMealDetail}>
							Retry
						</Button>
					</Alert>
				)}

				{detail && !isLoading && !error && (
					<Box className='ui-meal-detail-layout'>
						<Box className='ui-meal-detail-left'>
							<Text component={Link} to="/recommendation" className='ui-meal-detail-back'>
								Back to recommendation
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
									<ChefHat size={38}/>
								)}
							</Box>

							<Box className='ui-meal-detail-meta ui-card'>
								<Text fw={900}>Per serving</Text>
								<SimpleGrid cols={2} spacing="sm">
									<Text>{Math.round(detail.meal.nutrition.calories)} kcal</Text>
									<Text>Fat {formatMacro(detail.meal.nutrition.fatG)}</Text>
                  <Text>Protein {formatMacro(detail.meal.nutrition.proteinG)}</Text>
                  <Text>Carbs {formatMacro(detail.meal.nutrition.carbsG)}</Text>
								</SimpleGrid>

								<Text className="ui-meal-detail-price">
									{formatPriceRange(
										detail.meal.estimatedPriceRange.min,
										detail.meal.estimatedPriceRange.max,
									)}
								</Text>

								<Group>
									{detail.meal.signals.sodiumLevel && <Badge variant='light'>{detail.meal.signals.sodiumLevel}</Badge>}
									{detail.meal.signals.sugarLevel && <Badge variant="light">Sugar {detail.meal.signals.sugarLevel}</Badge>}
                  {detail.meal.signals.purineRisk && <Badge variant="light">Purine {detail.meal.signals.purineRisk}</Badge>}
								</Group>
							</Box>
						</Box>

						<Box className="ui-meal-detail-panel ui-surface">
							<Stack gap="lg">
								<Box>
									<Title order={2}>{detail.meal.name}</Title>
									<Text className='ui-meal-detail-question'>Why this is recommended to me?</Text>
									<Text className='ui-meal-detail-explanation'>
										{detail.recommendationExplanation}
									</Text>
								</Box>

								<Button className='ui-primary-button ui-meal-detail-log' onClick={openLogModal}>
									Log
								</Button>

								<Box>
									<Text className='ui-restaurant-heading'>
										Nearby restaurants that may serve this food
									</Text>

									{detail.location.query && (
										<Text className='ui-restaurant-location'>
											Near {detail.location.query}
										</Text>
									)}

									{detail.restaurantLookupStatus === 'ok' && detail.restaurants.length > 0 && (
										<Stack gap="sm">
											{detail.restaurants.map((restaurant) => (
												<RestaurantCard restaurant={restaurant} key={`${restaurant.name}-${restaurant.address}`}/>
											))}
										</Stack>
									)}

									{detail.restaurantLookupStatus === 'no_results' && (
										<Text className='ui-field-copy'>No nearby restaurant matches found.</Text>
									)}

									{detail.restaurantLookupStatus === 'unavailable' && (
										<Text className='ui-field-copy'>Restaurant lookup is unavailable right now.</Text>
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
