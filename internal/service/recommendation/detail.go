package recommendation

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *service) calculateCurrentMonthSpent(ctx context.Context, userID uuid.UUID, now time.Time) (float64, error) {
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	logs, err := s.mealLogRepository.ListByUserAndRange(ctx, userID, monthStart, now)
	if err != nil {
		return 0, fmt.Errorf("list current month meal logs: %w", err)
	}

	total := 0.0
	for _, log := range logs {
		total += log.Price
	}

	return total, nil
}

func (s *service) BuildMealDetail(ctx context.Context, userID uuid.UUID, input interfaces.MealDetailInput) (interfaces.MealDetailResult, error) {
	now := s.now()
	candidate := s.hydrateMealDetailCandidate(ctx, input.Candidate)

	currentMonthSpent, err := s.calculateCurrentMonthSpent(ctx, userID, now)
	if err != nil {
		return interfaces.MealDetailResult{}, fmt.Errorf("calculate current month spending: %w", err)
	}

	preferences, err := s.preferenceService.GetByUserID(ctx, userID)
	if err != nil {
		return interfaces.MealDetailResult{}, fmt.Errorf("load preferences: %w", err)
	}

	explanationInput := buildMealDetailExplanationInput(
		input.MealCategory,
		candidate,
		*preferences,
		currentMonthSpent,
		now,
		input.Location,
	)

	explanation, err := s.mealDetailExplainer.ExplainMealRecommendation(ctx, explanationInput)
	if err != nil {
		return interfaces.MealDetailResult{}, fmt.Errorf("explain meal recommendation: %w", err)
	}

	restaurantResult := interfaces.RestaurantSearchResult{
		Status:      interfaces.RestaurantLookupUnavailable,
		Restaurants: []interfaces.RestaurantResult{},
	}

	if strings.TrimSpace(explanationInput.Location) != "" && explanationInput.LocationBasis != "unavailable" {
		result, err := s.restaurantSearcher.SearchRestaurants(ctx, interfaces.RestaurantSearchInput{
			MealName: explanationInput.MealName,
			Location: explanationInput.Location,
			Limit:    30,
		})
		if err != nil {
			log.Printf(
				"restaurant lookup unavailable: meal=%q location=%q basis=%q error=%v",
				explanationInput.MealName,
				explanationInput.Location,
				explanationInput.LocationBasis,
				err,
			)
			restaurantResult = interfaces.RestaurantSearchResult{
				Status:      interfaces.RestaurantLookupUnavailable,
				Restaurants: []interfaces.RestaurantResult{},
			}
		} else {
			restaurantResult = result
		}
	} else {
		log.Printf(
			"restaurant lookup skipped: meal=%q location=%q basis=%q",
			explanationInput.MealName,
			explanationInput.Location,
			explanationInput.LocationBasis,
		)
	}

	return interfaces.MealDetailResult{
		Meal:                      buildMealDetailMeal(input.MealCategory, candidate),
		RecommendationExplanation: explanation,
		Location: interfaces.MealDetailLocation{
			Query: explanationInput.Location,
			Basis: explanationInput.LocationBasis,
		},
		Restaurants:            restaurantResult.Restaurants,
		RestaurantLookupStatus: restaurantResult.Status,
	}, nil

}

func (s *service) hydrateMealDetailCandidate(
	ctx context.Context,
	candidate interfaces.MatchedMealCandidate,
) interfaces.MatchedMealCandidate {
	mealID, err := uuid.Parse(strings.TrimSpace(candidate.Food.ID))
	if err != nil {
		return candidate
	}

	meal, err := s.catalogService.GetMeal(ctx, mealID)
	if err != nil {
		return candidate
	}

	if servingDescription := strings.TrimSpace(meal.SelectedPortion.Description); servingDescription != "" {
		candidate.Food.ServingDescription = servingDescription
	}
	if strings.TrimSpace(candidate.Food.ImageURL) == "" && meal.Image != nil {
		candidate.Food.ImageURL = strings.TrimSpace(meal.Image.URL)
	}

	return candidate
}

// buildMealDetailMeal converts the selected matched candidate into service-level
// display data for the meal detail response.
func buildMealDetailMeal(
	mealCategory string,
	candidate interfaces.MatchedMealCandidate,
) interfaces.MealDetailMeal {
	food := candidate.Food
	generated := candidate.GeneratedMeal

	healthFlags := map[string]string{}
	for condition, flag := range generated.HealthFlags {
		condition = strings.TrimSpace(condition)
		flag = strings.TrimSpace(flag)

		if condition == "" || flag == "" {
			continue
		}

		healthFlags[condition] = flag
	}

	return interfaces.MealDetailMeal{
		ID:                 strings.TrimSpace(food.ID),
		Name:               strings.TrimSpace(food.Name),
		MealCategory:       strings.TrimSpace(mealCategory),
		ImageURL:           strings.TrimSpace(food.ImageURL),
		ServingDescription: strings.TrimSpace(food.ServingDescription),
		EstimatedPriceRange: interfaces.PriceRange{
			Min: generated.EstimatedPriceRange.Min,
			Max: generated.EstimatedPriceRange.Max,
		},
		Nutrition: interfaces.MealDetailNutrition{
			Calories: food.Calories,
			FatG:     food.FatG,
			ProteinG: food.ProteinG,
			CarbsG:   food.CarbsG,
		},
		Signals: interfaces.MealDetailSignals{
			SodiumLevel: strings.TrimSpace(generated.SodiumLevel),
			SugarLevel:  strings.TrimSpace(generated.SugarLevel),
			PurineRisk:  strings.TrimSpace(generated.PurineRisk),
			HealthFlags: healthFlags,
		},
	}
}

