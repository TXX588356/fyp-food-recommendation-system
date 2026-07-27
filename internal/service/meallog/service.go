package meallog

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type service struct {
	mealLogRepo       interfaces.MealLogRepository
	customMealService interfaces.CustomMealService
	catalogService    interfaces.CatalogService
	preferenceService interfaces.PreferenceService
	now               func() time.Time
}

func NewService(mealLogRepo interfaces.MealLogRepository, customMealService interfaces.CustomMealService, catalogService interfaces.CatalogService, preferenceService interfaces.PreferenceService) interfaces.MealLogService {
	return &service{
		mealLogRepo:       mealLogRepo,
		customMealService: customMealService,
		catalogService:    catalogService,
		preferenceService: preferenceService,
		now:               time.Now,
	}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, input interfaces.MealLogInput) (*interfaces.MealLogResponse, error) {
	if err := validateMealLogInput(input); err != nil {
		return nil, err
	}

	mealID, err := uuid.Parse(input.MealID)
	if err != nil {
		return nil, errors.New("invalid meal id")
	}

	var log *model.MealLog

	switch input.Source {
	case interfaces.MealLogSourceCustom:
		log, err = s.buildCustomMealLog(ctx, userID, mealID, input)
	case interfaces.MealLogSourcePrebuilt:
		log, err = s.buildPrebuiltMealLog(ctx, userID, mealID, input)
	default:
		return nil, errors.New("unsupported log source")
	}

	if err != nil {
		return nil, err
	}

	saved, err := s.mealLogRepo.Create(ctx, log)
	if err != nil {
		return nil, err
	}

	return buildMealLogResponse(saved), nil
}

func (s *service) Update(ctx context.Context, userID, logID uuid.UUID, input interfaces.MealLogUpdateInput) (*interfaces.MealLogResponse, error) {
	if err := validateMealLogUpdateInput(input); err != nil {
		return nil, err
	}

	log, err := s.mealLogRepo.FindByIDAndUser(ctx, logID, userID)
	if err != nil {
		return nil, err
	}

	log.Price = input.Price
	log.EatenAt = input.EatenAt
	log.MealType = input.MealType

	updated, err := s.mealLogRepo.Update(ctx, log)
	if err != nil {
		return nil, err
	}

	return buildMealLogResponse(updated), nil
}

func (s *service) Delete(ctx context.Context, userID, logID uuid.UUID) error {
	// Ownership check to prevent deleting other user's meal log
	if _, err := s.mealLogRepo.FindByIDAndUser(ctx, logID, userID); err != nil {
		return err
	}

	return s.mealLogRepo.Delete(ctx, logID, userID)
}

func (s *service) buildCustomMealLog(ctx context.Context, userID uuid.UUID, mealID uuid.UUID, input interfaces.MealLogInput) (*model.MealLog, error) {
	meal, err := s.customMealService.FindVisibleByID(ctx, userID, mealID)
	if err != nil {
		return nil, err
	}

	return &model.MealLog{
		UserID:           userID,
		CustomMealItemID: &mealID,
		MealName:         meal.Name,
		Price:            input.Price,
		EatenAt:          input.EatenAt,
		MealType:         input.MealType,
		MealCategory:     model.StringArray(meal.MealCategoryTags),
		Calories:         meal.Calories,
		ProteinG:         meal.ProteinG,
		CarbsG:           meal.CarbsG,
		FatG:             meal.FatG,
	}, nil
}

func (s *service) buildPrebuiltMealLog(ctx context.Context, userID uuid.UUID, mealID uuid.UUID, input interfaces.MealLogInput) (*model.MealLog, error) {
	meal, err := s.catalogService.GetMeal(ctx, mealID)
	if err != nil {
		return nil, err
	}

	calories := 0.0
	if meal.SelectedNutrition.Calories != nil {
		calories = *meal.SelectedNutrition.Calories
	}

	proteinG := 0.0
	if meal.SelectedNutrition.ProteinG != nil {
		proteinG = *meal.SelectedNutrition.ProteinG
	}

	carbsG := 0.0
	if meal.SelectedNutrition.CarbsG != nil {
		carbsG = *meal.SelectedNutrition.CarbsG
	}

	fatG := 0.0
	if meal.SelectedNutrition.FatG != nil {
		fatG = *meal.SelectedNutrition.FatG
	}

	return &model.MealLog{
		UserID:         userID,
		PrebuiltMealID: &mealID,
		MealName:       meal.Name,
		Price:          input.Price,
		EatenAt:        input.EatenAt,
		MealType:       input.MealType,
		MealCategory:   model.StringArray(meal.Categories),
		Calories:       calories,
		ProteinG:       proteinG,
		CarbsG:         carbsG,
		FatG:           fatG,
	}, nil
}

