import { Box, Button, Group, NumberInput, SimpleGrid, Text, TextInput } from '@mantine/core'

import type {
  CustomMealDraft,
  CustomMealFieldErrors,
  CustomMealInputClassNames,
  UpdateCustomMealDraft,
} from '../customMealTypes'

type CustomMealBasicFieldsProps = {
  draft: CustomMealDraft
  fieldErrors: CustomMealFieldErrors
  inputClassNames: CustomMealInputClassNames
  isAutocompleting: boolean
  onUpdateDraft: UpdateCustomMealDraft
  onAutocomplete: () => void
}

export default function CustomMealBasicFields({
  draft,
  fieldErrors,
  inputClassNames,
  isAutocompleting,
  onUpdateDraft,
  onAutocomplete,
}: CustomMealBasicFieldsProps) {
  return (
    <Box className="ui-custom-meal-fields">
      <TextInput
        data-custom-meal-field="name"
        classNames={inputClassNames('name')}
        placeholder="Meal Name"
        value={draft.name}
        error={fieldErrors.name}
        onChange={(event) => onUpdateDraft('name', event.currentTarget.value)}
      />

      <NumberInput
        data-custom-meal-field="price"
        classNames={inputClassNames('price')}
        min={0}
        decimalScale={2}
        placeholder="Price (RM)"
        value={draft.price}
        error={fieldErrors.price}
        onChange={(value) => onUpdateDraft('price', value === '' ? '' : Number(value))}
      />

      <Box className="ui-serving-size">
        <Text fw={900}>Nutritional info</Text>
        <Group gap="sm" align="center">
          <NumberInput
            data-custom-meal-field="calories"
            hideControls
            classNames={{
              ...inputClassNames('calories'),
              input: `${inputClassNames('calories').input} ui-short-input`,
            }}
            min={0}
            clampBehavior="none"
            decimalScale={0}
            value={draft.calories}
            error={fieldErrors.calories}
            onChange={(value) => onUpdateDraft('calories', value === '' ? '' : Number(value))}
          />
          <Text fw={900}>kcal</Text>
        </Group>

        <SimpleGrid cols={{ base: 1, xs: 3 }} spacing="sm">
          <NumberInput
            data-custom-meal-field="carbsG"
            classNames={inputClassNames('carbsG')}
            min={0}
            clampBehavior="none"
            decimalScale={2}
            placeholder="Carbs"
            rightSection={<Text fw={900}>g</Text>}
            value={draft.carbsG}
            error={fieldErrors.carbsG}
            onChange={(value) => onUpdateDraft('carbsG', value === '' ? '' : Number(value))}
          />
          <NumberInput
            data-custom-meal-field="fatG"
            classNames={inputClassNames('fatG')}
            min={0}
            clampBehavior="none"
            decimalScale={2}
            placeholder="Fat"
            rightSection={<Text fw={900}>g</Text>}
            value={draft.fatG}
            error={fieldErrors.fatG}
            onChange={(value) => onUpdateDraft('fatG', value === '' ? '' : Number(value))}
          />
          <NumberInput
            data-custom-meal-field="proteinG"
            classNames={inputClassNames('proteinG')}
            min={0}
            clampBehavior="none"
            decimalScale={2}
            placeholder="Protein"
            rightSection={<Text fw={900}>g</Text>}
            value={draft.proteinG}
            error={fieldErrors.proteinG}
            onChange={(value) => onUpdateDraft('proteinG', value === '' ? '' : Number(value))}
          />
        </SimpleGrid>

        <Button variant="outline" loading={isAutocompleting} onClick={onAutocomplete}>
          AI autocomplete
        </Button>
      </Box>
    </Box>
  )
}
