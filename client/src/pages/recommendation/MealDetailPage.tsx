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
  findPersistedRecommendationLocation,
} from './recommendationStorage'
import { resolveWikipediaMealImage } from './wikiMealImages'
import type {
  CandidateScoreBreakdown,
  MatchedMealCandidate,
  MealCategory,
  MealDetailResponse,
  RestaurantResult,
} from './recommendationTypes.ts'
import './MealDetailPage.css'
import { ChefHat, Gauge, Soup } from 'lucide-react'
import { FiCheck } from 'react-icons/fi'
import MainNav from '@/theme/MainNav'



const validMealCategories: MealCategory[] = ['breakfast', 'lunch', 'dinner', 'snack']

const isMealCategory = (value: string | undefined): value is MealCategory => {
    return validMealCategories.includes(value as MealCategory)
}

const formatRM = (value: number) => `RM${value.toFixed(2)}`
const formatMacro = (value: number) => `${Number(value.toFixed(1))}g`
const formatRecommendationScore = (score: number | undefined) => {
	if (typeof score !== 'number' || Number.isNaN(score)) {
		return null
	}
	return Math.round(score).toString()
}

const scoreBreakdownItems = (breakdown: CandidateScoreBreakdown) => [
	{
		label: 'Goal alignment',
		value: breakdown.goal_alignment,
		description: 'Nutrition fit for your selected goal',
	},
	{
		label: 'Budget fit',
		value: breakdown.budget_fit,
		description: 'Price fit against your per-meal budget',
	},
	{
		label: 'Recency',
		value: breakdown.recency_penalty,
		description: 'Variety boost from recent meal history',
	},
	{
		label: 'Preference',
		value: breakdown.preference,
		description: 'Match with your preferred meal tags',
	},
]

// Custom meals can have a fixed price, while generated meals usually return a range.
const formatPriceRange = (min: number, max: number) => {
	if (min === max) {
		return formatRM(min)
	}

	return `${formatRM(min)} - ${formatRM(max)}`
}

export function RestaurantCard({ restaurant }: { restaurant: RestaurantResult }) {
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
					{restaurant.source === 'user_recommended' &&( 
						<Badge variant='light' color='yellow'>
							User recommended
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

function RecommendationScorePanel({
	score,
	breakdown,
}: {
	score: number
	breakdown: CandidateScoreBreakdown
}) {
	return (
		<Box className="ui-meal-detail-score">
			<Box className="ui-meal-detail-score-total">
				<Box className="ui-meal-detail-score-icon" aria-hidden="true">
					<Gauge size={18} />
				</Box>
				<Box className="ui-meal-detail-score-copy">
					<Text>Total recommendation score</Text>
					<Title order={3}>{Math.round(score)}</Title>
				</Box>
			</Box>

			<Box className="ui-meal-detail-score-grid">
				{scoreBreakdownItems(breakdown).map((item) => (
					<Box className="ui-meal-detail-score-item" key={item.label}>
						<Text>{item.label}</Text>
						<Title order={4}>{Math.round(item.value)}</Title>
						<Text>{item.description}</Text>
					</Box>
				))}
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

	const recommendationLocation = useMemo(() => {
		if (!isMealCategory(mealCategory)) {
			return ''
		}

		return findPersistedRecommendationLocation(storageKey, mealCategory)
	}, [mealCategory, storageKey])

	const customMealPrice =
		candidate?.food.source === 'custom' && typeof candidate.food.price === 'number'
			? candidate.food.price
			: null

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
				"/recommendations/meal-detail",
				{
					mealCategory,
					candidate,
					location: recommendationLocation,
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
					servingDescription:
						response.data.meal.servingDescription ||
						candidate.food.serving_description,
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
	}, [candidate, mealCategory, recommendationLocation])

	useEffect(() => {
		if (!detail || detail.meal.imageUrl || candidate?.food.source === 'custom') {
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
	}, [candidate?.food.source, detail])

	const openLogModal = () => {
		if (!detail) {
			return
		}

		setMealToLog({
			source: candidate?.food.source === 'custom' ? 'custom' : 'prebuilt',
			mealId: detail.meal.id,
			name: detail.meal.name,
			calories: detail.meal.nutrition.calories,
		})
	}

	if (!isMealCategory(mealCategory) || !mealId || !candidate) {
		return (
			<Box className='ui-settings-page ui-meal-detail-page'>
				<Box component="main" className='ui-settings-frame'>
					<MainNav active="recommendation" />

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

	const recommendationScore = formatRecommendationScore(candidate.score)
	const scoreBreakdown = candidate.score_breakdown

	return (
		<Box className='ui-settings-page ui-meal-detail-page'>
			<Box component="main" className='ui-settings-frame'>
				<MainNav active="recommendation" />

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
									{detail.meal.servingDescription && (
										<Text className="ui-field-copy">{detail.meal.servingDescription}</Text>
									)}
									<SimpleGrid cols={2} spacing="sm">
										<Text>{Math.round(detail.meal.nutrition.calories)} kcal</Text>
										<Text>Fat {formatMacro(detail.meal.nutrition.fatG)}</Text>
										<Text>Protein {formatMacro(detail.meal.nutrition.proteinG)}</Text>
										<Text>Carbs {formatMacro(detail.meal.nutrition.carbsG)}</Text>
									</SimpleGrid>

									<Group gap="xs" align="center">
										<Text className="ui-meal-detail-price">
											{customMealPrice !== null
												? formatRM(customMealPrice)
												: formatPriceRange(
													detail.meal.estimatedPriceRange.min,
													detail.meal.estimatedPriceRange.max,
												)}
										</Text>
										<Badge
											variant="gradient"
											gradient={{ from: 'rgba(37, 161, 21, 1)', to: 'rgba(247, 200, 153, 1)', deg: 90 }}
										>{customMealPrice !== null ? 'Actual' : 'Estimated'}</Badge>
									</Group>

									<Group>
										{detail.meal.signals.sodiumLevel && <Badge variant='light'>Sodium {detail.meal.signals.sodiumLevel}</Badge>}
										{detail.meal.signals.sugarLevel && <Badge variant="light">Sugar {detail.meal.signals.sugarLevel}</Badge>}
										{detail.meal.signals.purineRisk && <Badge variant="light">Purine {detail.meal.signals.purineRisk}</Badge>}
									</Group>
								</Box>
							</Box>

							<Box className="ui-meal-detail-panel ui-surface">
								<Stack gap="lg">
									<Box>
										<Title order={2}>{detail.meal.name}</Title>
										{recommendationScore && scoreBreakdown && (
											<RecommendationScorePanel score={candidate.score ?? 0} breakdown={scoreBreakdown} />
										)}
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

										{detail.restaurants.length > 0 && (
											<Stack gap="sm">
												{detail.restaurants.map((restaurant) => (
													<RestaurantCard restaurant={restaurant} key={`${restaurant.name}-${restaurant.address}`}/>
												))}
											</Stack>
										)}

										{detail.restaurants.length === 0 && detail.restaurantLookupStatus === 'no_results' && (
											<Text className='ui-field-copy'>No nearby restaurant matches found.</Text>
										)}

										{detail.restaurants.length === 0 && detail.restaurantLookupStatus === 'unavailable' && (
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
