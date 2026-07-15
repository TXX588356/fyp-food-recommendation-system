package restaurant

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestBuildRestaurantQuery(t *testing.T) {
	g := NewWithT(t)

	got := buildRestaurantQuery(interfaces.RestaurantSearchInput{
		MealName: "  Chicken Rice ",
		Location: " Pavilion Kuala Lumpur ",
	})

	g.Expect(got).To(Equal("Chicken Rice restaurant near Pavilion Kuala Lumpur"))
}

func TestBuildSerpAPIMapsURL(t *testing.T) {
	g := NewWithT(t)

	got, err := buildSerpAPIMapsURL("test-key", "Chicken Rice restaurant near Pavilion Kuala Lumpur")

	g.Expect(err).NotTo(HaveOccurred())
	parsedURL, err := url.Parse(got)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parsedURL.Scheme).To(Equal("https"))
	g.Expect(parsedURL.Host).To(Equal("serpapi.com"))
	g.Expect(parsedURL.Path).To(Equal("/search"))
	g.Expect(parsedURL.Query().Get("engine")).To(Equal("google_maps"))
	g.Expect(parsedURL.Query().Get("api_key")).To(Equal("test-key"))
	g.Expect(parsedURL.Query().Get("q")).To(Equal("Chicken Rice restaurant near Pavilion Kuala Lumpur"))
}

func TestParseOpenNow(t *testing.T) {
	g := NewWithT(t)

	open := parseOpenNow("Open ⋅ Closes 9 PM")
	closed := parseOpenNow("Closed ⋅ Opens 10 AM")
	unknown := parseOpenNow("Hours might differ")

	g.Expect(open).NotTo(BeNil())
	g.Expect(*open).To(BeTrue())
	g.Expect(closed).NotTo(BeNil())
	g.Expect(*closed).To(BeFalse())
	g.Expect(unknown).To(BeNil())
}

func TestBuildGoogleMapsURL(t *testing.T) {
	g := NewWithT(t)

	g.Expect(buildGoogleMapsURL("ChIJ test/place")).To(Equal("https://www.google.com/maps/place/?q=place_id:ChIJ+test%2Fplace"))
	g.Expect(buildGoogleMapsURL(" ")).To(BeEmpty())
}

func TestMapSerpAPILocalResults(t *testing.T) {
	g := NewWithT(t)

	results := []serpAPILocalResult{
		{Title: " "},
		{
			Title:     "  Dolly Dim Sum ",
			Address:   " Pavilion Kuala Lumpur ",
			Rating:    4.5,
			Reviews:   120,
			Price:     "$$ ",
			OpenState: "Open ⋅ Closes 10 PM",
			Thumbnail: " https://example.test/image.jpg ",
			PlaceID:   "place-1",
		},
		{Title: "Second Result"},
	}

	got := mapSerpAPILocalResults(results, 1)

	g.Expect(got).To(HaveLen(1))
	g.Expect(got[0].Name).To(Equal("Dolly Dim Sum"))
	g.Expect(got[0].Address).To(Equal("Pavilion Kuala Lumpur"))
	g.Expect(got[0].Rating).To(Equal(4.5))
	g.Expect(got[0].ReviewCount).To(Equal(120))
	g.Expect(got[0].Price).To(Equal("$$"))
	g.Expect(got[0].OpenNow).NotTo(BeNil())
	g.Expect(*got[0].OpenNow).To(BeTrue())
	g.Expect(got[0].ThumbnailURL).To(Equal("https://example.test/image.jpg"))
	g.Expect(got[0].SourceURL).To(Equal("https://www.google.com/maps/place/?q=place_id:place-1"))
}

func TestSearchRestaurantsReturnsUnavailableWithoutAPIKey(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient(" ")

	got, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
	g.Expect(got.Restaurants).To(BeEmpty())
}

func TestSearchRestaurantsReturnsUnavailableWithoutMealOrLocation(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient("test-key")

	got, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: " ",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
	g.Expect(got.Restaurants).To(BeEmpty())
}

func TestSearchRestaurantsMapsSuccessfulResponse(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient("test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			g.Expect(request.URL.Scheme).To(Equal("https"))
			g.Expect(request.URL.Host).To(Equal("serpapi.com"))
			g.Expect(request.URL.Query().Get("engine")).To(Equal("google_maps"))
			g.Expect(request.URL.Query().Get("api_key")).To(Equal("test-key"))
			g.Expect(request.URL.Query().Get("q")).To(Equal("Chicken Rice restaurant near Pavilion Kuala Lumpur"))

			return jsonResponse(http.StatusOK, `{
				"local_results": [
					{
						"title": "Dolly Dim Sum",
						"address": "Lot 1",
						"rating": 4.4,
						"reviews": 88,
						"price": "$$",
						"open_state": "Closed ⋅ Opens 10 AM",
						"thumbnail": "https://example.test/thumb.jpg",
						"place_id": "place-123"
					},
					{
						"title": "Second Restaurant"
					}
				]
			}`), nil
		}),
	}

	got, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
		Limit:    1,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.Status).To(Equal(interfaces.RestaurantLookupOK))
	g.Expect(got.Restaurants).To(HaveLen(1))
	g.Expect(got.Restaurants[0].Name).To(Equal("Dolly Dim Sum"))
	g.Expect(got.Restaurants[0].OpenNow).NotTo(BeNil())
	g.Expect(*got.Restaurants[0].OpenNow).To(BeFalse())
}

func TestSearchRestaurantsReturnsNoResultsForEmptyLocalResults(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient("test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"local_results": []}`), nil
		}),
	}

	got, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.Status).To(Equal(interfaces.RestaurantLookupNoResults))
	g.Expect(got.Restaurants).To(BeEmpty())
}

func TestSearchRestaurantsReturnsErrorForSerpAPIErrorPayload(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient("test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"error": "quota exceeded"}`), nil
		}),
	}

	_, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).To(MatchError("quota exceeded"))
}

func TestSearchRestaurantsReturnsErrorForNonSuccessStatus(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient("test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusTooManyRequests, `{}`), nil
		}),
	}

	_, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).To(MatchError("serpapi returned status 429"))
}

func TestSearchRestaurantsReturnsTransportError(t *testing.T) {
	g := NewWithT(t)

	client := NewSerpAPIClient("test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		}),
	}

	_, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).To(MatchError("Get \"https://serpapi.com/search?api_key=test-key&engine=google_maps&q=Chicken+Rice+restaurant+near+Pavilion+Kuala+Lumpur\": network down"))
}

func TestNoopSearcherReturnsUnavailable(t *testing.T) {
	g := NewWithT(t)

	got, err := NoopSearcher{}.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
		MealName: "Chicken Rice",
		Location: "Pavilion Kuala Lumpur",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
	g.Expect(got.Restaurants).To(BeEmpty())
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
