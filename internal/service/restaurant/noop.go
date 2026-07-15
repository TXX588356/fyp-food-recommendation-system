package restaurant

import (
	"context"
	"fyp/food-rs/internal/interfaces"
)

// NoopSearcher disables restaurant lookup while preserving meal detail responses.
type NoopSearcher struct{}

// SearchRestaurants returns unavailable without failing the meal detail flow.
func (NoopSearcher) SearchRestaurants(
	ctx context.Context,
	input interfaces.RestaurantSearchInput,
) (interfaces.RestaurantSearchResult, error) {
	return interfaces.RestaurantSearchResult{
		Status:      interfaces.RestaurantLookupUnavailable,
		Restaurants: []interfaces.RestaurantResult{},
	}, nil
}
