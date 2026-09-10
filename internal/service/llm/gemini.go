package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"sort"
	"strings"

	"google.golang.org/genai"
)

const defaultModel = "gemini-3.5-flash-lite"

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

var supportedAutocompleteDietaryTags = map[string]bool{
	"non_beef":     true,
	"halal":        true,
	"vegetarian":   true,
	"vegan":        true,
	"seafood_free": true,
	"nut_free":     true,
	"low_sugar":    true,
	"low_salt":     true,
	"low_fat":      true,
}

var supportedAutocompleteMealCategoryTags = map[string]bool{
	"malaysian":         true,
	"singaporean":       true,
	"indonesian":        true,
	"chinese":           true,
	"indian":            true,
	"thai":              true,
	"vietnamese":        true,
	"japanese":          true,
	"korean":            true,
	"middle_eastern":    true,
	"american":          true,
	"mexican":           true,
	"italian":           true,
	"french":            true,
	"greek":             true,
	"spanish":           true,
	"western":           true,
	"breakfast":         true,
	"rice_dishes":       true,
	"noodle_dishes":     true,
	"soups":             true,
	"stews":             true,
	"curries":           true,
	"stir_fries":        true,
	"grilled_roasted":   true,
	"fried_foods":       true,
	"salads":            true,
	"sandwiches_wraps":  true,
	"breads_flatbreads": true,
	"porridge":          true,
	"dumplings":         true,
	"snacks":            true,
	"desserts":          true,
	"beverages":         true,
	"condiments_sauces": true,
	"poultry":           true,
	"beef":              true,
	"pork":              true,
	"lamb":              true,
	"seafood":           true,
	"eggs":              true,
	"tofu_soy":          true,
	"legumes":           true,
	"vegetables":        true,
	"fruits":            true,
	"grains":            true,
	"dairy":             true,
	"nuts_seeds":        true,
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
	slog.Info("gemini prompt", "operation", "generate_meals", "model", c.model, "prompt", prompt)

	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return interfaces.GeminiMealsResponse{}, err
	}

	return ParseMeals(result.Text(), input.HealthConcerns)
}

func (c Client) AutocompleteCustomMeal(ctx context.Context, input interfaces.CustomMealAutocompleteInput) (interfaces.CustomMealAutocompleteResponse, error) {
	prompt := BuildCustomMealAutocompletePrompt(input.Name)
	slog.Info("gemini prompt", "operation", "autocomplete_custom_meal", "model", c.model, "prompt", prompt)

	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return interfaces.CustomMealAutocompleteResponse{}, err
	}

	return ParseCustomMealAutocomplete(result.Text())
}

