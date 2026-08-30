export type MealTime = 'breakfast' | 'lunch' | 'dinner' | 'snack'

export type CustomMealResponse = {
  id: string
  name: string
  price: number
  calories: number
  fatG: number
  proteinG: number
  carbsG: number
  state: string
  district: string
  restaurantName: string
  imageURL: string
  dietaryRestrictionTags: string[]
  mealCategoryTags: string[]
  isOwner: boolean
  isShared: boolean
}

export type MealSearchResult = {
  id: string
  name: string
  source: 'prebuilt' | 'custom'
  tags: string[]
  calories: number
  fat_g: number
  protein_g: number
  carbs_g: number
  price?: number
  serving_description?: string
  image_url?: string
}

export type ExistingMeal = {
  id: string
  name: string
  calories: number
  priceLabel: string
  tags: string[]
  source: 'prebuilt' | 'custom'
  imageUrl?: string
}

export type CustomMealDraft = {
  name: string
  price: number | ''
  calories: number | ''
  carbsG: number | ''
  fatG: number | ''
  proteinG: number | ''
  state: string
  district: string
  restaurantName: string
  dietaryRestrictionTags: string[]
  mealCategoryTags: string[]
}

export type CustomMealAutocompleteResponse = {
  calories: number
  fatG: number
  proteinG: number
  carbsG: number
  dietaryRestrictionTags: string[]
  mealCategoryTags: string[]
}

export type CustomMealField =
  | keyof Pick<
      CustomMealDraft,
      | 'name'
      | 'price'
      | 'calories'
      | 'carbsG'
      | 'fatG'
      | 'proteinG'
      | 'state'
      | 'district'
      | 'restaurantName'
      | 'mealCategoryTags'
    >
  | 'image'

export type CustomMealFieldErrors = Partial<Record<CustomMealField, string>>

export type CustomMealInputClassNames = (field: CustomMealField) => {
  label: string
  input: string
  wrapper: string
}

export type UpdateCustomMealDraft = <Key extends keyof CustomMealDraft>(
  key: Key,
  value: CustomMealDraft[Key],
) => void
