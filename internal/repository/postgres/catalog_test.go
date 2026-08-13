package postgres

import (
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestPostgresRepositories(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgres Repository Suite")
}

var _ = Describe("catalog repository SQL", func() {
	It("should use only flattened prebuilt meals for visible catalog meals", func() {
		db, err := gorm.Open(pgdriver.Open("postgres://catalog:catalog@localhost/catalog"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
		Expect(err).NotTo(HaveOccurred())

		sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB { return visibleCatalogMeals(tx).Find(&[]model.PrebuiltMeal{}) })

		for _, removed := range []string{"dataset_sources", "is_active", "catalog_status", "duplicate_of_id", "license_status"} {
			Expect(sql).NotTo(ContainSubstring(removed))
		}
		Expect(sql).To(ContainSubstring("prebuilt_meals"))
	})

	It("should use normalized names for generated meal source record IDs", func() {
		normalizedName := normalizedCatalogMealName("  Unknown   Meal  ")
		Expect(normalizedName).To(Equal("unknown meal"))

		got := aiGeneratedMealSourceRecordID(normalizedName)
		Expect(got).To(Equal("gemini:unknown meal"))
	})

	It("should insert generated meals with AI source and conflict target", func() {
		db, err := gorm.Open(pgdriver.Open("postgres://catalog:catalog@localhost/catalog"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
		Expect(err).NotTo(HaveOccurred())

		input := interfaces.GeneratedCatalogMealInput{
			Name:               "Unknown Meal",
			CategoryCodes:      []string{"rice_dishes"},
			ServingDescription: "1 serving",
			Calories:           500,
			ProteinG:           24,
			CarbsG:             62,
			FatG:               18,
		}
		normalizedName := normalizedCatalogMealName(input.Name)
		meal := model.PrebuiltMeal{
			SourceCode:         aiGeneratedMealSourceCode,
			SourceRecordID:     aiGeneratedMealSourceRecordID(normalizedName),
			Name:               input.Name,
			CategoryCodes:      model.StringArray(input.CategoryCodes),
			ServingDescription: input.ServingDescription,
			Calories:           floatPtr(input.Calories),
			ProteinG:           floatPtr(input.ProteinG),
			CarbsG:             floatPtr(input.CarbsG),
			FatG:               floatPtr(input.FatG),
		}

		sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "source_code"}, {Name: "source_record_id"}},
				DoNothing: true,
			}).Create(&meal)
		})

		for _, expected := range []string{
			`INSERT INTO "prebuilt_meals"`,
			`"source_code"`,
			`"source_record_id"`,
			`"serving_description"`,
			`ON CONFLICT ("source_code","source_record_id") DO NOTHING`,
			`ai_generated`,
			`gemini:unknown meal`,
		} {
			Expect(sql).To(ContainSubstring(expected))
		}
	})
})