func BuildCustomMealAutocompletePrompt(input string) string {
	return fmt.Sprintf(`You are generating custom meal detail for a user.
Meal name: %s

Estimate nutrition for one typical serving of exactly this meal.
Return only valid JSON. Do not include markdown or explanation.
Use exactly this JSON shape and these field names:
{
  "calories": 0,
  "fatG": 0,
  "proteinG": 0,
  "carbsG": 0,
  "dietaryRestrictionTags": [],
  "mealCategoryTags": []
}

Rules:
- First decide whether the meal name is actually food. Only continue if it clearly names an edible dish, beverage, ingredient, packaged food, or common menu item.
- Return exactly {"error":"unable to generate meal details"} when the name is not food, is ambiguous, is a brand/app/tool/company/person/place/fictional character/object, or looks like random text.
- Do not reinterpret, translate, or invent a dish from a non-food name. Names such as Gemini, ChatGPT, OpenAI, Google, Claude, Copilot, iPhone, Tesla, Batman, London, Dragon, and Laptop are invalid unless the user explicitly adds food context such as "cake", "rice", "drink", or "sandwich".
- If a name could refer to both food and a non-food entity, only estimate nutrition when the food meaning is clear from the full name.
- If you cannot estimate credible meal details for the provided meal name, return exactly {"error":"unable to generate meal details"}.
- Do not return mealName, carbs, fat, or protein keys.
- fatG, proteinG, and carbsG are grams.
- dietaryRestrictionTags is optional and may be empty.
- mealCategoryTags must contain at least one supported value.
- Use only these dietaryRestrictionTags: non_beef, halal, vegetarian, vegan, seafood_free, nut_free, low_sugar, low_salt, low_fat.
- Use only these mealCategoryTags: malaysian, singaporean, indonesian, chinese, indian, thai, vietnamese, japanese, korean, middle_eastern, american, mexican, italian, french, greek, spanish, western, breakfast, rice_dishes, noodle_dishes, soups, stews, curries, stir_fries, grilled_roasted, fried_foods, salads, sandwiches_wraps, breads_flatbreads, porridge, dumplings, snacks, desserts, beverages, condiments_sauces, poultry, beef, pork, lamb, seafood, eggs, tofu_soy, legumes, vegetables, fruits, grains, dairy, nuts_seeds.
- Use app tag codes exactly. gluten-free is not valid; dairy-free is not valid; lunch and dinner are not valid mealCategoryTags.
- If the meal is Kimchi, use Korean/vegetable-style tags such as korean and vegetables, not lunch or dinner.
`, strings.TrimSpace(input))
}

