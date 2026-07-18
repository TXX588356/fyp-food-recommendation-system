import type { CategoryState, FilteredMealCandidate, MatchedMealCandidate, MealCategory, PersistedRecommendationState } from "./recommendationTypes";

export const emptyCandidates: CategoryState<MatchedMealCandidate[]> = {
	breakfast: [],
	lunch: [],
	dinner: [],
	snack: [],
}

export const emptyFilteredOut: CategoryState<FilteredMealCandidate[]> = {
	breakfast: [],
	lunch: [],
	dinner: [],
	snack: [],
}

export const emptyGenerated: CategoryState<boolean> = {
  breakfast: false,
  lunch: false,
  dinner: false,
  snack: false,
}

export const getLocalDateKey = () => {
	const today = new Date()
	const year = today.getFullYear()
	const month = String(today.getMonth() + 1).padStart(2, '0')
	const day = String(today.getDate()).padStart(2, '0')

	return `${year}-${month}-${day}`
}

export const buildRecommendationStorageKey = (userKey: string | undefined) => {
	return `recommendation:${userKey ?? 'anonymous'}`
}

export const createEmptyPersistedRecommendation = (): PersistedRecommendationState => ({
	generatedDate: getLocalDateKey(),
	candidatesByCategory: {...emptyCandidates},
	filteredOutByCategory: {...emptyFilteredOut},
	generatedByCategory: {...emptyGenerated},
})

const hasScoreBreakdown = (candidate: MatchedMealCandidate) => (
	typeof candidate.score === 'number' &&
	typeof candidate.score_breakdown?.goal_alignment === 'number' &&
	typeof candidate.score_breakdown?.budget_fit === 'number' &&
	typeof candidate.score_breakdown?.recency_penalty === 'number' &&
	typeof candidate.score_breakdown?.preference === 'number'
)

const needsRegeneration = (
	candidates: MatchedMealCandidate[],
	filteredOut: FilteredMealCandidate[],
) => {
	const allCandidates = [
		...candidates,
		...filteredOut.map((item) => item.candidate),
	]

	return allCandidates.length > 0 && allCandidates.some((candidate) => !hasScoreBreakdown(candidate))
}

export const loadPersistedRecommendations = (storageKey: string): PersistedRecommendationState => {
	const emptyState = createEmptyPersistedRecommendation()

	try{
		const rawValue = localStorage.getItem(storageKey)
		if (!rawValue) {
			return emptyState
		}

		const parsedValue = JSON.parse(rawValue) as Partial<PersistedRecommendationState>

		// Recommendation cache is intentionally daily because generated meals are daily context.
		if (parsedValue.generatedDate !== getLocalDateKey()) {
			localStorage.removeItem(storageKey)

			return emptyState
		}

		const candidatesByCategory = {
			...emptyCandidates,
			...parsedValue.candidatesByCategory,
		}
		const filteredOutByCategory = {
			...emptyFilteredOut,
			...parsedValue.filteredOutByCategory,
		}
		const generatedByCategory = {
			...emptyGenerated,
			...parsedValue.generatedByCategory,
		}

		for (const category of Object.keys(generatedByCategory) as MealCategory[]) {
			if (generatedByCategory[category] && needsRegeneration(candidatesByCategory[category], filteredOutByCategory[category])) {
				candidatesByCategory[category] = []
				filteredOutByCategory[category] = []
				generatedByCategory[category] = false
			}
		}

		return {
			generatedDate: parsedValue.generatedDate,
			candidatesByCategory,
			filteredOutByCategory,
			generatedByCategory,
		}
	} catch {
		localStorage.removeItem(storageKey)
		return emptyState
	}
}

export const savePersistedRecommendations = (
	storageKey: string,
	nextState: PersistedRecommendationState,
) => {
	localStorage.setItem(storageKey, JSON.stringify(nextState))
}

export const findPersistedRecommendationCandidate = (
	storageKey: string,
	mealCategory: MealCategory,
	mealId: string,
) => {
	const persisted = loadPersistedRecommendations(storageKey)

	//return persisted.candidatesByCategory[mealCategory].find((candidate) => candidate.food.id === mealId) ?? null

	const candidates = persisted.candidatesByCategory[mealCategory]

	const result = candidates.find((candidate) => {
		return candidate.food.id === mealId
	})

	if (result === undefined) {
		return null
	}

	return result
}
