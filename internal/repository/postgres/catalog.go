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
			condition = prebuilt_meals.name ILIKE ?
			args = ["%fried rice%"]

			strings.Fields(value) = ["fried", "rice"]

			For each word, add one name condition:
			wordConditions = [
				prebuilt_meals.name ILIKE ?,
				prebuilt_meals.name ILIKE ?,
			]
			wordArgs = ["%fried%", "%rice%"]

			Because there is more than one word, append an order-independent
			all-words match:

			condition =
				prebuilt_meals.name ILIKE ?
				OR (
					prebuilt_meals.name ILIKE ?
					AND prebuilt_meals.name ILIKE ?
				)

				args = ["%fried rice", "fried%", "rice%"]

				This lets query "fried rice" match name "rice fried" because
				both words appear somewhere in the name.

		*/

		phrasePattern := "%" + value + "%"
		condition := `prebuilt_meals.name ILIKE ?`
		args = append(args, phrasePattern)

		wordConditions := make([]string, 0)
		wordArgs := make([]any, 0)

		for _, word := range strings.Fields(value) {
			wordPattern := "%" + word + "%"

			// Each query word must appear somewhere in the name
			// This keeps "a b" able to match with "b a"
			wordConditions = append(wordConditions, `prebuilt_meals.name ILIKE ?`)
			wordArgs = append(wordArgs, wordPattern)
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
		conditions = append(conditions,
			// key-set pagination
			// create a 2-column tuple (e.g. ("apple"), 123acbdef) and compare with the
			// provided meal name and meal id lexicographically
			// 1. if name > input meal: true
			// 2. if name = input meal: compare id
			// 3. else false
			"(lower(btrim(prebuilt_meals.name)), prebuilt_meals.id) > (?::text, ?::uuid)",
		)
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

	sql += fmt.Sprintf(" ORDER BY lower(btrim(prebuilt_meals.name)), prebuilt_meals.id LIMIT %d", limit+1)
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
		Order("lower(btrim(prebuilt_meals.name)), prebuilt_meals.id").Find(&meals).Error
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