// BuildMealRecommendationPrompt formats the user context and output rules into the prompt
func BuildMealRecommendationPrompt(input interfaces.MealPromptInput) string {
	return fmt.Sprintf(`You are generating meal recommendation for a user.
	SCOPE:
	- Generate meal candidates only.
	- Do not generate restaurants or location availability.
	- Do not generate food tags, preference tags, calories, protein, carbohydrates, or fat.
	- A food API will provide tags and macro nutrition values after matching.
	- Prefer meals commonly available in Malaysia. You may include suitable international meals if it is included in preferred meal tags.

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
	- Price market location: %s
	- Location basis: weekday/work-school or weekend/home

	USER RECENT MEAL HISTORY:
	- Recent meals: %s
	- Recently repeated meals: %s
	- Long-term learned categories: %s
	- Recent fatigued categories: %s

	HISTORY RULES: 
	- Rank recently repeated meals lower unless they are strongly aligned with user goals.
	- Treat long-term learned categories as mild preference signals, weaker than explicit preferred meal tags.
	- Avoid repeating exact meals and avoid generating several meals from recent fatigued categories.
	- Prefer variety across meal names and meaningful meal categories.
	- Do not use calorie totals to decide recommendations; calories are tracked separately in meal logs.

	HEALTH CONCERN MAPPINGS:
	- high_blood_pressure: focus on lower-sodium meals
	- diabetes: focus on lower-sugar meals with moderate carbohydrates
	- gout: avoid high-purine meals

	MEAL TIMING GUIDANCE:
	- Breakfast should be practical morning food: moderate calories, not overly oily, and suitable before work or school.
	- Lunch can include more filling meals with higher calories, carbohydrates, and digestive load because it is usually the main daytime meal.
	- For dinner, avoid very heavy fried rice/noodle dishes, oversized rice portions, and meals that are typically greasy unless the user's goal requires higher intake.
	- Still respect the user's goal: muscle_gain can use higher-protein dinners, but keep dinner less heavy than lunch when possible.

	Generate exactly 6 meals suitable for the selected meal category.

	For each meal:
	1. Return the shortest recognizable base food name suitable for exact food API search.
	Do not append flavours, toppings, cooking styles, or ingredients.
	Prefer names with at most 3 words.

	2. Return up to 3 alternative exact search terms when useful.
	Return an empty array if the meal has one obvious fixed name.

	3. Estimate a typical Malaysian price range in RM.
	Use non-negative numeric values.
	The minimum must not exceed the maximum.
	Estimate the typical dine-in/takeaway market price in RM for this meal around the selected location.
	Do not lower the estimate just to fit the user's per-meal budget.
	If the meal is an international cuisine item, estimate based on typical Malaysian restaurant/cafe pricing for that cuisine near the selected location.

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
		input.PriceMarketLocation,
		displayHistoryList(input.History.RecentMealNames),
		displayHistoryList(input.History.RepeatedMealNames),
		displayCategoryCounts(input.History.LearnedCategoryCounts),
		displayCategoryCounts(input.History.FatiguedCategoryCounts),
	)
}

var promptIgnoredHistoryCategories = map[string]bool{
	"malaysian":      true,
	"singaporean":    true,
	"indonesian":     true,
	"chinese":        true,
	"indian":         true,
	"thai":           true,
	"vietnamese":     true,
	"japanese":       true,
	"korean":         true,
	"middle_eastern": true,
	"american":       true,
	"mexican":        true,
	"italian":        true,
	"french":         true,
	"greek":          true,
	"spanish":        true,
	"western":        true,
}

func displayCategoryCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}

	type categoryCount struct {
		category string
		count    int
	}

	values := make([]categoryCount, 0, len(counts))
	for category, count := range counts {
		if category == "" || count <= 0 || promptIgnoredHistoryCategories[category] {
			continue
		}
		values = append(values, categoryCount{category: category, count: count})
	}

	if len(values) == 0 {
		return "none"
	}

	sort.Slice(values, func(i, j int) bool {
		if values[i].count == values[j].count {
			return values[i].category < values[j].category
		}
		return values[i].count > values[j].count
	})

	if len(values) > 8 {
		values = values[:8]
	}

	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%s %d", value.category, value.count))
	}

	return strings.Join(parts, ", ")
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

func ParseCustomMealAutocomplete(text string) (interfaces.CustomMealAutocompleteResponse, error) {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var errorResponse struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(cleaned), &errorResponse); err == nil && strings.TrimSpace(errorResponse.Error) != "" {
		return interfaces.CustomMealAutocompleteResponse{}, fmt.Errorf("%s", strings.TrimSpace(errorResponse.Error))
	}

	var response interfaces.CustomMealAutocompleteResponse
	decoder := json.NewDecoder(bytes.NewReader([]byte(cleaned)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return interfaces.CustomMealAutocompleteResponse{}, fmt.Errorf("parse AI autocomplete response: %w", err)
	}

	if err := ValidateCustomMealAutocomplete(response); err != nil {
		return interfaces.CustomMealAutocompleteResponse{}, fmt.Errorf("validate AI autocomplete response: %w", err)
	}

	return response, nil
}

func ValidateCustomMealAutocomplete(response interfaces.CustomMealAutocompleteResponse) error {
	if response.Calories <= 0 || response.Calories > 3000 {
		return fmt.Errorf("calories must be greater than 0 and no more than 3000")
	}

	if response.FatG < 0 || response.FatG > 500 {
		return fmt.Errorf("fatG must be between 0 and 500")
	}

	if response.ProteinG < 0 || response.ProteinG > 500 {
		return fmt.Errorf("proteinG must be between 0 and 500")
	}

	if response.CarbsG < 0 || response.CarbsG > 500 {
		return fmt.Errorf("carbsG must be between 0 and 500")
	}

	for _, tag := range response.DietaryRestrictionTags {
		if !supportedAutocompleteDietaryTags[tag] {
			return fmt.Errorf("unsupported dietary restriction tag: %s", tag)
		}
	}

	if len(response.MealCategoryTags) == 0 {
		return fmt.Errorf("meal category tags are required")
	}

	for _, tag := range response.MealCategoryTags {
		if !supportedAutocompleteMealCategoryTags[tag] {
			return fmt.Errorf("unsupported meal category tag: %s", tag)
		}
	}

	return nil
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

func displayHistoryList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func displayMap(values map[string]string) string {
	if len(values) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "" || value == "" {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, ", ")
}
