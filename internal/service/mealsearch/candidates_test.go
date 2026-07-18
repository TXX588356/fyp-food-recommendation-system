package mealsearch

import (
	"context"
	"errors"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// candidateSearchCall records one call made to the fake catalog candidate
// searcher. The tests use it to verify that the aggregator forwards the full
// query list and requested limit to the catalog layer.
type candidateSearchCall struct {
	userID  uuid.UUID
	queries []string
	limit   int
}

// fakeFoodCandidateSearcher is a small hand-written test double for
// interfaces.FoodCandidateSearcher.
//
// We use this instead of a generated mock so the tests can stay focused on the
// aggregator behavior without depending on mockery output for the new interface.
type fakeFoodCandidateSearcher struct {
	candidates []interfaces.FoodMatchCandidate
	err        error
	calls      []candidateSearchCall
}

// SearchFoodCandidates records the call and returns the fake catalog candidates
// configured by the test.
func (s *fakeFoodCandidateSearcher) SearchFoodCandidates(
	ctx context.Context,
	userID uuid.UUID,
	queries []string,
	limit int,
) ([]interfaces.FoodMatchCandidate, error) {
	s.calls = append(s.calls, candidateSearchCall{
		userID:  userID,
		queries: append([]string(nil), queries...),
		limit:   limit,
	})

	if s.err != nil {
		return nil, s.err
	}

	return s.candidates, nil
}

var _ = Describe("CandidateSearcher", func() {
	var (
		ctx               context.Context
		userID            uuid.UUID
		customMealService *mocks.CustomMealService
		catalogSearcher   *fakeFoodCandidateSearcher
		searcher          interfaces.FoodCandidateSearcher
	)

	BeforeEach(func() {
		ctx = context.Background()
		userID = uuid.New()
		customMealService = mocks.NewCustomMealService(GinkgoT())
		catalogSearcher = &fakeFoodCandidateSearcher{}
		searcher = NewCandidateSearcher(customMealService, catalogSearcher)
	})

	It("should merge ranked custom and catalog candidates", func() {
		// Custom meals should be searched once for each unique query.
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return([]*interfaces.CustomMealResponse{
				{
					ID:               "custom-1",
					Name:             "Nasi Lemak",
					Calories:         530,
					FatG:             18,
					ProteinG:         20,
					CarbsG:           75,
					MealCategoryTags: []string{"rice_dishes"},
					ImageURL:         "https://example.test/custom.jpg",
				},
			}, nil).
			Once()

		// Catalog candidates are already ranked by the catalog candidate searcher.
		catalogSearcher.candidates = []interfaces.FoodMatchCandidate{
			{
				Food: interfaces.FoodSearchResult{
					ID:       "catalog-1",
					Name:     "Nasi Lemak Ayam Goreng",
					Tags:     []string{"rice_dishes"},
					Calories: 744,
					FatG:     32,
					ProteinG: 28,
					CarbsG:   85,
				},
				MatchKind:   interfaces.FoodMatchFuzzy,
				MatchedTerm: "Nasi Lemak",
				Score:       0.81,
			},
		}

		candidates, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Nasi Lemak"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(2))

		// The exact custom candidate should rank before the fuzzy catalog candidate.
		Expect(candidates[0].Food.ID).To(Equal("custom-1"))
		Expect(candidates[0].Food.Name).To(Equal("Nasi Lemak"))
		Expect(candidates[0].Food.Source).To(Equal("custom"))
		Expect(candidates[0].MatchKind).To(Equal(interfaces.FoodMatchExactName))
		Expect(candidates[0].MatchedTerm).To(Equal("Nasi Lemak"))
		Expect(candidates[0].Score).To(Equal(float64(1)))

		Expect(candidates[1].Food.ID).To(Equal("catalog-1"))
		Expect(candidates[1].Food.Name).To(Equal("Nasi Lemak Ayam Goreng"))

		// The aggregator should call the catalog searcher once with the original
		// query list, not once per custom query.
		Expect(catalogSearcher.calls).To(HaveLen(1))
		Expect(catalogSearcher.calls[0].userID).To(Equal(userID))
		Expect(catalogSearcher.calls[0].queries).To(Equal([]string{"Nasi Lemak"}))
		Expect(catalogSearcher.calls[0].limit).To(Equal(5))
	})

	It("should search custom meals once per unique non-empty generated term", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return([]*interfaces.CustomMealResponse{}, nil).
			Once()

		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Roti Canai").
			Return([]*interfaces.CustomMealResponse{}, nil).
			Once()

		_, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Nasi Lemak", " nasi   lemak ", "", "Roti Canai"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(catalogSearcher.calls).To(HaveLen(1))

		// Catalog search receives the original query list. The catalog candidate
		// searcher owns its own deduplication.
		Expect(catalogSearcher.calls[0].queries).To(Equal([]string{
			"Nasi Lemak",
			" nasi   lemak ",
			"",
			"Roti Canai",
		}))
	})

	It("should deduplicate custom candidates by namespaced custom ID and keep the strongest match", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Wantan Mee").
			Return([]*interfaces.CustomMealResponse{
				customMeal("custom-1", "Wantan Mee"),
			}, nil).
			Once()

		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Wantan Mee Soup").
			Return([]*interfaces.CustomMealResponse{
				customMeal("custom-1", "Wantan Mee"),
			}, nil).
			Once()

		candidates, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Wantan Mee", "Wantan Mee Soup"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].Food.ID).To(Equal("custom-1"))
		Expect(candidates[0].MatchKind).To(Equal(interfaces.FoodMatchExactName))
		Expect(candidates[0].Score).To(Equal(float64(1)))
	})

	It("should not deduplicate custom and catalog candidates with the same public ID", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Chicken Rice").
			Return([]*interfaces.CustomMealResponse{
				customMeal("same-id", "Chicken Rice"),
			}, nil).
			Once()

		catalogSearcher.candidates = []interfaces.FoodMatchCandidate{
			{
				Food: interfaces.FoodSearchResult{
					ID:       "same-id",
					Name:     "Chicken Rice",
					Tags:     []string{"rice_dishes"},
					Calories: 600,
					FatG:     20,
					ProteinG: 30,
					CarbsG:   70,
				},
				MatchKind:   interfaces.FoodMatchExactName,
				MatchedTerm: "Chicken Rice",
				Score:       1,
			},
		}

		candidates, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Chicken Rice"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(2))
		Expect(candidates[0].Food.ID).To(Equal("same-id"))
		Expect(candidates[1].Food.ID).To(Equal("same-id"))
	})

	It("should return at most the requested limit after merging sources", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Chicken Rice").
			Return([]*interfaces.CustomMealResponse{
				customMeal("custom-1", "Chicken Rice"),
				customMeal("custom-2", "Chicken Rice"),
			}, nil).
			Once()

		catalogSearcher.candidates = []interfaces.FoodMatchCandidate{
			catalogCandidate("catalog-1", "Chicken Rice", 1),
			catalogCandidate("catalog-2", "Chicken Rice", 1),
		}

		candidates, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Chicken Rice"},
			2,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(2))
	})

	It("should return no candidates when limit is zero", func() {
		candidates, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Nasi Lemak"},
			0,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(BeEmpty())

		// No downstream search should happen when the limit is zero.
		Expect(catalogSearcher.calls).To(BeEmpty())
	})

	It("should return an error when custom meal search fails", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return(nil, errors.New("custom search unavailable")).
			Once()

		_, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Nasi Lemak"},
			5,
		)

		Expect(err).To(MatchError(ContainSubstring("custom search unavailable")))
	})

	It("should return an error when catalog candidate search fails", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return([]*interfaces.CustomMealResponse{}, nil).
			Once()

		catalogSearcher.err = errors.New("catalog search unavailable")

		_, err := searcher.SearchFoodCandidates(
			ctx,
			userID,
			[]string{"Nasi Lemak"},
			5,
		)

		Expect(err).To(MatchError(ContainSubstring("catalog search unavailable")))
	})
})

// customMeal creates a complete custom meal test fixture.
func customMeal(id string, name string) *interfaces.CustomMealResponse {
	return &interfaces.CustomMealResponse{
		ID:               id,
		Name:             name,
		Calories:         500,
		FatG:             15,
		ProteinG:         25,
		CarbsG:           65,
		MealCategoryTags: []string{"test_category"},
		ImageURL:         "https://example.test/custom.jpg",
	}
}

// catalogCandidate creates a catalog candidate test fixture.
// The catalog candidate searcher normally creates these; tests use this helper
// to focus on aggregation behavior.
func catalogCandidate(id string, name string, score float64) interfaces.FoodMatchCandidate {
	return interfaces.FoodMatchCandidate{
		Food: interfaces.FoodSearchResult{
			ID:       id,
			Name:     name,
			Source:   "prebuilt",
			Tags:     []string{"test_category"},
			Calories: 500,
			FatG:     15,
			ProteinG: 25,
			CarbsG:   65,
		},
		MatchKind:   interfaces.FoodMatchExactName,
		MatchedTerm: name,
		Score:       score,
	}
}
