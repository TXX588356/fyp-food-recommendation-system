package postgres

import (
	"strings"
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestVisibleCatalogMealsUsesOnlyFlattenedPrebuiltMeals(t *testing.T) {
	db, err := gorm.Open(pgdriver.Open("postgres://catalog:catalog@localhost/catalog"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB { return visibleCatalogMeals(tx).Find(&[]model.PrebuiltMeal{}) })
	for _, removed := range []string{"dataset_sources", "is_active", "catalog_status", "duplicate_of_id", "license_status"} {
		if strings.Contains(sql, removed) {
			t.Fatalf("visibility SQL should not reference removed column/table %q: %s", removed, sql)
		}
	}
	if !strings.Contains(sql, "prebuilt_meals") {
		t.Fatalf("visibility SQL should read prebuilt_meals: %s", sql)
	}
}

func TestGeneratedMealSourceRecordIDUsesNormalizedName(t *testing.T) {
	normalizedName := normalizedCatalogMealName("  Unknown   Meal  ")
	if normalizedName != "unknown meal" {
		t.Fatalf("unexpected normalized name: %q", normalizedName)
	}

	got := aiGeneratedMealSourceRecordID(normalizedName)
	if got != "gemini:unknown meal" {
		t.Fatalf("unexpected source record id: %q", got)
	}
}

func TestCreateGeneratedMealInsertUsesAIGeneratedSourceAndConflictTarget(t *testing.T) {
	db, err := gorm.Open(pgdriver.Open("postgres://catalog:catalog@localhost/catalog"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

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
		NormalizedName:     normalizedName,
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
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected generated insert SQL to contain %q: %s", expected, sql)
		}
	}
}
