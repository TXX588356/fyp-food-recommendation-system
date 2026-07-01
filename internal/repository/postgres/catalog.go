package postgres

import (
	"context"
	"fmt"
	"strings"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type catalogRepository struct{ db *gorm.DB }

const aiGeneratedMealSourceCode = "ai_generated"

func NewCatalogPostgresRepository(db *gorm.DB) interfaces.CatalogRepository {
	return &catalogRepository{db: db}
}

func visibleCatalogMeals(db *gorm.DB) *gorm.DB {
	return db.Table("prebuilt_meals")
}

func normalizedCatalogMealName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(name))), " ")
}

func aiGeneratedMealSourceRecordID(normalizedName string) string {
	return "gemini:" + normalizedName
}

func (r *catalogRepository) SearchMeals(ctx context.Context, query interfaces.CatalogQuery) ([]model.PrebuiltMeal, error) {
	db := visibleCatalogMeals(r.db.WithContext(ctx)).Select("prebuilt_meals.id")
	if value := strings.TrimSpace(query.Query); value != "" {
		pattern := "%" + strings.ToLower(value) + "%"
		db = db.Where(`LOWER(prebuilt_meals.name) LIKE ? OR prebuilt_meals.normalized_name LIKE ?`, pattern, pattern)
	}
	if query.Source != "" {
		db = db.Where("prebuilt_meals.source_code = ?", query.Source)
	}
	if len(query.Categories) > 0 {
		db = db.Where("prebuilt_meals.category_codes @> ?::text[]", pgTextArrayLiteral(query.Categories))
	}
	if query.HasImage != nil {
		exists := `EXISTS (SELECT 1 FROM prebuilt_meal_images i WHERE i.prebuilt_meal_id = prebuilt_meals.id AND i.is_primary = TRUE AND i.match_status IN ('auto_accepted','approved'))`
		if *query.HasImage {
			db = db.Where(exists)
		} else {
			db = db.Where("NOT " + exists)
		}
	}
	if query.AfterID != nil {
		db = db.Where("(prebuilt_meals.normalized_name, prebuilt_meals.id) > (?, ?)", query.AfterName, *query.AfterID)
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var ids []uuid.UUID
	if err := db.Order("prebuilt_meals.normalized_name, prebuilt_meals.id").Limit(limit + 1).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return r.hydrateMeals(ctx, ids)
}

func (r *catalogRepository) hydrateMeals(ctx context.Context, ids []uuid.UUID) ([]model.PrebuiltMeal, error) {
	if len(ids) == 0 {
		return []model.PrebuiltMeal{}, nil
	}
	var meals []model.PrebuiltMeal
	err := r.db.WithContext(ctx).
		Preload("Images", "is_primary = TRUE AND match_status IN ?", []string{"auto_accepted", "approved"}).
		Where("prebuilt_meals.id IN ?", ids).
		Order("prebuilt_meals.normalized_name, prebuilt_meals.id").Find(&meals).Error
	return meals, err
}

func (r *catalogRepository) FindMeal(ctx context.Context, id uuid.UUID) (*model.PrebuiltMeal, error) {
	var ids []uuid.UUID
	if err := visibleCatalogMeals(r.db.WithContext(ctx)).Select("prebuilt_meals.id").Where("prebuilt_meals.id = ?", id).Scan(&ids).Error; err != nil {
		return nil, err
	}
	meals, err := r.hydrateMeals(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(meals) == 0 {
		return nil, interfaces.ErrCatalogNotFound
	}
	return &meals[0], nil
}

func (r *catalogRepository) ListCategories(ctx context.Context) ([]interfaces.CatalogCategory, error) {
	var categories []interfaces.CatalogCategory
	err := visibleCatalogMeals(r.db.WithContext(ctx)).
		Select("meal_categories.code, meal_categories.label, COUNT(DISTINCT prebuilt_meals.id) AS meal_count").
		Joins("JOIN meal_categories ON meal_categories.code = ANY(prebuilt_meals.category_codes) AND meal_categories.is_active = TRUE").
		Group("meal_categories.id").Order("meal_categories.display_order").Scan(&categories).Error
	return categories, err
}

func (r *catalogRepository) CategoryCodesExist(ctx context.Context, codes []string) (bool, error) {
	if len(codes) == 0 {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&model.MealCategory{}).Where("code IN ? AND is_active = TRUE", codes).Distinct("code").Count(&count).Error
	return count == int64(len(codes)), err
}

func (r *catalogRepository) CreateGeneratedMeal(ctx context.Context, input interfaces.GeneratedCatalogMealInput) (*model.PrebuiltMeal, error) {
	normalizedName := normalizedCatalogMealName(input.Name)
	if normalizedName == "" {
		return nil, interfaces.ErrInvalidCatalogQuery
	}

	meal := model.PrebuiltMeal{
		SourceCode:         aiGeneratedMealSourceCode,
		SourceRecordID:     aiGeneratedMealSourceRecordID(normalizedName),
		Name:               strings.TrimSpace(input.Name),
		NormalizedName:     normalizedName,
		CategoryCodes:      model.StringArray(input.CategoryCodes),
		ServingDescription: strings.TrimSpace(input.ServingDescription),
		Calories:           floatPtr(input.Calories),
		ProteinG:           floatPtr(input.ProteinG),
		CarbsG:             floatPtr(input.CarbsG),
		FatG:               floatPtr(input.FatG),
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_code"}, {Name: "source_record_id"}},
		DoNothing: true,
	}).
		Create(&meal).Error

	if err != nil {
		return nil, err
	}

	var saved model.PrebuiltMeal
	err = r.db.WithContext(ctx).
		Where("source_code = ? AND source_record_id = ?", meal.SourceCode, meal.SourceRecordID).
		First(&saved).Error

	if err != nil {
		return nil, err
	}

	return &saved, nil
}

func floatPtr(value float64) *float64 {
	return &value
}

func pgTextArrayLiteral(values []string) string {
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		escaped = append(escaped, `"`+strings.ReplaceAll(value, `"`, `\"`)+`"`)
	}
	return fmt.Sprintf("{%s}", strings.Join(escaped, ","))
}
