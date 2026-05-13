package foodapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchFoodReturnsFirstResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "Nasi Lemak" {
			t.Fatalf("expected query Nasi Lemak, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Fatalf("expected API key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[{"name":"Nasi Lemak"},{"name":"Other"}],"count":2}`))
	}))
	defer server.Close()

	client := NewKaloriClientWithBaseURL("test-key", server.URL, server.Client())
	got, found, err := client.SearchFood(context.Background(), "Nasi Lemak")
	if err != nil {
		t.Fatalf("SearchFood returned error: %v", err)
	}
	if !found {
		t.Fatal("expected result to be found")
	}
	if got.Count != 1 {
		t.Fatalf("expected count to be trimmed to 1, got %d", got.Count)
	}
	if len(got.Data) != 1 {
		t.Fatalf("expected data to be trimmed to 1 item, got %d", len(got.Data))
	}
}

func TestSearchFoodReturnsNotFoundForEmptyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[],"count":0}`))
	}))
	defer server.Close()

	client := NewKaloriClientWithBaseURL("test-key", server.URL, server.Client())
	_, found, err := client.SearchFood(context.Background(), "Unknown")
	if err != nil {
		t.Fatalf("SearchFood returned error: %v", err)
	}
	if found {
		t.Fatal("expected empty data to return found=false")
	}
}

func TestSearchFoodReturnsErrorForInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewKaloriClientWithBaseURL("test-key", server.URL, server.Client())
	_, _, err := client.SearchFood(context.Background(), "Nasi Lemak")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}
