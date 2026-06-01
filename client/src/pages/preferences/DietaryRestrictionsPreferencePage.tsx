import { useNavigate } from "react-router-dom";
import { DietaryRestrictionStep } from "../steps/DietaryRestrictionPage";
import { filterMealTagsForDietaryRestrictions, noneOptionToFormValue , noneOptionToPayloadValue , getEmptyPreference } from "@/preferences/helpers";
import type { PreferenceData } from "@/preferences/types";
import { useEffect, useState } from "react";
import axios from "axios";
import { Alert, Box, Button, Group, Text, Title } from "@mantine/core";


const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

export default function DietaryRestrictionsPreferencePage() {
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
        dietaryRestrictions: noneOptionToFormValue (
          response.data.dietaryRestrictions,
        )})
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

    if (draft.dietaryRestrictions.length === 0) {
      setError("Please select at least one option.")
      return
    }

    const dietaryRestrictions = noneOptionToPayloadValue(
      draft.dietaryRestrictions,
    )
    const payload = {
      ...draft, 
      dietaryRestrictions,
      preferredMealTags: filterMealTagsForDietaryRestrictions(
        draft.preferredMealTags,
        dietaryRestrictions,
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
      console.error("Failed to update dietary restrictions", error)
      setError("Could not update dietary restrictions.")
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
              <Text>Dietary restrictions</Text>
            </Box>

            <Title order={1}>Update dietary restrictions</Title>
          </Box>

          {error && (
            <Alert color="red" mt="lg">{error}</Alert>
          )}

          {isLoading ? (
            <Box className="ui-question">
              <Text fw={800}>Loading your saved preferences...</Text>
            </Box>
          ) : (
            <DietaryRestrictionStep 
              value={draft.dietaryRestrictions}
              onChange={(value) => 
                setDraft((current) => ({
                  ...current, 
                  dietaryRestrictions: value,
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
              disabled={isLoading || draft.dietaryRestrictions.length === 0}
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
