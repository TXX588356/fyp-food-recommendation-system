import { getEmptyPreference, noneOptionToFormValue, noneOptionToPayloadValue } from "@/preferences/helpers"
import type { PreferenceData } from "@/preferences/types"
import { Alert, Box, Button, Group, Text, Title } from "@mantine/core"
import axios from "axios"
import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { HealthConcernStep } from "../steps/HealthConcernPage"


const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

export default function HealthConcernsPreferencePage() {
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
				healthConcerns: noneOptionToFormValue(
					response.data.healthConcerns,
				)
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
		setIsSaving(true)
		setError(null)

		if (draft.healthConcerns.length === 0) {
			setError("Please select at least one option.")
			return
		}

		const payload = {
			...draft,
			healthConcerns: noneOptionToPayloadValue (
				draft.healthConcerns,
			),
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
			console.error("Failed to update health concerns", error)
			setError("Could not update health concerns.")
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
								<Text>Health concerns</Text>
							</Box>
	
							<Title order={1}>Update health concerns</Title>
						</Box>
	
						{error && (
							<Alert color="red" mt="lg">{error}</Alert>
						)}
	
						{isLoading ? (
							<Box className="ui-question">
								<Text fw={800}>Loading your saved preferences...</Text>
							</Box>
						) : (
							<HealthConcernStep 
								value={draft.healthConcerns}
								onChange={(value) => 
									setDraft((current) => ({
										...current, 
										healthConcerns: value,
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
								disabled={isLoading || draft.healthConcerns.length === 0}
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
