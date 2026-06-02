package foodapi

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"strings"
)

type mockKaloriClient struct {
	foods map[string]interfaces.FoodSearchResult
}

// Ensure that the mock is interchangeable with the real Kalori client
var _ interfaces.FoodSearcher = (*mockKaloriClient)(nil)

// NewMockKaloriClient creates an in-memory Kalori replacement for development
func NewMockKaloriClient() interfaces.FoodSearcher {
	return &mockKaloriClient{
		foods: map[string]interfaces.FoodSearchResult{
			"nasi lemak": {
				ID:       "mock-nasi-lemak",
				Name:     "Nasi Lemak",
				Tags:     []string{"rice"},
				Calories: 655,
				FatG:     24,
				ProteinG: 16,
				CarbsG:   84,
			},
			"roti canai": {
				ID:       "mock-roti-canai",
				Name:     "Roti Canai",
				Tags:     []string{"roti"},
				Calories: 301,
				FatG:     10,
				ProteinG: 7,
				CarbsG:   46,
			},
			"wantan mee": {
				ID:       "mock-wantan-mee",
				Name:     "Wantan Mee",
				Tags:     []string{"noodles", "chinese"},
				Calories: 411,
				FatG:     12,
				ProteinG: 18,
				CarbsG:   58,
			},
			"bubur ayam": {
				ID:       "mock-bubur-ayam",
				Name:     "Bubur Ayam",
				Tags:     []string{"rice"},
				Calories: 372,
				FatG:     12,
				ProteinG: 20,
				CarbsG:   45,
			},
			"oatmeal": {
				ID:       "mock-oatmeal",
				Name:     "Oatmeal",
				Tags:     []string{"breakfast", "grains", "healthy"},
				Calories: 158,
				FatG:     3,
				ProteinG: 6,
				CarbsG:   27,
			},
		},
	}
}

// SearchFood performs a case-insensitive exact-name lookup in the mock dataset.
func (c *mockKaloriClient) SearchFood(ctx context.Context, query string) (interfaces.FoodSearchResult, bool, error) {
	food, found := c.foods[normalizeFoodQuery(query)]
	return food, found, nil
}

// normalizeFoodQuery makes mock lookups insensitive to casing and whitespace.
func normalizeFoodQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}
