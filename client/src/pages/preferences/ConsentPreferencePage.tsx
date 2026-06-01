import { getEmptyPreference } from "@/preferences/helpers"
import type { PreferenceData } from "@/preferences/types"
import { Alert, Box, Button, Checkbox, Group, Text, Title } from "@mantine/core"
import axios from "axios"
import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL


export default function ConsentPreferencePage() {
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
				dataSharingConsent: response.data.dataSharingConsent,
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

		if (draft.dataSharingConsent === null) {
			setError("Please choose whether you agree to share your meal logs.")
			return
		}

		setIsSaving(true)

		try {
			await axios.put(
				`${API_BASE_URL}/preferences/data-sharing`,
				{ dataSharingConsent: draft.dataSharingConsent },
				{
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			)

			navigate("/preferences")
		} catch (error) {
			console.error("Failed to update data sharing consent", error)
			setError("Could not update data sharing consent.")
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
							<Text>Data sharing consent</Text>
						</Box>

						<Title order={1}>Update data sharing consent</Title>
						<Text className="ui-page-copy">
							Choose whether anonymised meal logs can help improve recommendations.
						</Text>
					</Box>

					{error && (
						<Alert color="red" mt="lg">{error}</Alert>
					)}

					{isLoading ? (
						<Box className="ui-question">
							<Text fw={800}>Loading your saved preferences...</Text>
						</Box>
					) : (
						<Box className="ui-question ui-field-group">
							<Title order={2} className="ui-section-title">Help improve meal recommendations</Title>
							<Text className="ui-page-copy">
								Your manually logged meals can be used anonymously to enrich the recommendation dataset. You can change this choice at any time.
							</Text>

							<Box className="ui-form-grid">
								<Box className="ui-card ui-field-group">
									<Text className="ui-eyebrow">May be shared anonymously</Text>
									<ul className="ui-list">
										<li>Meal name and price</li>
										<li>Calories and nutrition details</li>
										<li>Dietary tags and place information</li>
									</ul>
								</Box>

								<Box className="ui-card ui-field-group">
									<Text className="ui-eyebrow">Always private</Text>
									<ul className="ui-list">
										<li>Your name and email</li>
										<li>Password</li>
										<li>Personal account identity</li>
									</ul>
								</Box>
							</Box>

							<Text className="ui-field-title">
								Do you agree to share your manually logged meal data anonymously?
							</Text>

							<Box className="ui-stack ui-stack-compact">
								<Checkbox
									checked={draft.dataSharingConsent === true}
									label="Yes, I agree"
									onChange={() =>
										setDraft((current) => ({
											...current,
											dataSharingConsent: true,
										}))
									}
								/>
								<Checkbox
									checked={draft.dataSharingConsent === false}
									label="No, keep my meal logs private"
									onChange={() =>
										setDraft((current) => ({
											...current,
											dataSharingConsent: false,
										}))
									}
								/>
							</Box>
						</Box>
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
							disabled={isLoading || draft.dataSharingConsent === null}
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