func (s *service) GetMonth(ctx context.Context, userID uuid.UUID, month string) (*interfaces.MealLogMonthResponse, error) {
	start, end, err := parseMonth(month)
	if err != nil {
		return nil, err
	}

	logs, err := s.mealLogRepo.ListByUserAndMonth(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	summary := buildMonthSummary(month, logs)

	if !isPastMonth(start, s.now()) {
		preferences, err := s.preferenceService.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}

		remaining := preferences.MonthlyMealBudget - summary.TotalSpent
		if remaining < 0 {
			remaining = 0
		}

		summary.BudgetRemaining = &remaining
		summary.ShowBudgetRemaining = true
	}

	return &interfaces.MealLogMonthResponse{
		Summary: summary,
		Items:   buildMealLogResponses(logs),
	}, nil
}

func parseMonth(value string) (time.Time, time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 1, 0), nil
	}

	start, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("month must use YYYY-MM format")
	}

	return start, start.AddDate(0, 1, 0), nil
}

func isPastMonth(monthStart time.Time, now time.Time) bool {
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	selectedMonth := time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, now.Location())

	return selectedMonth.Before(currentMonth)
}

func buildMonthSummary(month string, logs []model.MealLog) interfaces.MealLogMonthSummary {
	totalSpent := 0.0
	totalCalories := 0.0

	for _, log := range logs {
		totalSpent += log.Price
		totalCalories += log.Calories
	}

	return interfaces.MealLogMonthSummary{
		Month:               month,
		TotalMealsEaten:     len(logs),
		TotalSpent:          totalSpent,
		TotalCalories:       totalCalories,
		ShowBudgetRemaining: false,
	}
}

func buildMealLogResponse(log *model.MealLog) *interfaces.MealLogResponse {
	var customMealItemID *string
	if log.CustomMealItemID != nil {
		value := log.CustomMealItemID.String()
		customMealItemID = &value
	}

	var prebuiltMealID *string
	if log.PrebuiltMealID != nil {
		value := log.PrebuiltMealID.String()
		prebuiltMealID = &value
	}

	return &interfaces.MealLogResponse{
		ID:               log.ID.String(),
		CustomMealItemID: customMealItemID,
		PrebuiltMealID:   prebuiltMealID,
		EatenAt:          log.EatenAt,
		MealName:         log.MealName,
		MealType:         log.MealType,
		Calories:         log.Calories,
		ProteinG:         log.ProteinG,
		CarbsG:           log.CarbsG,
		FatG:             log.FatG,
		Price:            log.Price,
		MealCategory:     []string(log.MealCategory),
	}
}

func buildMealLogResponses(logs []model.MealLog) []interfaces.MealLogResponse {
	responses := make([]interfaces.MealLogResponse, 0, len(logs))

	for i := range logs {
		response := buildMealLogResponse(&logs[i])
		responses = append(responses, *response)
	}

	return responses
}

func validateMealLogUpdateInput(input interfaces.MealLogUpdateInput) error {
	if input.Price < 0 {
		return errors.New("price cannot be negative")
	}

	if input.EatenAt.IsZero() {
		return errors.New("eaten time is required")
	}

	if !isSupportedMealType(input.MealType) {
		return errors.New("unsupported meal type")
	}

	return nil
}

func validateMealLogInput(input interfaces.MealLogInput) error {
	if input.Source != interfaces.MealLogSourceCustom && input.Source != interfaces.MealLogSourcePrebuilt {
		return errors.New("unsupported meal log source")
	}

	if strings.TrimSpace(input.MealID) == "" {
		return errors.New("meal id is required")
	}

	if input.Price < 0 {
		return errors.New("price cannot be negative")
	}

	if input.EatenAt.IsZero() {
		return errors.New("eaten time is required")
	}

	if !isSupportedMealType(input.MealType) {
		return errors.New("unsupported meal type")
	}

	return nil
}

func isSupportedMealType(value string) bool {
	switch strings.TrimSpace(value) {
	case "breakfast", "lunch", "dinner", "other":
		return true
	default:
		return false
	}
}
