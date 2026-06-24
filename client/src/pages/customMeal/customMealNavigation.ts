const customMealSuccessMessage = 'Custom meal added successfully.'

export const customMealCreatedNavigationState = {
  successMessage: customMealSuccessMessage,
} as const

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
