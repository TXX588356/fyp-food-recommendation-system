import { filterMealTagsForDietaryRestrictions } from "@/preferences/helpers"
import { mealCategoryOptions, restrictedMealCategories } from "@/preferences/options"
import { OptionButton } from "@/types/OptionButton"
import { QuestionBlock } from "@/types/QuestionBlock"
import { Box } from "@mantine/core"
import { useEffect, useMemo } from "react"


type MealPreferenceStepProps = {
  value: string[]
  dietaryRestrictions: string[]
  onChange: (value: string[]) => void
}

export function MealPreferenceStep({
  value,
  dietaryRestrictions,
  onChange,
}: MealPreferenceStepProps) {
  const disabledMealCategories = useMemo(
    () => new Set(
      dietaryRestrictions.flatMap(
        (restriction) => restrictedMealCategories[restriction] ?? [],
      ),
    ),
    [dietaryRestrictions],
  )

  useEffect(() => {
    const nextMealCategories = filterMealTagsForDietaryRestrictions(
      value,
      dietaryRestrictions,
    )

    if (nextMealCategories.length !== value.length) {
      onChange(nextMealCategories)
    }
  }, [value, disabledMealCategories, onChange])

  const toggleMealCategory = (category: string) => {
    const nextValue = value.includes(category)
    ? value.filter((item) => item !== category)
    : [...value, category]

    onChange(nextValue)
  }

  return (
  <QuestionBlock title="Which meal types do you want to see more often?">
    <Box className="ui-chip-grid">
      {mealCategoryOptions.map((category) => (
        <OptionButton 
          key={category.value}
          label={category.label}
          selected={value.includes(category.value)}
          disabled={disabledMealCategories.has(category.value)}
          onClick={() => {toggleMealCategory(category.value)}}
        />
      ))}
    </Box>
  </QuestionBlock>
)

}