// buildMealDetailExplanationInput converts the selected recommendation candidate,
// saved user preferences, and real spending context into the prompt input used by Gemini.
func buildMealDetailExplanationInput(
	mealCategory string,
	candidate interfaces.MatchedMealCandidate,
	preferences interfaces.PreferenceResponse,
	currentMonthSpent float64,
	now time.Time,
	locationOverride string,
) interfaces.MealDetailExplanationInput {
	food := candidate.Food
	generated := candidate.GeneratedMeal

	remainingBudget := preferences.MonthlyMealBudget - currentMonthSpent
	if remainingBudget < 0 {
		remainingBudget = 0
	}

	perMealBudget := calculateMealDetailDynamicPerMealBudget(
		preferences.MonthlyMealBudget,
		currentMonthSpent,
		now,
	)

	location, locationBasis := selectMealDetailPreferenceLocation(preferences, now)
	if selectedLocation := strings.TrimSpace(locationOverride); selectedLocation != "" {
		location = selectedLocation
		locationBasis = "selected"
	}

	return interfaces.MealDetailExplanationInput{
		MealName:     strings.TrimSpace(food.Name),
		MealCategory: strings.TrimSpace(mealCategory),
		Nutrition: interfaces.MealDetailNutritionInput{
			Calories: food.Calories,
			FatG:     food.FatG,
			ProteinG: food.ProteinG,
			CarbsG:   food.CarbsG,
		},
		PriceRange: generated.EstimatedPriceRange,

		SodiumLevel: strings.TrimSpace(generated.SodiumLevel),
		SugarLevel:  strings.TrimSpace(generated.SugarLevel),
		PurineRisk:  strings.TrimSpace(generated.PurineRisk),
		HealthFlags: copyTrimmedMealDetailHealthFlags(generated.HealthFlags),

		UserGoal:            strings.TrimSpace(preferences.MainGoal),
		DietaryRestrictions: copyTrimmedStringSlice(preferences.DietaryRestrictions),
		HealthConcerns:      copyTrimmedStringSlice(preferences.HealthConcerns),
		PreferredMealTags:   copyTrimmedStringSlice(preferences.PreferredMealTags),

		MonthlyMealBudget: preferences.MonthlyMealBudget,
		CurrentMonthSpent: currentMonthSpent,
		RemainingBudget:   remainingBudget,
		PerMealBudget:     perMealBudget,

		Location:      location,
		LocationBasis: locationBasis,
	}
}

// calculateMealDetailDynamicPerMealBudget estimates a per-meal budget from the
// remaining monthly budget and the number of planned meals left this month.
func calculateMealDetailDynamicPerMealBudget(
	monthlyBudget float64,
	currentMonthSpent float64,
	now time.Time,
) float64 {
	remainingBudget := monthlyBudget - currentMonthSpent
	if remainingBudget <= 0 {
		return 0
	}

	firstOfNextMonth := time.Date(
		now.Year(),
		now.Month()+1,
		1,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	todayStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		now.Location(),
	)

	daysRemaining := int(firstOfNextMonth.Sub(todayStart).Hours() / 24)
	if daysRemaining < 1 {
		daysRemaining = 1
	}

	const plannedMealsPerDay = 3
	return remainingBudget / float64(daysRemaining*plannedMealsPerDay)
}

// copyTrimmedStringSlice returns a clean copy of a string slice without blank entries.
func copyTrimmedStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		cleaned = append(cleaned, value)
	}

	return cleaned
}

// copyTrimmedMealDetailHealthFlags returns a clean copy of health flags for prompt input.
func copyTrimmedMealDetailHealthFlags(values map[string]string) map[string]string {
	cleaned := map[string]string{}

	for condition, flag := range values {
		condition = strings.TrimSpace(condition)
		flag = strings.TrimSpace(flag)

		if condition == "" || flag == "" {
			continue
		}

		cleaned[condition] = flag
	}

	return cleaned
}

// selectMealDetailPreferenceLocation chooses the saved location used for meal detail context.
// Weekdays prefer work/school, while weekends prefer home. If the preferred
// location is blank, it falls back to the other saved location.
func selectMealDetailPreferenceLocation(
	preferences interfaces.PreferenceResponse,
	now time.Time,
) (string, string) {
	homeLocation := strings.TrimSpace(preferences.HomeLocation)
	workSchoolLocation := strings.TrimSpace(preferences.WorkSchoolLocation)

	isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday
	if isWeekend {
		if homeLocation != "" {
			return homeLocation, "home"
		}

		if workSchoolLocation != "" {
			return workSchoolLocation, "fallback_work_school"
		}

		return "", "unavailable"
	}

	if workSchoolLocation != "" {
		return workSchoolLocation, "work_school"
	}

	if homeLocation != "" {
		return homeLocation, "fallback_home"
	}

	return "", "unavailable"
}
