package foodapi

import (
	"context"
	"encoding/json"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultKaloriBaseURL = "https://api.kalori-api.my"

type kaloriSearchResponse struct {
	Success bool               `json:"success"`
	Data    []kaloriFoodResult `json:"data"`
	Count   int                `json:"count"`
}

type kaloriFoodResult struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
	Calories float64  `json:"calories"`
	FatG     float64  `json:"fat_g"`
	ProteinG float64  `json:"protein_g"`
	CarbsG   float64  `json:"carbs_g"`
}

type client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

var _ interfaces.FoodSearcher = (*client)(nil)

// NewKaloriClient creates a Kalori API searcher using the production base URL.
func NewKaloriClient(apiKey string, httpClient *http.Client) interfaces.FoodSearcher {
	return NewKaloriClientWithBaseURL(apiKey, defaultKaloriBaseURL, httpClient)
}

// NewKaloriClientWithBaseURL creates a Kalori API searcher with a configurable URL.
// Use this constructor with httptest.Server in unit tests.
func NewKaloriClientWithBaseURL(apiKey string, baseURL string, httpClient *http.Client) interfaces.FoodSearcher {
	return &client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  httpClient,
	}
}

// SearchFood searches Kalori using a food name and returns the first match.
// It returns found=false when Kalori does not provide a matching result.
func (c *client) SearchFood(ctx context.Context, query string) (interfaces.FoodSearchResult, bool, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v1/foods/search")
	if err != nil {
		return interfaces.FoodSearchResult{}, false, fmt.Errorf("build Kalori search URL: %w", err)
	}

	params := endpoint.Query()
	params.Set("q", query)
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return interfaces.FoodSearchResult{}, false, fmt.Errorf("build Kalori search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return interfaces.FoodSearchResult{}, false, fmt.Errorf("search Kalori food %q: %w", query, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return interfaces.FoodSearchResult{}, false, fmt.Errorf("read Kalori search response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return interfaces.FoodSearchResult{}, false, fmt.Errorf("kalori search for %q failed with status %d: %s", query, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response kaloriSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return interfaces.FoodSearchResult{}, false, fmt.Errorf("parse Kalori search response for %q: %w", query, err)
	}

	if len(response.Data) == 0 {
		return interfaces.FoodSearchResult{}, false, nil
	}

	food := response.Data[0]

	return interfaces.FoodSearchResult{
		ID:       food.ID,
		Name:     food.Name,
		Tags:     food.Tags,
		Calories: food.Calories,
		FatG:     food.FatG,
		ProteinG: food.ProteinG,
		CarbsG:   food.CarbsG,
	}, true, nil
}
