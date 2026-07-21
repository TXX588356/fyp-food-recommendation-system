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
	conditions := make([]string, 0)
	args := make([]any, 0)
	if value := strings.TrimSpace(query.Query); value != "" {
		/*
			Example:
			value = fried rice

			phrasePattern = %fried rice%
			condition = (name ILIKE ? OR normalized_name ILIKE ?)
			args = [%fried rice%, %fried rice%]

			strings.Fields(value) = ["fried", "rice"]

			Loop 1:
			wordPattern = "%fried%"
			wordConditions = (name ILIKE ? OR normalized_name ILIKE ?)
			wordArgs = ["%fried", "%fried%"]

			Loop 2:
			wordPattern = "%rice%"
			wordConditions = [(name ILIKE ? OR normalized_name ILIKE ?), (name ILIKE ? OR normalized_name ILIKE ?)],
			wordArgs = ["%fried", "fried%", "%rice", "rice%"]

			len(wordConditions) > 1  => true
			condition = name ILIKE ? OR normalized_name ILIKE ? OR (name ILIKE ? OR normalized_name ILIKE ? AND name ILIKE ? OR normalized_name ILIKE ?)
			args = [%fried rice%, %fried rice%, "%fried"%, "%fried%", "%rice%", "%rice%"]

			conditions =( name ILIKE ? OR normalized_name ILIKE ?) OR ((name ILIKE ? OR normalized_name ILIKE ?) AND (name ILIKE ? OR normalized_name ILIKE ?))
		*/

		phrasePattern := "%" + value + "%"
		condition := `(prebuilt_meals.name ILIKE ? OR prebuilt_meals.normalized_name ILIKE ?)`
		args = append(args, phrasePattern, phrasePattern)

		wordConditions := make([]string, 0)
		wordArgs := make([]any, 0)
		for _, word := range strings.Fields(value) {
			wordPattern := "%" + word + "%"
			wordConditions = append(wordConditions, `(prebuilt_meals.name ILIKE ? OR prebuilt_meals.normalized_name ILIKE ?)`)
			wordArgs = append(wordArgs, wordPattern, wordPattern)
		}
		if len(wordConditions) > 1 {
			condition = condition + ` OR (` + strings.Join(wordConditions, ` AND `) + `)`
			args = append(args, wordArgs...)
		}
		conditions = append(conditions, condition)
	}
	if query.Source != "" {
		conditions = append(conditions, "prebuilt_meals.source_code = ?")
		args = append(args, query.Source)
	}
	if len(query.Categories) > 0 {
		conditions = append(conditions, "prebuilt_meals.category_codes @> ?::text[]")
		args = append(args, pgTextArrayLiteral(query.Categories))
	}
	if query.AfterID != nil {
		conditions = append(conditions, "(prebuilt_meals.normalized_name, prebuilt_meals.id) > (?::text, ?::uuid)")
		args = append(args, query.AfterName, *query.AfterID)
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var ids []uuid.UUID
	sql := "SELECT prebuilt_meals.id FROM prebuilt_meals"
	if len(conditions) > 0 {
		sql += " WHERE " + strings.Join(conditions, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY prebuilt_meals.normalized_name, prebuilt_meals.id LIMIT %d", limit+1)
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&ids).Error; err != nil {
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
