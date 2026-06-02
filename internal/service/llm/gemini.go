package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"strings"

	"google.golang.org/genai"
)

const defaultModel = "gemini-2.5-flash-lite"

var riskLevels = map[string]bool{
	"LOW":    true,
	"MEDIUM": true,
	"HIGH":   true,
}

var healthFlags = map[string]bool{
	"SAFE":    true,
	"CAUTION": true,
	"AVOID":   true,
}

type Client struct {
	client *genai.Client
	model  string
}

// NewClient creates a Gemini meal-generation client using the default model.
func NewClient(client *genai.Client) Client {
	return Client{
		client: client,
		model:  defaultModel,
	}
}

// GenerateMeals builds a user-specific prompt, sends it to Gemini, and parses
// the generated meal candidates into the external-integration response shape.
func (c Client) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	prompt := BuildMealRecommendationPrompt(input)

	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return interfaces.GeminiMealsResponse{}, err
	}

	return ParseMeals(result.Text(), input.HealthConcerns)
}

// BuildMealRecommendationPrompt formats the user context and output rules into the prompt
func BuildMealRecommendationPrompt(input interfaces.MealPromptInput) string {
	return fmt.Sprintf(`You are generating meal recommendation for a user.
	SCOPE:
	- Generate meal candidates only.
	- Do not generate restaurants or location availability.
	- Do not generate food tags, preference tags, calories, protein, carbohydrates, or fat.
	- A food API will provide tags and macro nutrition values after matching.
	- Prefer meals commonly available in Malaysia. You may include suitable international meals.

	USER CONTEXT:
	- Goal: %s
	- Dietary restrictions: %s
	- Health concerns: %s
	- Preferred meal tags: %s
	- Meal category: %s
	- Monthly meal budget: RM%.2f
	- Current-month spending: RM%.2f
	- Remaining monthly budget: RM%.2f
	- Per-meal budget: RM%.2f

	HEALTH CONCERN MAPPINGS:
	- high_blood_pressure: focus on lower-sodium meals
	- diabetes: focus on lower-sugar meals with moderate carbohydrates
	- gout: avoid high-purine meals

	Generate exactly 6 meals suitable for the selected meal category.

	For each meal:
	1. Return the shortest recognizable base food name suitable for exact food API search.
	Do not append flavours, toppings, cooking styles, or ingredients.
	Prefer names with at most 3 words.

	GOOD: "Oatmeal", "Roti Canai", "Nasi Lemak", "Bubur Ayam", "Tom Yum", "Wantan Mee"
	BAD: "Pancake Butter Maple Syrup", "Nasi Minyak Ayam Masak Merah", "Mee Goreng Pedas Tambah Telur"

	2. Return up to 3 alternative exact search terms when useful.
	Return an empty array if the meal has one obvious fixed name.

	3. Estimate a typical Malaysian price range in RM.
	Use non-negative numeric values.
	The minimum must not exceed the maximum.

	4. Estimate typical sodium, sugar, and purine risk levels.
	Use only: "LOW", "MEDIUM", or "HIGH".

	5. Return health_flags only for health concerns listed in USER CONTEXT.
	Do not add flags for any other condition.
	Use only: "SAFE", "CAUTION", or "AVOID".

	Return only valid JSON. Do not wrap the response in markdown.

	Example:
	{
	"meals": [
		{
		"name": "Nasi Lemak",
		"alternative_search_terms": [],
		"estimated_price_range": {
			"min": 5.00,
			"max": 8.00
		},
		"sodium_level": "HIGH",
		"sugar_level": "MEDIUM",
		"purine_risk": "LOW",
		"health_flags": {
			"diabetes": "CAUTION"
		}
		}
	]
	}`,
		displayValue(input.Goal),
		displayList(input.DietaryRestrictions),
		displayList(input.HealthConcerns),
		displayList(input.PreferredMealTags),
		displayValue(input.MealCategory),
		input.MonthlyMealBudget,
		input.CurrentMonthSpent,
		input.RemainingBudget,
		input.PerMealBudget,
	)
}

// ParseMeals removes optional Markdown fencing, decodes Gemini's JSON response,
// and validates the generated candidates before returning them to the caller.
func ParseMeals(text string, expectedHealthConcerns []string) (interfaces.GeminiMealsResponse, error) {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var response interfaces.GeminiMealsResponse
	if err := json.Unmarshal([]byte(cleaned), &response); err != nil {
		return interfaces.GeminiMealsResponse{}, fmt.Errorf("parse Gemini meal response: %w", err)
	}

	if err := ValidateMeals(response, expectedHealthConcerns); err != nil {
		return interfaces.GeminiMealsResponse{}, fmt.Errorf("validate Gemini meal response: %w", err)
	}

	return response, nil
}

// ValidateMeals checks response-level constraints and validates each generated
// candidate against the health concerns requested in the prompt.
func ValidateMeals(response interfaces.GeminiMealsResponse, expectedHealthConcerns []string) error {
	if len(response.Meals) != 6 {
		return fmt.Errorf("expected exactly 6 meals, got %d", len(response.Meals))
	}

	expectedFlags := make(map[string]bool, len(expectedHealthConcerns))
	for _, concern := range expectedHealthConcerns {
		expectedFlags[concern] = true
	}

	for index, meal := range response.Meals {
		if err := validateMeal(meal, expectedFlags); err != nil {
			return fmt.Errorf("meal %d: %w", index, err)
		}
	}

	return nil
}

// validateMeal checks the fields of one generated meal candidate.
func validateMeal(meal interfaces.GeneratedMeal, expectedFlags map[string]bool) error {
	if strings.TrimSpace(meal.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if len(meal.AlternativeSearchTerms) > 3 {
		return fmt.Errorf("alternative search terms cannot exceed 3")
	}

	for _, term := range meal.AlternativeSearchTerms {
		if strings.TrimSpace(term) == "" {
			return fmt.Errorf("alternative search terms cannot be empty")
		}
	}

	priceRange := meal.EstimatedPriceRange
	if priceRange.Min < 0 || priceRange.Max < 0 {
		return fmt.Errorf("estimated price range cannot be negative")
	}

	if priceRange.Min > priceRange.Max {
		return fmt.Errorf("estimated price range min cannot exceed max")
	}

	if !riskLevels[meal.SodiumLevel] {
		return fmt.Errorf("unsupported sodium level %q", meal.SodiumLevel)
	}

	if !riskLevels[meal.SugarLevel] {
		return fmt.Errorf("unsupported sugar level %q", meal.SugarLevel)
	}
	if !riskLevels[meal.PurineRisk] {
		return fmt.Errorf("unsupported purine risk %q", meal.PurineRisk)
	}

	for concern, flag := range meal.HealthFlags {
		if !expectedFlags[concern] {
			return fmt.Errorf("unsupported health flag %q", concern)
		}

		if !healthFlags[flag] {
			return fmt.Errorf("unsupported health flag %q for %q", flag, concern)
		}
	}

	return nil
}

// displayList converts an optional list into readable prompt text
func displayList(values []string) string {
	if len(values) == 0 {
		return "none"
	}

	return strings.Join(values, ", ")
}

// displayValue converts an optional value into readable prompt text
func displayValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}
