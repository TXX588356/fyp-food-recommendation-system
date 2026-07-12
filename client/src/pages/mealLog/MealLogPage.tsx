import {
  ActionIcon,
  Alert,
  Badge,
  Box,
  Button,
  Group,
  Loader,
  Modal,
  NumberInput,
  SimpleGrid,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core'
import axios from 'axios'
import { Link } from 'react-router-dom'
import { useCallback, useEffect, useMemo, useState } from 'react'

import type { MealLogItem, MealLogMonthResponse, MealLogUpdateInput } from './mealLogTypes'
import {
  fromDateTimeLocalValue,
  formatKcal,
  formatMealTime,
  formatMonthTitle,
  formatRM,
  getNextMonthKey,
  getPreviousMonthKey,
  groupLogsByDay,
  sumCalories,
  sumPrice,
  toDateTimeLocalValue,
  toMonthKey,
} from './mealLogHelpers'
import './MealLogPage.css'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

function EditIcon() {
	return (
		<svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
			<path d="M4 20h4.7L19.4 9.3a2.1 2.1 0 0 0 0-3l-1.7-1.7a2.1 2.1 0 0 0-3 0L4 15.3V20Zm3.8-2H6v-1.8l8.7-8.7 1.8 1.8L7.8 18ZM18 7.9l-1.8-1.8.8-.8 1.8 1.8-.8.8Z" />
		</svg>
	)
}

function DeleteIcon() {
	return (
		<svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
			<path d="M8 21a2 2 0 0 1-2-2V8H5a1 1 0 1 1 0-2h4V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v1h4a1 1 0 1 1 0 2h-1v11a2 2 0 0 1-2 2H8Zm8-13H8v11h8V8Zm-5 8a1 1 0 1 1-2 0v-5a1 1 0 1 1 2 0v5Zm4 0a1 1 0 1 1-2 0v-5a1 1 0 1 1 2 0v5ZM11 6h2V5h-2v1Z" />
		</svg>
	)
}

export default function MealLogPage() {
	// State
	const [month, setMonth] = useState(() => toMonthKey(new Date()))
	const [data, setData] = useState<MealLogMonthResponse | null>(null)
	const [isLoading, setIsLoading] = useState(false)
	const [error, setError] = useState<string | null>(null)
	const [successMessage, setSuccessMessage] = useState<string | null>(null)
	const [editingLog, setEditingLog] = useState<MealLogItem | null>(null)
	const [deletingLog, setDeletingLog] = useState<MealLogItem | null>(null)
	const [editPrice, setEditPrice] = useState<number | ''>('')
	const [editEatenAt, setEditEatenAt] = useState('')
	const [editError, setEditError] = useState<string | null>(null)
	const [deleteError, setDeleteError] = useState<string | null>(null)
	const [isSavingEdit, setIsSavingEdit] = useState(false)
	const [isDeleting, setIsDeleting] = useState(false)

	const loadLogs = useCallback(async (signal?: AbortSignal) => {
		const token = localStorage.getItem('token')

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
					signal,
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
			if (!signal?.aborted) {
				setIsLoading(false)
			}
		}
	}, [month])

	// Fetch
	useEffect(() => {
		const request = new AbortController()

		void loadLogs(request.signal)

		return () => request.abort()
	}, [loadLogs])

	useEffect(() => {
		if (!successMessage) {
			return
		}

		const dismissTimer = window.setTimeout(() => {
			setSuccessMessage(null)
		}, 4000)

		return () => window.clearTimeout(dismissTimer)
	}, [successMessage])

	const openEditModal = (item: MealLogItem) => {
		setEditingLog(item)
		setEditPrice(item.price)
		setEditEatenAt(toDateTimeLocalValue(new Date(item.eatenAt)))
		setEditError(null)
	}

	const closeEditModal = () => {
		if (isSavingEdit) {
			return
		}

		setEditingLog(null)
		setEditError(null)
	}

	const saveEditedLog = async () => {
		if (!editingLog) {
			return
		}

		if (editPrice === '' || Number(editPrice) < 0) {
			setEditError('Please enter a valid price')
			return
		}

		if (!editEatenAt) {
			setEditError('Please choose when you ate this meal')
			return
		}

		const token = localStorage.getItem('token')
		const payload: MealLogUpdateInput = {
			price: Number(editPrice),
			eatenAt: fromDateTimeLocalValue(editEatenAt),
		}

		setIsSavingEdit(true)
		setEditError(null)

		try {
			await axios.patch(`${API_BASE_URL}/meal-logs/${editingLog.id}`, payload, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			})

			setEditingLog(null)
			setSuccessMessage(`${editingLog.mealName} updated`)
			await loadLogs()
		} catch (error) {
			console.error('Failed to update meal log', error)
			setEditError('Could not update this meal log')
		} finally {
			setIsSavingEdit(false)
		}
	}

	const openDeleteModal = (item: MealLogItem) => {
		setDeletingLog(item)
		setDeleteError(null)
	}

	const closeDeleteModal = () => {
		if (isDeleting) {
			return
		}

		setDeletingLog(null)
		setDeleteError(null)
	}

	const deleteLog = async () => {
		if (!deletingLog) {
			return
		}

		const token = localStorage.getItem('token')

		setIsDeleting(true)
		setDeleteError(null)

		try {
			await axios.delete(`${API_BASE_URL}/meal-logs/${deletingLog.id}`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			})

			setDeletingLog(null)
			setSuccessMessage(`${deletingLog.mealName} deleted`)
			await loadLogs()
		} catch (error) {
			console.error('Failed to delete meal log', error)
			setDeleteError('Could not delete this meal log')
		} finally {
			setIsDeleting(false)
		}
	}

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
			{successMessage && (
				<Box
					className="ui-meal-log-success-toast"
					role="status"
					aria-live="polite"
				>
					<span aria-hidden="true">✓</span>
					<Text fw={900}>{successMessage}</Text>
				</Box>
			)}

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
												<Text fw={900} style={{textAlign: 'right'}}>{formatKcal(sumCalories(items))}</Text>
												<Text fw={900} style={{textAlign: 'right'}}>{formatRM(sumPrice(items))}</Text>
												<Text className="ui-meal-log-actions-heading" aria-hidden="true"></Text>
											</Box>

											{items.map((item) => (
												<Box className='ui-meal-log-row' key={item.id}>
													<Text>{formatMealTime(item.eatenAt)}</Text>
													<Text fw={900}>{item.mealName}</Text>
													<Text>{formatKcal(item.calories)}</Text>
													<Text>{formatRM(item.price)}</Text>
													<Group className="ui-meal-log-row-actions" gap={6} justify="flex-end">
														<ActionIcon
															aria-label={`Edit ${item.mealName}`}
															variant="subtle"
															size="sm"
															className="ui-meal-log-action-button"
															onClick={() => openEditModal(item)}
														>
															<EditIcon />
														</ActionIcon>

														<ActionIcon
															aria-label={`Delete ${item.mealName}`}
															variant="subtle"
															color="red"
															size="sm"
															className="ui-meal-log-action-button ui-meal-log-delete-button"
															onClick={() => openDeleteModal(item)}
														>
															<DeleteIcon />
														</ActionIcon>
													</Group>
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

			<Modal
				opened={editingLog !== null}
				onClose={closeEditModal}
				centered
				title="Edit meal log"
				classNames={{
					content: 'ui-meal-log-modal',
					title: 'ui-meal-log-modal-title',
				}}
			>
				{editingLog && (
					<Stack gap="md">
						<Stack gap={4}>
							<Text fw={900} style={{fontSize: 24}}>{editingLog.mealName}</Text>
							<Text size="sm" c="dimmed">{formatKcal(editingLog.calories)}</Text>
						</Stack>

						{editError && <Text c="red" fw={800}>{editError}</Text>}

						<NumberInput
							label="Price (RM)"
							min={0}
							decimalScale={2}
							value={editPrice}
							onChange={(value) => setEditPrice(value === '' ? '' : Number(value))}
							classNames={{ input: 'ui-input'}}
						/>

						<TextInput
							label="Time eaten"
							type="datetime-local"
							value={editEatenAt}
							onChange={(event) => setEditEatenAt(event.currentTarget.value)}
							classNames={{ input: 'ui-input'}}
						/>

						<Group justify="flex-end">
							<Button variant="subtle" onClick={closeEditModal} disabled={isSavingEdit}>
								Cancel
							</Button>
							<Button className="ui-primary-button" loading={isSavingEdit} onClick={saveEditedLog}>
								Save changes
							</Button>
						</Group>
					</Stack>
				)}
			</Modal>

			<Modal
				opened={deletingLog !== null}
				onClose={closeDeleteModal}
				centered
				title="Delete meal log"
				classNames={{
					content: 'ui-meal-log-modal',
					title: 'ui-meal-log-modal-title',
				}}
			>
				{deletingLog && (
					<Stack gap="md">
						<Text>
							Delete <strong>{deletingLog.mealName}</strong> from your meal log?
						</Text>

						{deleteError && <Text c="red" fw={800}>{deleteError}</Text>}

						<Group justify="flex-end">
							<Button variant="subtle" onClick={closeDeleteModal} disabled={isDeleting}>
								Cancel
							</Button>
							<Button color="red" loading={isDeleting} onClick={deleteLog}>
								Delete
							</Button>
						</Group>
					</Stack>
				)}
			</Modal>
		</Box>
	)
}
