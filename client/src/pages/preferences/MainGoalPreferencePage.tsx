import { getEmptyPreference } from "@/preferences/helpers"
import type { PreferenceData } from "@/preferences/types"
import { Title, Text, Box, Alert, Button, Group } from "@mantine/core"
import axios from "axios"
import { useEffect, useState } from "react"
import { GoalStep } from "../steps/MainGoalPage"
import { useNavigate } from "react-router-dom"

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

export default function MainGoalPreferencePage() {
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
				mainGoal: response.data.mainGoal,
			})
			} catch (error) {
				console.error("Failed to load main goal data", error)
				setError("Could not load your saved preferences.")
			} finally {
				setIsLoading(false)
			}
		}

		loadingPreferences()
	}, [token])

	const savePreferences = async() => {
		setIsSaving(true)
		setError(null)

		if (draft.mainGoal.length === 0) {
			setError("Please select one main goal.")
			setIsSaving(false)
			return
		}

		try {
			await axios.put<PreferenceData>(
				`${API_BASE_URL}/preferences`, draft,
				{
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			)

			navigate("/preferences")
		} catch (error) {
			console.error("Failed to update main goal", error)
			setError("Could not update main goal.")
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
									<Text>Main goal</Text>
								</Box>
		
								<Title order={1}>Update main goal</Title>
							</Box>
		
							{error && (
								<Alert color="red" mt="lg">{error}</Alert>
							)}
		
							{isLoading ? (
								<Box className="ui-question">
									<Text fw={800}>Loading your saved preferences...</Text>
								</Box>
							) : (
								<GoalStep 
									value={draft.mainGoal}
									onChange={(value) => 
										setDraft((current) => ({
											...current, 
											mainGoal: value,
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
									disabled={isLoading || draft.mainGoal.length === 0}
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
