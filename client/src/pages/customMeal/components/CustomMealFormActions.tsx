import { Button, Group } from '@mantine/core'

type CustomMealFormActionsProps = {
  isSaving: boolean
  isCheckingConsent: boolean
  onBack: () => void
  onSave: () => void
}

export default function CustomMealFormActions({
  isSaving,
  isCheckingConsent,
  onBack,
  onSave,
}: CustomMealFormActionsProps) {
  return (
    <Group justify="space-between" className="ui-custom-meal-actions">
      <Button className="ui-dark-button" onClick={onBack}>
        Back
      </Button>

      <Button
        className="ui-dark-button"
        loading={isSaving || isCheckingConsent}
        onClick={onSave}
      >
        Save
      </Button>
    </Group>
  )
}
