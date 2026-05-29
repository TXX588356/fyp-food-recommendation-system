import type { LocationValue, PreferencesFormValues } from "./types";

export const createEmptyLocation = (): LocationValue => ({
  state: '',
  district: '',
})

export const formatLocation = (location: LocationValue) =>
    `${location.district}, ${location.state}`

export function buildPreferencePayload(values: PreferencesFormValues) {
	return {
		mainGoal: values.mainGoal,
		dietaryRestrictions: values.dietaryRestrictions.filter(
			(item) => item !== 'none',
		),
		healthConcerns: values.healthConcerns.filter(
			(item) => item !== 'none',
		),
		preferredMealTags: values.preferredMealCategories,
		monthlyMealBudget: Number(values.monthlyMealBudget),
		workSchoolLocation: formatLocation(values.workSchoolLocation),
		homeLocation: formatLocation(values.homeLocation),
	}
}