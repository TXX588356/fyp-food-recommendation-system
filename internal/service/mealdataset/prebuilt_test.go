package mealdataset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMealDataset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Meal Dataset Suite")
}

var _ = Describe("Prebuilt meal searcher", func() {
	var (
		ctx      context.Context
		userID   uuid.UUID
		searcher *PrebuiltSearcher
	)

	BeforeEach(func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "meals.json")

		data := []byte(`[
		{
			"id": "nasi-lemak",
			"name": "Nasi Lemak",
			"alternative_search_terms": ["Malaysian coconut rice"],
			"calories": 494,
			"protein": 13,
			"carbs": 80,
			"fat": 14,
			"serving": "1 plate (350g)",
			"category": "Rice",
			"image_url": "https://example.com/nasi-lemak.jpg"
		},
		{
			"id": "nasi-lemak-ayam-goreng",
			"name": "Nasi Lemak Ayam Goreng",
			"calories": 744,
			"protein": 28,
			"carbs": 85,
			"fat": 32,
			"serving": "1 plate (450g)",
			"category": "Rice"
		},
		{
			"id": "tempeh-bowl",
			"name": "Tempeh Bowl",
			"calories": 430,
			"protein": 24,
			"carbs": 52,
			"fat": 12,
			"serving": "1 bowl (350g)",
			"category": ["Vegan", "Proteins"]
		},
		{
			"id": "tofu",
			"name": "Tofu",
			"calories": 76,
			"protein": 8,
			"carbs": 2,
			"fat": 5,
			"serving": "100g",
			"category": "Proteins"
		}
		]`)

		err := os.WriteFile(path, data, 0o600)
		Expect(err).NotTo(HaveOccurred())

		searcher, err = NewPrebuiltSearcher(path)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
		userID = uuid.New()
	})

	It("should search exact meal name", func() {
		results, err := searcher.Search(ctx, "nasi lemak")
		Expect(err).NotTo(HaveOccurred())

		Expect(results).To(HaveLen(1))
		Expect(results[0].Name).To(Equal("Nasi Lemak"))
		Expect(results[0].Calories).To(Equal(float64(494)))
	})

	It("should return partial matches when exact meal name is not found", func() {
		results, err := searcher.Search(ctx, "ayam goreng")
		Expect(err).NotTo(HaveOccurred())

		Expect(results).To(HaveLen(1))
		Expect(results[0].Name).To(Equal("Nasi Lemak Ayam Goreng"))
	})

	It("should return fuzzy matches when the query has a typo", func() {
		results, err := searcher.Search(ctx, "nasi lemak ayam gorng")
		Expect(err).NotTo(HaveOccurred())

		Expect(results).To(HaveLen(1))
		Expect(results[0].Name).To(Equal("Nasi Lemak Ayam Goreng"))
	})

	It("should search alternative meal terms", func() {
		results, err := searcher.Search(ctx, "Malaysian coconut rice")
		Expect(err).NotTo(HaveOccurred())

		Expect(results).To(HaveLen(1))
		Expect(results[0].Name).To(Equal("Nasi Lemak"))
	})

	It("should not match a generated dish to a single ingredient", func() {
		results, err := searcher.Search(ctx, "Tofu Ramen")
		Expect(err).NotTo(HaveOccurred())

		Expect(results).To(BeEmpty())
	})

	It("should adapt prebuilt meals to food search results", func() {
		food, found, err := searcher.SearchFood(ctx, userID, "Nasi Lemak")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())

		Expect(food.ID).To(Equal("nasi-lemak"))
		Expect(food.Name).To(Equal("Nasi Lemak"))
		Expect(food.Tags).To(Equal([]string{"Rice"}))
		Expect(food.Calories).To(Equal(float64(494)))
		Expect(food.FatG).To(Equal(float64(14)))
		Expect(food.ProteinG).To(Equal(float64(13)))
		Expect(food.CarbsG).To(Equal(float64(80)))
		Expect(food.ImageURL).To(Equal("https://example.com/nasi-lemak.jpg"))
	})

	It("should adapt array categories to food search result tags", func() {
		food, found, err := searcher.SearchFood(ctx, userID, "Tempeh Bowl")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())

		Expect(food.Name).To(Equal("Tempeh Bowl"))
		Expect(food.Tags).To(Equal([]string{"Vegan", "Proteins"}))
	})

	It("should return found false when no prebuilt meal matches", func() {
		_, found, err := searcher.SearchFood(ctx, userID, "Unknown Meal")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeFalse())
	})
})
