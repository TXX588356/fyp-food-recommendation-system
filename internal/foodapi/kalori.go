package foodapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultKaloriBaseURL = "https://api.kalori-api.my"

type KaloriSearchResponse struct {
	Success bool              `json:"success"`
	Data    []json.RawMessage `json:"data"`
	Count   int               `json:"count"`
}

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewKaloriClient(apiKey string, client *http.Client) Client {
	return NewKaloriClientWithBaseURL(apiKey, defaultKaloriBaseURL, client)
}

func NewKaloriClientWithBaseURL(apiKey string, baseURL string, client *http.Client) Client {
	return Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c Client) SearchFood(ctx context.Context, mealName string) (KaloriSearchResponse, bool, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v1/foods/search")
	if err != nil {
		return KaloriSearchResponse{}, false, err
	}

	query := endpoint.Query()
	query.Set("q", mealName)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return KaloriSearchResponse{}, false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return KaloriSearchResponse{}, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return KaloriSearchResponse{}, false, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return KaloriSearchResponse{}, false, fmt.Errorf("Kalori search for %q failed with status %d: %s", mealName, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if !json.Valid(body) {
		return KaloriSearchResponse{}, false, fmt.Errorf("Kalori search for %q returned invalid JSON: %s", mealName, strings.TrimSpace(string(body)))
	}

	var result KaloriSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return KaloriSearchResponse{}, false, err
	}

	if len(result.Data) == 0 {
		return KaloriSearchResponse{}, false, nil
	}

	result.Data = result.Data[:1]
	result.Count = 1

	return result, true, nil
}
