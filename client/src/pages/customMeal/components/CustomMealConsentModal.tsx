import { Box, Button, Modal, Stack, Text } from '@mantine/core'

import { modalContentClassName, modalTransitionProps } from '@/theme/modalTransitions'

type CustomMealConsentModalProps = {
  opened: boolean
  consentChoiceBeingSaved: boolean | null
  onClose: () => void
  onSaveConsent: (consent: boolean) => void
}

export default function CustomMealConsentModal({
  opened,
  consentChoiceBeingSaved,
  onClose,
  onSaveConsent,
}: CustomMealConsentModalProps) {
  const canClose = consentChoiceBeingSaved === null

  return (
    <Modal
      opened={opened}
      onClose={() => {
        if (canClose) {
          onClose()
        }
      }}
      closeOnClickOutside={canClose}
      closeOnEscape={canClose}
      transitionProps={modalTransitionProps}
      centered
      title="Choose your meal sharing preference"
      classNames={{
        content: modalContentClassName(),
      }}
    >
      <Stack gap="md">
        <Text>
          This preference applies to every custom meal you create. You can change it later from Preferences.
        </Text>

        <Box className="ui-card-field-group">
          <Text fw={900}>Share anonymously</Text>
          <Text size="sm">
            Other users may discover your custom meals, but your personal account details are not shown
          </Text>
        </Box>

        <Box className="ui-card-field-group">
          <Text fw={900}>Keep my meals private</Text>
          <Text size="sm">
            Only you can view and search your custom meals.
          </Text>
        </Box>

        <Button
          className="ui-primary-button"
          loading={consentChoiceBeingSaved === true}
          disabled={consentChoiceBeingSaved !== null && consentChoiceBeingSaved !== true}
          onClick={() => onSaveConsent(true)}
        >
          Share anonymously
        </Button>

        <Button
          variant="outline"
          loading={consentChoiceBeingSaved === false}
          disabled={consentChoiceBeingSaved !== null && consentChoiceBeingSaved !== false}
          onClick={() => onSaveConsent(false)}
        >
          No, keep my meals private
        </Button>

        <Button variant="subtle" disabled={!canClose} onClick={onClose}>
          Cancel
        </Button>
      </Stack>
    </Modal>
  )
}
