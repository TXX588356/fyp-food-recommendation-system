export type MealCategory = 'breakfast' | 'lunch' | 'dinner' | 'snack'

export type CategoryState<T> = Record<MealCategory, T>

export type PriceRange = {
	min: number
	max: number
}

export type GeneratedMeal = {
	name: string
	alternative_search_terms: string[]
	estimated_price_range: PriceRange
	sodium_level: string
	sugar_level: string
	purine_risk: string
	health_flags: Record<string, string>
}

export type FoodSearchResult = {
	id: string
	name: string
	source?: 'prebuilt' | 'custom'
	tags: string[]
	calories: number
	fat_g: number
	protein_g: number
	carbs_g: number
	serving_description?: string
	image_url?: string
}

export type CandidateScoreBreakdown = {
	goal_alignment: number
	budget_fit: number
	recency_penalty: number
	preference: number
}

export type MatchedMealCandidate = {
	generated_meal: GeneratedMeal
	food: FoodSearchResult
	matched_query: string
	score: number
	score_breakdown?: CandidateScoreBreakdown
}

export type FilteredMealCandidate = {
  candidate: MatchedMealCandidate
  reason: string
}

export type GenerateRecommendationResponse = {
	candidate: MatchedMealCandidate[]
	filtered_out: FilteredMealCandidate[]
	filtering_applied: boolean
}

export type PersistedRecommendationState = {
	generatedDate: string
	candidatesByCategory: CategoryState<MatchedMealCandidate[]>
	filteredOutByCategory: CategoryState<FilteredMealCandidate[]>
	generatedByCategory: CategoryState<boolean>
	locationByCategory: CategoryState<string>
}

export type MealDetailResponse = {
	meal: {
		id: string
		name: string
		mealCategory: MealCategory
		imageUrl?: string
		servingDescription?: string
		estimatedPriceRange: PriceRange
		nutrition: {
			calories: number
			fatG: number
			proteinG: number
			carbsG: number
		}
		signals: {
			sodiumLevel: string
			sugarLevel: string
			purineRisk: string
			healthFlags: Record<string, string>
		}
	}
	recommendationExplanation: string
	location: {
		query: string
		basis: string
	}
	restaurants: RestaurantResult[]
	restaurantLookupStatus: 'ok' | 'no_results' | 'unavailable'
}

export type RestaurantResult = {
	name: string
  address: string
  rating: number
  reviewCount: number
  price: string
  openNow?: boolean
  thumbnailUrl?: string
  sourceUrl?: string
}
