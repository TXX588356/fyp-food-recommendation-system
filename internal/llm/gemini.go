package llm

import (
	"context"
	"encoding/json"
	"strings"

	"google.golang.org/genai"
)

const defaultModel = "gemini-2.5-flash-lite"

type GeminiMealsResponse struct {
	Meals []Meal `json:"meals"`
}

type Meal struct {
	Name                   string            `json:"name"`
	Format                 string            `json:"format"`
	Cuisines               []string          `json:"cuisines"`
	SodiumLevel            string            `json:"sodium_level"`
	SugarLevel             string            `json:"sugar_level"`
	PurineRisk             string            `json:"purine_risk"`
	HealthFlags            map[string]string `json:"health_flags"`
	MatchedPreferences     []string          `json:"matched_preferences"`
	AlternativeSearchTerms []string          `json:"alternative_search_terms"`
}

type Client struct {
	client *genai.Client
	model  string
}

func NewClient(client *genai.Client) Client {
	return Client{
		client: client,
		model:  defaultModel,
	}
}

func (c Client) GenerateMeals(ctx context.Context) (GeminiMealsResponse, error) {
	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(mealRecommendationPrompt), nil)
	if err != nil {
		return GeminiMealsResponse{}, err
	}

	return ParseMeals(result.Text())
}

func ParseMeals(text string) (GeminiMealsResponse, error) {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var meals GeminiMealsResponse
	if err := json.Unmarshal([]byte(cleaned), &meals); err != nil {
		return GeminiMealsResponse{}, err
	}
	return meals, nil
}

const mealRecommendationPrompt = `You are generating Malaysian meal recommendations for a user.

USER CONTEXT:
- Goal: {eat healthier}
- Dietary restrictions: {halal, seafood}
- Health concerns: {diabetes}
- Preferences: {chinese, noodles, kuih}
- meal budget: RM300 monthly, RM50 used, remaining 50.00 daily max

FIXED PREFERENCE TAGS (use ONLY these exact strings):
American
Basics
Breakfast
Chinese
Condiments
Desserts
Drinks
French
Fruits
Grains
Greek
Healthy
Indian
Italian
Japanese
Korean
Kuih
Legumes
Meat
Mexican
Middle Eastern
Noodles
Nuts
Proteins
Rice 
Roti
Seafood
Seeds
Snacks
Soups
Spanish
Thai
Vegetables
Vietnamese
Western


HEALTH CONCERN MAPPINGS (use these for filtering hints):
- High blood pressure → focus on low sodium
- Diabetes → focus on low sugar, moderate carbs
- Gout → avoid high purine foods

Generate 6 {dinner} meal candidates. For each meal:
1. Return ONLY the base food name — no flavours, no toppings, no cooking styles, 
   no ingredients appended. The name should be the shortest recognisable form 
   of the dish that someone would search for in a food database.

   GOOD: "Oatmeal", "Pancake", "Roti Canai", "Nasi Lemak", "Bubur Ayam", "Tom Yum", "Char Kuey Teow", "Wantan Mee"
   BAD:  "Oatmeal Buah Buahan", "Pancake Butter Maple Syrup", "Tom Yum Soup", "Char Kway Teow", "Wan Tan Mee"
         "Nasi Minyak Ayam Masak Merah", "Mee Goreng Pedas Tambah Telur"

   Rule: if the name has more than 3 words, it is probably too specific — shorten it.
2. Classify its format (rice/noodle/bread/soup/light/snack)
3. Classify its cuisines (malaysian/western/chinese/indian/fusion)
4. Estimate sodium, sugar, purine risk (LOW/MEDIUM/HIGH) based on typical preparation
5. Return health_flags ONLY for the health concerns listed in USER CONTEXT.
   Do not include flags for conditions the user does not have.
   Use only these values: "SAFE" | "CAUTION" | "AVOID"
6. Match it against the fixed preference tags — list only the tags that apply
7. The food API does not support fuzzy search. Generate 3 alternative search terms for each meal, except for meals that have obvious fixed name


Return ONLY valid JSON, no other text:
Example:
{
  "meals": [
    {
      "name": "Nasi Lemak",
      "format": "rice",
      "cuisines": ["malaysian"],
      "sodium_level": "HIGH",
      "health_flags": {
        "high_blood_pressure": "AVOID",
        "diabetes": "CAUTION",
        "gout": "SAFE",
        "weight_loss": "CAUTION"
      },
      "matched_preferences": ["rice-based meals", "local malaysian food"],
    },
  ]
}`
