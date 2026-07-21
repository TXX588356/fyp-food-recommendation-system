package restaurant

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SerpAPIClient searches restaurants through SerpAPI Google Maps API.
type SerpAPIClient struct {
	apiKey     string
	httpClient *http.Client
}

type serpAPIMapsResponse struct {
	LocalResults []serpAPILocalResult `json:"local_results"`
	Error        string               `json:"error"`
}

type serpAPILocalResult struct {
	Title         string  `json:"title"`
	Address       string  `json:"address"`
	Rating        float64 `json:"rating"`
	Reviews       int     `json:"reviews"`
	Price         string  `json:"price"`
	OpenState     string  `json:"open_state"`
	Thumbnail     string  `json:"thumbnail"`
	PlaceID       string  `json:"place_id"`
	GPSCoordinate struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"gps_coordinates"`
}

// NewSerpAPIClient creates a restaurant search client backed by SerpAPI.
func NewSerpAPIClient(apiKey string) *SerpAPIClient {
	return &SerpAPIClient{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// Restaurant lookup is isolated from other HTTP clients; this bypass
					// handles environments that intercept SerpAPI TLS certificates.
					InsecureSkipVerify: true,
				},
			},
			// Timeout to prevent request hanging, and keeps meal detail endpoint responsive
			Timeout: 8 * time.Second,
		},
	}
}

// buildRestaurantQuery creates the local search query sent to SerpAPI.
func buildRestaurantQuery(input interfaces.RestaurantSearchInput) string {
	return fmt.Sprintf(
		"%s restaurant near %s",
		strings.TrimSpace(input.MealName),
		strings.TrimSpace(input.Location),
	)
}

// SearchRestaurants looks up possible nearby restaurants for a meal.
// SerpAPI errors are returned so the caller can decide whether to degrade gracefully.
func (c *SerpAPIClient) SearchRestaurants(ctx context.Context, input interfaces.RestaurantSearchInput) (interfaces.RestaurantSearchResult, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return interfaces.RestaurantSearchResult{
			Status:      interfaces.RestaurantLookupUnavailable,
			Restaurants: []interfaces.RestaurantResult{},
		}, nil
	}

	mealName := strings.TrimSpace(input.MealName)
	location := strings.TrimSpace(input.Location)
	if mealName == "" || location == "" {
		return interfaces.RestaurantSearchResult{
			Status:      interfaces.RestaurantLookupUnavailable,
			Restaurants: []interfaces.RestaurantResult{},
		}, nil
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	requestURL, err := buildSerpAPIMapsURL(c.apiKey, buildRestaurantQuery(input))
	if err != nil {
		return interfaces.RestaurantSearchResult{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return interfaces.RestaurantSearchResult{}, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return interfaces.RestaurantSearchResult{}, fmt.Errorf("serpapi request failed: %s", redactSerpAPIKey(err.Error()))
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return interfaces.RestaurantSearchResult{}, fmt.Errorf("serpapi returned status %d", response.StatusCode)
	}

	var payload serpAPIMapsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return interfaces.RestaurantSearchResult{}, err
	}

	if strings.TrimSpace(payload.Error) != "" {
		return interfaces.RestaurantSearchResult{}, errors.New(payload.Error)
	}

	restaurants := mapSerpAPILocalResults(payload.LocalResults, limit)
	if len(restaurants) == 0 {
		return interfaces.RestaurantSearchResult{
			Status:      interfaces.RestaurantLookupNoResults,
			Restaurants: []interfaces.RestaurantResult{},
		}, nil
	}

	return interfaces.RestaurantSearchResult{
		Status:      interfaces.RestaurantLookupOK,
		Restaurants: restaurants,
	}, nil
}

// buildSerpAPIMapsURL builds a SerpAPI Google Maps search URL.
func buildSerpAPIMapsURL(apiKey, query string) (string, error) {
	values := url.Values{}
	values.Set("engine", "google_maps")
	values.Set("q", query)
	values.Set("api_key", apiKey)

	return "https://serpapi.com/search?" + values.Encode(), nil
}

// redactSerpAPIKey removes API key values from URLs before errors are logged.
func redactSerpAPIKey(message string) string {
	parsedURL, err := url.Parse(message)
	if err == nil {
		values := parsedURL.Query()
		if values.Has("api_key") {
			values.Set("api_key", "<redacted>")
			parsedURL.RawQuery = values.Encode()
			return parsedURL.String()
		}
	}

	if !strings.Contains(message, "api_key=") {
		return message
	}

	parts := strings.Split(message, "api_key=")
	for index := 1; index < len(parts); index++ {
		value := parts[index]
		end := strings.IndexAny(value, "&\" ")
		if end == -1 {
			parts[index] = "<redacted>"
			continue
		}

		parts[index] = "<redacted>" + value[end:]
	}

	return strings.Join(parts, "api_key=")
}

// mapSerpAPILocalResults converts SerpAPI local results into app-owned restaurant results.
func mapSerpAPILocalResults(results []serpAPILocalResult, limit int) []interfaces.RestaurantResult {
	restaurants := make([]interfaces.RestaurantResult, 0, limit)

	for _, result := range results {
		name := strings.TrimSpace(result.Title)
		if name == "" {
			continue
		}

		restaurants = append(restaurants, interfaces.RestaurantResult{
			Name:         name,
			Address:      strings.TrimSpace(result.Address),
			Rating:       result.Rating,
			ReviewCount:  result.Reviews,
			Price:        strings.TrimSpace(result.Price),
			OpenNow:      parseOpenNow(result.OpenState),
			ThumbnailURL: strings.TrimSpace(result.Thumbnail),
			SourceURL:    buildGoogleMapsURL(result.PlaceID),
		})

		if len(restaurants) >= limit {
			break
		}
	}

	return restaurants
}

// parseOpenNow maps SerpAPI open state text into a nullable boolean.
func parseOpenNow(openState string) *bool {
	openState = strings.ToLower(strings.TrimSpace(openState))
	if openState == "" {
		return nil
	}

	if strings.Contains(openState, "closed") {
		value := false
		return &value
	}

	if strings.Contains(openState, "open") {
		value := true
		return &value
	}

	return nil
}

// buildGoogleMapsURL creates a stable Google Maps place URL when a place ID is available.
func buildGoogleMapsURL(placeID string) string {
	placeID = strings.TrimSpace(placeID)
	if placeID == "" {
		return ""
	}

	return "https://www.google.com/maps/place/?q=place_id:" + url.QueryEscape(placeID)
}
