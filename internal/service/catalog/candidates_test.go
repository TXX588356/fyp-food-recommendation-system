package catalog

import (
	"context"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CandidateSearcher", func() {
	It("should return ranked catalog candidates", func() {
		nasiLemakID := uuid.New()
		nasiLemakAyamID := uuid.New()

		// The repository returns a weaker candidate first and a stronger exact
		// candidate second. The candidate searcher should reorder them by rank.
		repo := &testRepository{
			meals: []model.PrebuiltMeal{
				{
					ID:                 nasiLemakAyamID,
					Name:               "Nasi Lemak Ayam Goreng",
					CategoryCodes:      []string{"rice_dishes"},
					ServingDescription: "1 plate",
					Calories:           float(744),
					ProteinG:           float(28),
					CarbsG:             float(85),
					FatG:               float(32),
				},
				{
					ID:                 nasiLemakID,
					Name:               "Nasi Lemak",
					CategoryCodes:      []string{"rice_dishes"},
					ServingDescription: "1 plate",
					Calories:           float(494),
					ProteinG:           float(13),
					CarbsG:             float(80),
					FatG:               float(14),
				},
			},
		}

		service := NewService(repo, testResolver{})
		searcher := NewCandidateSearcher(service)

		candidates, err := searcher.SearchFoodCandidates(
			context.Background(),
			uuid.New(),
			[]string{"Nasi Lemak"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(2))

		// Exact canonical name should rank first.
		Expect(candidates[0].Food.ID).To(Equal(nasiLemakID.String()))
		Expect(candidates[0].Food.Name).To(Equal("Nasi Lemak"))
		Expect(candidates[0].MatchKind).To(Equal(interfaces.FoodMatchExactName))
		Expect(candidates[0].MatchedTerm).To(Equal("Nasi Lemak"))
		Expect(candidates[0].Score).To(Equal(float64(1)))

		// Partial/fuzzy longer candidate remains eligible but ranks lower.
		Expect(candidates[1].Food.ID).To(Equal(nasiLemakAyamID.String()))
		Expect(candidates[1].Food.Name).To(Equal("Nasi Lemak Ayam Goreng"))
	})

	It("should skip catalog meals with incomplete required macros", func() {
		completeID := uuid.New()

		repo := &testRepository{
			meals: []model.PrebuiltMeal{
				{
					ID:                 uuid.New(),
					Name:               "Nasi Lemak",
					CategoryCodes:      []string{"rice_dishes"},
					ServingDescription: "1 plate",
					Calories:           float(494),
					ProteinG:           nil, // Missing required macro; should be skipped.
					CarbsG:             float(80),
					FatG:               float(14),
				},
				{
					ID:                 completeID,
					Name:               "Nasi Lemak Ayam",
					CategoryCodes:      []string{"rice_dishes"},
					ServingDescription: "1 plate",
					Calories:           float(600),
					ProteinG:           float(25),
					CarbsG:             float(82),
					FatG:               float(20),
				},
			},
		}

		service := NewService(repo, testResolver{})
		searcher := NewCandidateSearcher(service)

		candidates, err := searcher.SearchFoodCandidates(
			context.Background(),
			uuid.New(),
			[]string{"Nasi Lemak"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].Food.ID).To(Equal(completeID.String()))
	})

	It("should deduplicate catalog candidates by food ID and keep the strongest match", func() {
		mealID := uuid.New()

		// Same ID appears twice because the same meal can be returned for
		// multiple generated search terms.
		repo := &testRepository{
			meals: []model.PrebuiltMeal{
				catalogTestMealWithID(mealID, "Wantan Mee", "1"),
				catalogTestMealWithID(mealID, "Wantan Mee", "1"),
			},
		}

		service := NewService(repo, testResolver{})
		searcher := NewCandidateSearcher(service)

		candidates, err := searcher.SearchFoodCandidates(
			context.Background(),
			uuid.New(),
			[]string{"Wantan Mee", "Wantan Mee Soup"},
			5,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].Food.ID).To(Equal(mealID.String()))
		Expect(candidates[0].MatchKind).To(Equal(interfaces.FoodMatchExactName))
		Expect(candidates[0].Score).To(Equal(float64(1)))
	})

	It("should return at most the requested limit", func() {
		repo := &testRepository{
			meals: []model.PrebuiltMeal{
				catalogTestMeal("1", "Chicken Rice"),
				catalogTestMeal("2", "Chicken Rice"),
				catalogTestMeal("3", "Chicken Rice"),
			},
		}

		service := NewService(repo, testResolver{})
		searcher := NewCandidateSearcher(service)

		candidates, err := searcher.SearchFoodCandidates(
			context.Background(),
			uuid.New(),
			[]string{"Chicken Rice"},
			2,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(2))
	})

	It("should return no candidates when limit is zero", func() {
		repo := &testRepository{
			meals: []model.PrebuiltMeal{
				catalogTestMeal("1", "Nasi Lemak"),
			},
		}

		service := NewService(repo, testResolver{})
		searcher := NewCandidateSearcher(service)

		candidates, err := searcher.SearchFoodCandidates(
			context.Background(),
			uuid.New(),
			[]string{"Nasi Lemak"},
			0,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(BeEmpty())
	})
})

// catalogTestMeal creates a complete catalog model for candidate-search tests.
func catalogTestMeal(seed string, name string) model.PrebuiltMeal {
	return catalogTestMealWithID(uuid.New(), name, seed)
}

// catalogTestMealWithID creates a complete catalog model using a specific ID.
// Use this when a test needs duplicate catalog rows to represent the same meal.
func catalogTestMealWithID(id uuid.UUID, name string, seed string) model.PrebuiltMeal {
	return model.PrebuiltMeal{
		ID:                 id,
		SourceCode:         "test",
		SourceRecordID:     seed,
		Name:               name,
		CategoryCodes:      []string{"test_category"},
		ServingDescription: "1 serving",
		Calories:           float(500),
		ProteinG:           float(20),
		CarbsG:             float(60),
		FatG:               float(15),
	}
}
