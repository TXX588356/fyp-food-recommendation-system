import { Alert, Box, Button, Checkbox, Text, Title } from '@mantine/core'
import { useState } from 'react'
import '@/App.css'

const DATA_SHARING_CONSENT_KEY = 'dataSharingConsent'

const readSavedConsent = () => localStorage.getItem(DATA_SHARING_CONSENT_KEY) === 'true'

export default function PreferencePage() {
  const [dataSharingConsent, setDataSharingConsent] = useState(readSavedConsent)
  const [showSavedMessage, setShowSavedMessage] = useState(false)

  const handleSave = () => {
    localStorage.setItem(DATA_SHARING_CONSENT_KEY, String(dataSharingConsent))
    setShowSavedMessage(true)
  }

  return (
    <Box className="preference-page">
      <Box component="main" className="preference-shell">
        <Box component="section" className="preference-panel preference-settings-panel">
          <Box className="preference-header">
            <Box className="preference-topline">
              <span>profile controls</span>
              <Text>Preferences</Text>
            </Box>

            <Title order={1}>Preference settings</Title>
            <Text className="preference-intro">
              Manage how your saved profile is used after onboarding.
            </Text>
          </Box>

          {showSavedMessage && (
            <Alert color="green" mb="lg">
              Preference settings saved.
            </Alert>
          )}

          <Box className="preference-consent">
            <Checkbox
              checked={dataSharingConsent}
              onChange={(event) => {
                setDataSharingConsent(event.currentTarget.checked)
                setShowSavedMessage(false)
              }}
              label="Use my saved preference data to improve future meal recommendations."
            />
          </Box>

          <Box className="preference-actions">
            <Button className="preference-next-button" onClick={handleSave}>
              Save preferences
            </Button>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
