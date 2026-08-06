package llm

import (
	"context"
	"errors"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

func (c Client) ExplainMealRecommendation(ctx context.Context, input interfaces.MealDetailExplanationInput) (string, error) {
	prompt := BuildMealDetailExplanationPrompt(input)
	slog.Info("gemini prompt", "operation", "explain_meal_recommendation", "model", c.model, "prompt", prompt)

	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return "", err
	}

	explanation := strings.TrimSpace(result.Text())
	if explanation == "" {
		return "", errors.New("empty meal explanation")
	}

	return explanation, nil
}

func BuildMealDetailExplanationPrompt(input interfaces.MealDetailExplanationInput) string {
	return fmt.Sprintf(`You are explaining a meal recommendation to a user.

Write exactly one concise paragraph within 3 to 5 sentences explaining why this meal fits this specific user.

MEAL:
- Name: %s
- Meal category: %s
- Calories: %.0f kcal
- Fat: %.2fg
- Protein: %.2fg
- Carbohydrates: %.2fg
- Estimated price range: RM%.2f - RM%.2f
- Sodium level: %s
- Sugar level: %s
- Purine risk: %s
- Health flags: %s

USER:
- Goal: %s
- Dietary restrictions: %s
- Health concerns: %s
- Preferred meal tags: %s
- Monthly meal budget: RM%.2f
- Current-month spending: RM%.2f
- Remaining monthly budget: RM%.2f
- Per-meal budget: RM%.2f
- Location context: %s
- Location basis: %s

RULES:
- Use only the facts above.
- The UI already shows calories, macros, and price. Do not restate raw numbers unless needed to explain a meaningful tradeoff.
- Start with the meal's practical food qualities, such as cooking style, heaviness, flavor profile, portion style, or when it is usually satisfying.
- Then connect those food qualities to the user's preferences, restrictions, budget, or health context.
- Do not make the explanation mostly a checklist of user profile matches.
- Explain the 2 strongest reasons this specific food makes sense for this user.
- If nutrition matters, interpret it in plain language instead of repeating exact macro values.
- If budget matters, explain whether it is comfortably within budget, slightly above target, or a reasonable tradeoff. Do not repeat every budget number.
- When judging budget fit, compare the upper end of the estimated price range against the per-meal budget. Only call it comfortably within budget if the upper end is at or below the per-meal budget.
- If there is a concern or tradeoff, mention it honestly and briefly.
- Mention at most one budget point and one health point.
- Prefer concrete food descriptions over generic praise.
- Do not invent restaurant availability.
- Do not invent nutrition values.
- Do not claim this meal treats, prevents, or cures any disease.
- Do not give medical advice.
- Do not use markdown.
- Return only the paragraph.`,
		displayValue(input.MealName),
		displayValue(input.MealCategory),
		input.Nutrition.Calories,
		input.Nutrition.FatG,
		input.Nutrition.ProteinG,
		input.Nutrition.CarbsG,
		input.PriceRange.Min,
		input.PriceRange.Max,
		displayValue(input.SodiumLevel),
		displayValue(input.SugarLevel),
		displayValue(input.PurineRisk),
		displayMap(input.HealthFlags),
		displayValue(input.UserGoal),
		displayList(input.DietaryRestrictions),
		displayList(input.HealthConcerns),
		displayList(input.PreferredMealTags),
		input.MonthlyMealBudget,
		input.CurrentMonthSpent,
		input.RemainingBudget,
		input.PerMealBudget,
		displayValue(input.Location),
		displayValue(input.LocationBasis),
	)
}
