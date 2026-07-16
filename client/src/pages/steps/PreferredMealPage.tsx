import { filterMealTagsForDietaryRestrictions } from "@/preferences/helpers"
import { mealCategoryOptions, restrictedMealCategories } from "@/preferences/options"
import { OptionButton } from "@/types/OptionButton"
import { QuestionBlock } from "@/types/QuestionBlock"
import { Box } from "@mantine/core"
import {
  Apple,
  Bean,
  Beef,
  CakeSlice,
  Carrot,
  Coffee,
  Croissant,
  Drumstick,
  Egg,
  Fish,
  Flame,
  GlassWater,
  Leaf,
  Milk,
  Nut,
  Pizza,
  Salad,
  Sandwich,
  Soup,
  Utensils,
  Vegan,
  Wheat,
  PiggyBank,
  type LucideIcon,
  Hamburger,
} from "lucide-react"
import { useEffect, useMemo } from "react"


type MealPreferenceStepProps = {
  value: string[]
  dietaryRestrictions: string[]
  onChange: (value: string[]) => void
}

// Icons are intentionally grouped by food family/cuisine style so the long
// preference list is easier to scan without making every chip visually noisy.
const mealCategoryIcons: Record<string, LucideIcon> = {
  malaysian: Utensils,
  singaporean: Utensils,
  indonesian: Utensils,
  chinese: Utensils,
  indian: Utensils,
  thai: Utensils,
  vietnamese: Utensils,
  japanese: Utensils,
  korean: Utensils,
  middle_eastern: Utensils,
  american: Utensils,
  mexican: Utensils,
  italian: Pizza,
  french: Croissant,
  greek: Utensils,
  spanish: Utensils,
  western: Hamburger,
  breakfast: Croissant,
  rice_dishes: Wheat,
  noodle_dishes: Soup,
  soups: Soup,
  stews: Soup,
  curries: Flame,
  stir_fries: Utensils,
  grilled_roasted: Flame,
  fried_foods: Flame,
  salads: Salad,
  sandwiches_wraps: Sandwich,
  breads_flatbreads: Wheat,
  porridge: Soup,
  dumplings: Utensils,
  snacks: Coffee,
  desserts: CakeSlice,
  beverages: GlassWater,
  condiments_sauces: Utensils,
  poultry: Drumstick,
  beef: Beef,
  pork: PiggyBank,
  lamb: Utensils,
  seafood: Fish,
  eggs: Egg,
  tofu_soy: Vegan,
  legumes: Bean,
  vegetables: Carrot,
  fruits: Apple,
  grains: Wheat,
  dairy: Milk,
  nuts_seeds: Nut,
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
  }, [value, dietaryRestrictions, onChange])

  const toggleMealCategory = (category: string) => {
    const nextValue = value.includes(category)
    ? value.filter((item) => item !== category)
    : [...value, category]

    onChange(nextValue)
  }

  return (
  <QuestionBlock title="Which meal types do you want to see more often?">
    <Box className="ui-chip-grid">
      {mealCategoryOptions.map((category) => {
        const Icon = mealCategoryIcons[category.value] ?? Leaf

        return (
          <OptionButton
            key={category.value}
            icon={<Icon size={18} strokeWidth={2.2} />}
            label={category.label}
            selected={value.includes(category.value)}
            disabled={disabledMealCategories.has(category.value)}
            onClick={() => {toggleMealCategory(category.value)}}
          />
        )
      })}
    </Box>
  </QuestionBlock>
)

}
