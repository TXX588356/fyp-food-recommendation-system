	import {
	Button,
	Group,
	Modal,
	NumberInput,
	Stack,
	Text,
	TextInput,
	} from '@mantine/core'
	import axios from 'axios'
	import { useState } from 'react'
import './MealLogPage.css'

import type { LoggableMeal, MealLogInput } from './mealLogTypes'
import { formatKcal, toDateTimeLocalValue, fromDateTimeLocalValue} from './mealLogHelpers'

	const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

	type LogMealModalProps = {
	opened: boolean
	meal: LoggableMeal | null
	onClose: () => void
	onLogged: () => void
	}

	type LogMealFormProps = {
	meal: LoggableMeal
	onClose: () => void
	onLogged: () => void
	}

	function LogMealForm({
	meal,
	onClose,
	onLogged,
	}: LogMealFormProps) {
	const [price, setPrice] = useState<number | ''>('')
	const [eatenAt, setEatenAt] = useState(() => toDateTimeLocalValue(new Date()))
	const [error, setError] = useState<string | null>(null)
	const [isSaving, setIsSaving] = useState(false)

	const token = localStorage.getItem('token')

	const saveLog = async () => {
		if (price === '' || Number(price) < 0) {
				setError('Please enter a valid price')
				return
		}

		if (!eatenAt) {
				setError('Please choose when you ate this meal')
				return
		}

		const payload: MealLogInput = {
			source: meal.source,
			mealId: meal.mealId,
			price: Number(price),
			eatenAt: fromDateTimeLocalValue(eatenAt),
		}

		setIsSaving(true)
		setError(null)

		try {
			await axios.post(`${API_BASE_URL}/meal-logs`, payload, {
					headers: {
							Authorization: `Bearer ${token}`,
					},
			})

			onLogged()
			onClose()
		} catch (error) {
			console.error('Failed to log meal', error)
			setError('Could not log this meal')
		} finally {
			setIsSaving(false)
		}
	}

	return (
		<Stack gap="md" className="ui-log-meal-form">
			<Stack gap={4} className="ui-log-meal-summary">
				<Text fw={900}>{meal.name}</Text>
				<Text size="sm" c="dimmed">{formatKcal(meal.calories)}</Text>
			</Stack>

			{error && <Text c="red" fw={800}>{error}</Text>}

			<NumberInput
				label="Price (RM)"
				min={0}
				decimalScale={2}
				value={price}
				onChange={(value) => setPrice(value === '' ? '' : Number(value))}
				classNames={{ input: 'ui-input'}}
			/>

			<TextInput
				label="Time eaten"
				type="datetime-local"
				value={eatenAt}
				onChange={(event) => setEatenAt(event.currentTarget.value)}
				classNames={{ input: 'ui-input'}}
			/>

			<Group justify="flex-end" className="ui-log-meal-actions">
				<Button className="ui-primary-button" loading={isSaving} onClick={saveLog}>Save</Button>
			</Group>
		</Stack>
	)
	}

	export default function LogMealModal({
	opened,
	meal, 
	onClose,
	onLogged,
	}: LogMealModalProps) {
	return (
		<Modal
			opened={opened}
			onClose={onClose}
			overlayProps={{
				backgroundOpacity: 0.3,
			}}
			centered
			title="Log meal"
			classNames={{
				content: 'ui-meal-log-modal',
				header: 'ui-meal-log-modal-header',
				body: 'ui-meal-log-modal-body',
				title: 'ui-meal-log-modal-title',
				close: 'ui-meal-log-modal-close',
			}}
			>
				{meal && (
					<LogMealForm
						key={`${meal.source}-${meal.mealId}`}
						meal={meal}
						onClose={onClose}
						onLogged={onLogged}
					/>
				)}
		</Modal>
	)
}
