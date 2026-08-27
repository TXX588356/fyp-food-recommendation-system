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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRestaurantService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Restaurant Service Suite")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

var _ = Describe("SerpAPI restaurant search", func() {
	Describe("buildRestaurantQuery", func() {
		It("trims input and builds the local restaurant search query", func() {
			got := buildRestaurantQuery(interfaces.RestaurantSearchInput{
				MealName: "  Chicken Rice ",
				Location: " Pavilion Kuala Lumpur ",
			})

			Expect(got).To(Equal("Chicken Rice restaurant near Pavilion Kuala Lumpur"))
		})
	})

	Describe("buildSerpAPIMapsURL", func() {
		It("builds a Google Maps SerpAPI search URL", func() {
			got, err := buildSerpAPIMapsURL("test-key", "Chicken Rice restaurant near Pavilion Kuala Lumpur")

			Expect(err).NotTo(HaveOccurred())
			parsedURL, err := url.Parse(got)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedURL.Scheme).To(Equal("https"))
			Expect(parsedURL.Host).To(Equal("serpapi.com"))
			Expect(parsedURL.Path).To(Equal("/search"))
			Expect(parsedURL.Query().Get("engine")).To(Equal("google_maps"))
			Expect(parsedURL.Query().Get("api_key")).To(Equal("test-key"))
			Expect(parsedURL.Query().Get("q")).To(Equal("Chicken Rice restaurant near Pavilion Kuala Lumpur"))
		})
	})

	Describe("parseOpenNow", func() {
		It("maps open, closed, and unknown hours text", func() {
			open := parseOpenNow("Open ⋅ Closes 9 PM")
			closed := parseOpenNow("Closed ⋅ Opens 10 AM")
			unknown := parseOpenNow("Hours might differ")

			Expect(open).NotTo(BeNil())
			Expect(*open).To(BeTrue())
			Expect(closed).NotTo(BeNil())
			Expect(*closed).To(BeFalse())
			Expect(unknown).To(BeNil())
		})
	})

	Describe("buildGoogleMapsURL", func() {
		It("builds a place URL from a place ID", func() {
			Expect(buildGoogleMapsURL("ChIJ test/place")).To(Equal("https://www.google.com/maps/place/?q=place_id:ChIJ+test%2Fplace"))
		})

		It("returns empty when the place ID is blank", func() {
			Expect(buildGoogleMapsURL(" ")).To(BeEmpty())
		})
	})

	Describe("mapSerpAPILocalResults", func() {
		It("maps local results, trims fields, skips blank titles, and applies the limit", func() {
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

			Expect(got).To(HaveLen(1))
			Expect(got[0].Name).To(Equal("Dolly Dim Sum"))
			Expect(got[0].Address).To(Equal("Pavilion Kuala Lumpur"))
			Expect(got[0].Rating).To(Equal(4.5))
			Expect(got[0].ReviewCount).To(Equal(120))
			Expect(got[0].Price).To(Equal("$$"))
			Expect(got[0].OpenNow).NotTo(BeNil())
			Expect(*got[0].OpenNow).To(BeTrue())
			Expect(got[0].ThumbnailURL).To(Equal("https://example.test/image.jpg"))
			Expect(got[0].SourceURL).To(Equal("https://www.google.com/maps/place/?q=place_id:place-1"))
		})
	})

	Describe("NewSerpAPIClient", func() {
		It("bypasses TLS verification only for the restaurant lookup HTTP client", func() {
			client := NewSerpAPIClient("test-key")

			transport, ok := client.httpClient.Transport.(*http.Transport)
			Expect(ok).To(BeTrue())
			Expect(transport.TLSClientConfig).NotTo(BeNil())
			Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeTrue())

			defaultTransport, ok := http.DefaultTransport.(*http.Transport)
			Expect(ok).To(BeTrue())
			if defaultTransport.TLSClientConfig != nil {
				Expect(defaultTransport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			}
		})
	})

	Describe("SearchRestaurants", func() {
		It("returns unavailable when the API key is blank", func() {
			client := NewSerpAPIClient(" ")

			got, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
				MealName: "Chicken Rice",
				Location: "Pavilion Kuala Lumpur",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
			Expect(got.Restaurants).To(BeEmpty())
		})

		It("returns unavailable when meal name or location is blank", func() {
			client := NewSerpAPIClient("test-key")

			got, err := client.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
				MealName: " ",
				Location: "Pavilion Kuala Lumpur",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
			Expect(got.Restaurants).To(BeEmpty())
		})

		It("maps a successful SerpAPI response", func() {
			client := NewSerpAPIClient("test-key")
			client.httpClient = &http.Client{
				Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					Expect(request.URL.Scheme).To(Equal("https"))
					Expect(request.URL.Host).To(Equal("serpapi.com"))
					Expect(request.URL.Query().Get("engine")).To(Equal("google_maps"))
					Expect(request.URL.Query().Get("api_key")).To(Equal("test-key"))
					Expect(request.URL.Query().Get("q")).To(Equal("Chicken Rice restaurant near Pavilion Kuala Lumpur"))

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

			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status).To(Equal(interfaces.RestaurantLookupOK))
			Expect(got.Restaurants).To(HaveLen(1))
			Expect(got.Restaurants[0].Name).To(Equal("Dolly Dim Sum"))
			Expect(got.Restaurants[0].OpenNow).NotTo(BeNil())
			Expect(*got.Restaurants[0].OpenNow).To(BeFalse())
		})

		It("returns no results when SerpAPI has no local results", func() {
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

			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status).To(Equal(interfaces.RestaurantLookupNoResults))
			Expect(got.Restaurants).To(BeEmpty())
		})

		It("returns an error when SerpAPI returns an error payload", func() {
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

			Expect(err).To(MatchError("quota exceeded"))
		})

		It("returns an error when SerpAPI returns a non-success status", func() {
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

			Expect(err).To(MatchError("serpapi returned status 429"))
		})

		It("redacts the API key from transport errors", func() {
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

			Expect(err).To(MatchError("serpapi request failed: Get \"https://serpapi.com/search?api_key=<redacted>&engine=google_maps&q=Chicken+Rice+restaurant+near+Pavilion+Kuala+Lumpur\": network down"))
		})
	})

	Describe("redactSerpAPIKey", func() {
		It("redacts API keys embedded in URLs", func() {
			got := redactSerpAPIKey(`Get "https://serpapi.com/search?api_key=secret-key&engine=google_maps": tls failed`)

			Expect(got).To(Equal(`Get "https://serpapi.com/search?api_key=<redacted>&engine=google_maps": tls failed`))
			Expect(got).NotTo(ContainSubstring("secret-key"))
		})
	})
})

var _ = Describe("NoopSearcher", func() {
	Describe("SearchRestaurants", func() {
		It("returns unavailable without performing a lookup", func() {
			got, err := NoopSearcher{}.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
				MealName: "Chicken Rice",
				Location: "Pavilion Kuala Lumpur",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
			Expect(got.Restaurants).To(BeEmpty())
		})
	})
})

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
