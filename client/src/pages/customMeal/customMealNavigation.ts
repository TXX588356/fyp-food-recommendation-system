const customMealSuccessMessage = 'Custom meal added successfully.'

type CustomMealCreatedNavigationState = {
  successMessage: typeof customMealSuccessMessage
  mealCategory?: string
}

export const customMealCreatedNavigationState = (mealCategory?: string): CustomMealCreatedNavigationState => ({
  successMessage: customMealSuccessMessage,
  mealCategory,
})

export const getCustomMealSuccessMessage = (state: unknown): string | null => {
  if (
    typeof state === 'object' &&
    state !== null &&
    'successMessage' in state &&
    state.successMessage === customMealSuccessMessage
  ) {
    return state.successMessage
  }

  return null
}

export const getCustomMealCreatedCategory = (state: unknown): string | null => {
  if (
    typeof state === 'object' &&
    state !== null &&
    'successMessage' in state &&
    state.successMessage === customMealSuccessMessage &&
    'mealCategory' in state &&
    typeof state.mealCategory === 'string'
  ) {
    return state.mealCategory
  }

  return null
}
