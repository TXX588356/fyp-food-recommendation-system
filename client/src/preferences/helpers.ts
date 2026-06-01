import { mainGoalOptions, restrictedMealCategories } from "./options";
import type { LocationValue, PreferenceData, PreferencesFormValues } from "./types";

export const createEmptyLocation = (): LocationValue => ({
  state: '',
  district: '',
})

export const formatLocation = (location: LocationValue) =>
    `${location.district}, ${location.state}`


// Convert backend string into frontend form object
export const parseLocation = (location: string): LocationValue => {
	const separatorIndex = location.lastIndexOf(',')

	if (separatorIndex === -1) {
		return {
			district: location.trim(),
			state: '',
		}
	}

	return {
		district: location.slice(0, separatorIndex).trim(),
		state: location.slice(separatorIndex + 1).trim(),
	}
}

export const formatList = (values: string[]) => {
	if (values.length === 0) return 'Not set'

	return values
	.map((value) =>
		value
		.split("_")
		.map((word) => word[0].toUpperCase() + word.slice(1))
		.join(' '),
	)
	.join(', ')
}

export const formatGoal = (goal: string) =>
	mainGoalOptions.find((option) => option.value === goal)?.label || 'Not set'

export const formatConsent = (consent: boolean | null) => {
	if (consent === null) return 'Not answered'
	return consent ? 'Enabled' : 'Disabled'
}

export const formatDietaryRestrictions = (values: string[]) => {
	if (values.length === 0) return 'No restrictions'

	return formatList(values)
}

export const formatHealthConcerns = (values: string[]) => {
	if (values.length === 0) return 'No health concerns'

	return formatList(values)
}

export const noneOptionToFormValue  = (values: string[]) => {
	if (values.length === 0) return ['none']

	return values
}

export const noneOptionToPayloadValue  = (values: string[]) => {
	return values.filter((item) => item !== 'none')
}

/*
1. Takes current selected meal tags & dietary restrictions
2. Returns only compatible meal tags
Applied when saving dietary restrictions & meal preferences & onboarding payload builder
*/
export const filterMealTagsForDietaryRestrictions = (
	tags: string[],
	restrictions: string[],
) => {
	const disabledTags = new Set(
		restrictions.flatMap(
			(restriction) => restrictedMealCategories[restriction] ?? [],
		),
	)

	return tags.filter((tag) => !disabledTags.has(tag))
}

export const getEmptyPreference = (): PreferenceData => ({
  mainGoal: '',
  monthlyMealBudget: 0,
  dataSharingConsent: null,
  homeLocation: '',
  workSchoolLocation: '',
  healthConcerns: [],
  dietaryRestrictions: [],
  preferredMealTags: [],
})

export function buildPreferencePayload(values: PreferencesFormValues) {
	const dietaryRestrictions = values.dietaryRestrictions.filter(
		(item) => item !== 'none',
	)

	return {
		mainGoal: values.mainGoal,
		dietaryRestrictions,
		healthConcerns: values.healthConcerns.filter(
			(item) => item !== 'none',
		),
		preferredMealTags: filterMealTagsForDietaryRestrictions(
			values.preferredMealCategories,
			dietaryRestrictions,
		),
		monthlyMealBudget: Number(values.monthlyMealBudget),
		workSchoolLocation: formatLocation(values.workSchoolLocation),
		homeLocation: formatLocation(values.homeLocation),
	}
}
