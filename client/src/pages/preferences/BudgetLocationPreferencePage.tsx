import { formatLocation, getEmptyPreference, parseLocation } from "@/preferences/helpers"
import type { PreferenceData } from "@/preferences/types"
import { Alert, Box, Button, Group, Text, Title } from "@mantine/core"
import axios from "axios"
import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { BudgetLocationStep } from "../steps/BudgetLocationPage"

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

export default function BudgetLocationPreferencePage() {
	const navigate = useNavigate()
	const [draft, setDraft] = useState<PreferenceData>(getEmptyPreference)
	const [isLoading, setIsLoading] = useState(true)
	const [isSaving, setIsSaving] = useState(false)
	const [error, setError] = useState<string | null>(null)

	const token = localStorage.getItem("token")

	useEffect(() => {
		const loadingPreferences = async () => {
			setIsLoading(true)
			setError(null)

			try {
				const response = await axios.get<PreferenceData>(
					`${API_BASE_URL}/preferences`,
				{
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			)

			setDraft({
				...response.data,
				monthlyMealBudget: response.data.monthlyMealBudget,
				homeLocation: response.data.homeLocation,
				workSchoolLocation: response.data.workSchoolLocation,
			
			})
			} catch (error) {
				console.error("Failed to load preferences data", error)
				setError("Could not load your saved preferences.")
			} finally {
				setIsLoading(false)
			}
		}

		loadingPreferences()
	}, [token])

	const savePreferences = async () => {
		setError(null)
		const homeLocation = parseLocation(draft.homeLocation)
		const workSchoolLocation = parseLocation(draft.workSchoolLocation)

		if (draft.monthlyMealBudget <= 0) {
			setError("Please provide valid monthly meal budget.")
			return
		} else if (
			!workSchoolLocation.state ||
			!workSchoolLocation.district ||
			!homeLocation.state ||
			!homeLocation.district
		) {
			setError("Please provide your state and district.")
			return
		}

		setIsSaving(true)

		const payload = {
			...draft,
			monthlyMealBudget: draft.monthlyMealBudget,
			homeLocation: draft.homeLocation,
			workSchoolLocation: draft.workSchoolLocation,
		}

		try {
			await axios.put<PreferenceData>(
				`${API_BASE_URL}/preferences`, payload,
				{
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			)

			navigate("/preferences")
		} catch (error) {
			console.error("Failed to update budget and location", error)
			setError("Could not update budget and location.")
		} finally {
			setIsSaving(false)
		}
	}

	return (
			<Box className="ui-page">
				<Box component="main" className="ui-page-shell">
					<Box component="section" className="ui-feature-panel">
						<Box className="ui-page-header">
							<Box className="ui-topline">
								<span>Preference settings</span>
								<Text>Budget & Location</Text>
							</Box>
	
							<Title order={1}>Update budget and location</Title>
						</Box>
	
						{error && (
							<Alert color="red" mt="lg">{error}</Alert>
						)}
	
						{isLoading ? (
							<Box className="ui-question">
								<Text fw={800}>Loading your saved preferences...</Text>
							</Box>
						) : (
							<BudgetLocationStep 
								monthlyMealBudget={draft.monthlyMealBudget}
								onMonthlyMealBudgetChange={(value) =>
									setDraft((current) => ({
										...current, 
										monthlyMealBudget: Number(value) || 0,
									}))
								}
								workSchoolLocation={parseLocation(draft.workSchoolLocation)}
								onWorkSchoolLocationChange={(value) =>
									setDraft((current) => ({
										...current,
										workSchoolLocation: formatLocation(value),
									}))
								}
								homeLocation={parseLocation(draft.homeLocation)}
								onHomeLocationChange={(value) =>
									setDraft((current) => ({
										...current,
										homeLocation: formatLocation(value),
									}))
								}
							/>
						)}
	
						<Group justify="space-between" className="ui-actions">
							<Button
								variant="subtle"
								className="ui-ghost-button"
								onClick={() => navigate("/preferences")}
							>
								Cancel
							</Button>
	
							<Button
								className="ui-primary-button ui-action-button"
								loading={isSaving}
								disabled={isLoading || draft.monthlyMealBudget <= 0 || draft.homeLocation.length === 0 || draft.workSchoolLocation.length === 0}
								onClick={savePreferences}
							>
								Save changes
							</Button>
						</Group>
					</Box>
				</Box>
			</Box>
		)
}
