import { Box, Checkbox, MantineProvider, SimpleGrid, Text, createTheme } from '@mantine/core'

import { dietaryRestrictionOptions, mealCategoryOptions } from '@/preferences/options'
import type { CustomMealDraft, CustomMealFieldErrors } from '../customMealTypes'

const checkboxTheme = createTheme({
  cursorType: 'pointer',
})

const customMealDietaryOptions = dietaryRestrictionOptions.filter((option) => option.value !== 'none')

type CustomMealTagFieldsProps = {
  draft: CustomMealDraft
  fieldErrors: CustomMealFieldErrors
  restrictedMealCategoryTags: Set<string>
  onToggleDietaryTag: (tag: string) => void
  onToggleMealCategoryTag: (tag: string) => void
}

export default function CustomMealTagFields({
  draft,
  fieldErrors,
  restrictedMealCategoryTags,
  onToggleDietaryTag,
  onToggleMealCategoryTag,
}: CustomMealTagFieldsProps) {
  return (
    <>
      <Box className="ui-dietary-tag-group">
        <Text fw={900}>Dietary tags (optional)</Text>
        <SimpleGrid cols={{ base: 1, xs: 2, sm: 3 }} spacing="sm">
          {customMealDietaryOptions.map((option) => (
            <MantineProvider key={option.value} theme={checkboxTheme}>
              <Checkbox
                label={option.label}
                checked={draft.dietaryRestrictionTags.includes(option.value)}
                onChange={() => onToggleDietaryTag(option.value)}
              />
            </MantineProvider>
          ))}
        </SimpleGrid>
      </Box>

      <Box
        className={`ui-dietary-tag-group ui-tag-error-target t-input ${
          fieldErrors.mealCategoryTags ? 'is-error' : ''
        }`}
        data-custom-meal-field="mealCategoryTags"
        role="group"
        aria-labelledby="meal-category-tags-label"
        aria-required="true"
        aria-invalid={fieldErrors.mealCategoryTags ? 'true' : undefined}
      >
        <Box>
          <Text id="meal-category-tags-label" fw={900}>
            Meal category tags <Text component="span" c="red">*</Text>
          </Text>
          <Text size="sm" c="dimmed">Select at least one.</Text>
        </Box>
        {fieldErrors.mealCategoryTags && (
          <Text size="sm" className="ui-custom-meal-field-error">
            {fieldErrors.mealCategoryTags}
          </Text>
        )}
        <SimpleGrid cols={{ base: 1, xs: 2, sm: 3 }} spacing="sm">
          {mealCategoryOptions.map((option) => {
            const isRestricted = restrictedMealCategoryTags.has(option.value)

            return (
              <Checkbox
                key={option.value}
                label={option.label}
                description={isRestricted ? 'Blocked by selected dietary restriction' : undefined}
                disabled={isRestricted}
                checked={draft.mealCategoryTags.includes(option.value)}
                onChange={() => onToggleMealCategoryTag(option.value)}
              />
            )
          })}
        </SimpleGrid>
      </Box>
    </>
  )
}
