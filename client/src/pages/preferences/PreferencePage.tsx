import { useAuth } from "@/auth/useAuth"
import { formatConsent, formatDietaryRestrictions, formatGoal, formatHealthConcerns, formatList, getEmptyPreference } from "@/preferences/helpers"
import type { PreferenceData, SettingKey } from "@/preferences/types"
import { Alert, Box, Button, Text, Title, UnstyledButton } from "@mantine/core"
import axios from "axios"
import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import MainNav from "@/theme/MainNav"


export default function PreferencePage() {
  const navigate = useNavigate()
  const { logout } = useAuth()
  const [preference, setPreference] = useState<PreferenceData>(getEmptyPreference)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const token = localStorage.getItem('token')

  const openPreferenceSetting = (key: SettingKey) => {
    const routes: Record<SettingKey, string> = {
      dietary: '/preferences/dietaryRestrictions',
      meals: '/preferences/mealPreferences',
      budgetLocation: '/preferences/budget-location',
      health: '/preferences/healthConcerns',
      goals: '/preferences/goal',
      consent: '/preferences/consent',
    }

    navigate(routes[key])
  }

  useEffect(() => {
    const loadPreferences = async() => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await axios.get<PreferenceData>("/preferences", {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        })

        setPreference(response.data)
      } catch(error) {
        console.error('Failed to load preferences', error)
        setError('Could not load your saved preferences.')
      } finally {
        setIsLoading(false)
      }
    }

    loadPreferences()
  }, [token])

  const settings = useMemo(
    () => [
      {
        key: 'dietary' as const,
        title: 'Dietary Restrictions',
        value: formatDietaryRestrictions(preference.dietaryRestrictions)
      },
      {
        key: 'meals' as const,
        title: 'Meal Preferences',
        value: formatList(preference.preferredMealTags)
      },
      {
        key: 'budgetLocation' as const,
        title: 'Budget & Location',
        value: 
        preference.monthlyMealBudget > 0 ||
        preference.homeLocation ||
        preference.workSchoolLocation
          ? [
              preference.monthlyMealBudget > 0
                ? `RM ${preference.monthlyMealBudget}`
                : 'Budget not set',
              preference.homeLocation || 'Home not set',
              preference.workSchoolLocation || 'Work / school not set',
            ].join(' / ')
          : 'Not set',
      },
      {
        key: 'health' as const,
        title: 'Health Conditions / Concerns',
        value: formatHealthConcerns(preference.healthConcerns)
      },
      {
        key: 'goals' as const,
        title: 'Goals',
        value: formatGoal(preference.mainGoal)
      },
      {
        key: 'consent' as const,
        title: 'Data Sharing Consent',
        value: formatConsent(preference.dataSharingConsent)
      },
    ],
    [preference],
  )

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <Box className="ui-settings-page">
      <Box component="main" className="ui-settings-frame">
        <MainNav active="preferences" />

        <Box className="ui-settings-layout">
          <Box component="section" className="ui-settings-panel ui-surface ui-panel">
            <Box className="ui-page-header ui-settings-header">
              <Text className="ui-eyebrow">Profile controls</Text>
              <Title order={1}>Preference settings</Title>
              <Text className="ui-page-copy">Review and adjust the preference profile that shapes your recommendations.</Text>
            </Box>

          {error && (
            <Alert color="red" mt="lg">
              {error}
            </Alert>
          )}

          {isLoading ? (
            <Box className="ui-settings-editor ui-card" mt="xl">
              <Text fw={800}>Loading your saved preferences...</Text>
            </Box>
          ) : (
            <>
              <Box className="ui-settings-list">
                {settings.map((item) => (
                  <Box key={item.key}>
                    <UnstyledButton
                      className="ui-settings-row ui-card"
                      onClick={() => openPreferenceSetting(item.key)}
                    >
                      <Box>
                        <Text className="ui-settings-title">{item.title}</Text>
                      </Box>

                      <Text className="ui-settings-value">{item.value}</Text>
                      <span className="ui-settings-chevron">›</span>
                    </UnstyledButton>
                  </Box>
                ))}
              </Box>            
            </>
          )}
          <Box className="ui-centered-row" mt={60}>
            <Button className="ui-danger-button" onClick={handleLogout}>
              Log out
            </Button>
          </Box>
        </Box>
      </Box>
      </Box>
    </Box>
  )
}
