import {
  Alert,
  Badge,
  Box,
  Button,
  Group,
  Loader,
  SimpleGrid,
  Text,
  Title,
} from '@mantine/core'
import axios from 'axios'
import { Link } from 'react-router-dom'
import { useEffect, useMemo, useState } from 'react'

import type { MealLogItem, MealLogMonthResponse } from './mealLogTypes'
import {
  formatKcal,
  formatMealTime,
  formatMonthTitle,
  formatRM,
  getNextMonthKey,
  getPreviousMonthKey,
  groupLogsByDay,
  sumCalories,
  sumPrice,
  toMonthKey,
} from './mealLogHelpers'
import './MealLogPage.css'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL


export default function MealLogPage() {
	// State
	const [month, setMonth] = useState(() => toMonthKey(new Date()))
	const [data, setData] = useState<MealLogMonthResponse | null>(null)
	const [isLoading, setIsLoading] = useState(false)
	const [error, setError] = useState<string | null>(null)


	// Fetch
	useEffect(() => {
		const request = new AbortController()
		const token = localStorage.getItem('token')

		const loadLogs = async () => {
			setIsLoading(true)
			setError(null)

			try {
				const response = await axios.get<MealLogMonthResponse>(
					`${API_BASE_URL}/meal-logs`,
					{
						params: { month },
						headers: {
							Authorization: `Bearer ${token}`,
						},
						signal: request.signal, // connect to AbortControlller
					},
				)

				setData(response.data)
			} catch (error) {
				if (axios.isCancel(error)) {
					return
				}

				console.error('Failed to load meal logs', error)
				setError('Could not load meal logs')
			} finally {
				if (!request.signal.aborted) {
					setIsLoading(false)
				}
			}
		}

		void loadLogs()

		return () => request.abort()
	}, [month])

	const groupedLogs = useMemo(() => {
		return groupLogsByDay(data?.items ?? [])
	}, [data])

	const dayEntries = (Object.entries(groupedLogs) as Array<[string, MealLogItem[]]>).sort(
		([left], [right]) => Number(right) - Number(left),
	)

	const date: Date = new Date()
	const shortDay: string = date.toLocaleDateString('en-US', { weekday: 'short' })

	return (
		<Box className='ui-settings-page ui-meal-log-page'>
			<Box component="main" className='ui-settings-frame'>
				<nav className='ui-settings-nav ui-surface' aria-label="Main navigation">
					<Link to="/recommendation">Recommendation</Link>
					<Link to="/meal-logs" aria-current="page">Logs</Link>
					<Link to="/preferences">Preferences</Link>
				</nav>

				<Box component="section" className='ui-meal-log-shell ui-surface'>
					<Group className="ui-meal-log-monthbar" justify="space-between">
						<Button
							variant="subtle"
							className='ui-meal-log-button'
							onClick={() => setMonth(getPreviousMonthKey(month))}	
						>
							{'< Previous Month'}
						</Button>

						<Title order={1}>Meal Log - {formatMonthTitle(month)}</Title>

						<Button
							variant="subtle"
							className="ui-meal-log-button"
							onClick={() => setMonth(getNextMonthKey(month))}
						>{'> Next Month'}</Button>
					</Group>

					{error && <Alert color="red">{error}</Alert>}

					{isLoading && (
						<Box className='ui-meal-log-loading'>
							<Loader color='green' />
							<Text fw={900}>Loading meal logs...</Text>
						</Box>
					)}

					{!isLoading && data && (
						<>
							<SimpleGrid cols={{ base: 2, md: data?.summary.showBudgetRemaining ? 4 : 3}} className='ui-meal-log-summary'>
								<Box>
									<Text>Total Meals</Text>
									<strong>{data.summary.totalMealsEaten}</strong>
								</Box>

								<Box>
									<Text>Total Spent</Text>
									<strong>{formatRM(data.summary.totalSpent)}</strong>
								</Box>
								
								<Box>
									<Text>Total Calories</Text>
									<strong>{formatKcal(data.summary.totalCalories)}</strong>
								</Box>
								{data.summary.showBudgetRemaining && (
									<Box>
										<Text>Budget Remaining</Text>
										<strong className='ui-meal-log-budget'>
											{formatRM(data.summary.budgetRemaining ?? 0)}
										</strong>
									</Box>
								)}
							</SimpleGrid>

							<Box className='ui-meal-log-divider' />

							{data.items.length === 0 ? (
								<Box className='ui-meal-log-empty'>
									<Text fw={900}>No meals logged for this month</Text>
									<Text className='ui-field-copy'>Meals you log from recommendations or search will appear here.</Text>
								</Box>
							) : (
								<Box className='ui-meal-log-days'>
									{dayEntries.map(([day, items]) => (
										<Box className='ui-meal-log-day' key={day}>
											<Box className='ui-meal-log-day-header'>
												<Box style={{
													display: 'flex',
													alignItems: 'center',
													gap: '8px',
												}}>
												<Text fw={900} style={{fontSize: 20}}>{String(day).padStart(2, '0')}</Text>
												<Badge variant='light' color='rgb(231, 139, 0)'>{shortDay}</Badge>
												</Box>
												<Text></Text>
												<Text fw={900} style={{textAlign: 'left'}}>{formatKcal(sumCalories(items))}</Text>
												<Text fw={900} style={{textAlign: 'right'}}>{formatRM(sumPrice(items))}</Text>
											</Box>

											{items.map((item) => (
												<Box className='ui-meal-log-row' key={item.id}>
													<Text>{formatMealTime(item.eatenAt)}</Text>
													<Text fw={900}>{item.mealName}</Text>
													<Text>{formatKcal(item.calories)}</Text>
													<Text>{formatRM(item.price)}</Text>
												</Box>
											))}
										</Box>
									))}
								</Box>
							)}
						</>
					)}
				</Box>
			</Box>
		</Box>
	)
}
