import { Box, FileInput, Select, SimpleGrid, Text, TextInput } from '@mantine/core'

import type {
  CustomMealDraft,
  CustomMealFieldErrors,
  CustomMealInputClassNames,
  UpdateCustomMealDraft,
} from '../customMealTypes'

type CustomMealLocationFieldsProps = {
  draft: CustomMealDraft
  fieldErrors: CustomMealFieldErrors
  inputClassNames: CustomMealInputClassNames
  mealImage: File | null
  stateOptions: Array<{ value: string; label: string }>
  onUpdateDraft: UpdateCustomMealDraft
  onMealImageChange: (file: File | null) => void
}

export default function CustomMealLocationFields({
  draft,
  fieldErrors,
  inputClassNames,
  mealImage,
  stateOptions,
  onUpdateDraft,
  onMealImageChange,
}: CustomMealLocationFieldsProps) {
  return (
    <Box className="ui-location-fields">
      <Text fw={900}>Enter location where you had this meal</Text>
      <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="sm">
        <Select
          data-custom-meal-field="state"
          label="State"
          classNames={inputClassNames('state')}
          data={stateOptions}
          value={draft.state}
          error={fieldErrors.state}
          onChange={(value) => onUpdateDraft('state', value ?? '')}
        />
        <TextInput
          data-custom-meal-field="district"
          label="District"
          classNames={inputClassNames('district')}
          value={draft.district}
          error={fieldErrors.district}
          onChange={(event) => onUpdateDraft('district', event.currentTarget.value)}
        />
      </SimpleGrid>
      <TextInput
        data-custom-meal-field="restaurantName"
        label="Restaurant Name"
        classNames={inputClassNames('restaurantName')}
        value={draft.restaurantName}
        error={fieldErrors.restaurantName}
        onChange={(event) => onUpdateDraft('restaurantName', event.currentTarget.value)}
      />
      <FileInput
        data-custom-meal-field="image"
        classNames={inputClassNames('image')}
        clearable
        accept="image/png,image/jpeg"
        label="Upload meal image (optional)"
        value={mealImage}
        error={fieldErrors.image}
        onChange={onMealImageChange}
      />
    </Box>
  )
}
