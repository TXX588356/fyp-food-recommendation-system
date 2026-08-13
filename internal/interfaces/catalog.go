package interfaces

import (
	"context"
	"errors"

	"fyp/food-rs/types/model"

	"github.com/google/uuid"
)

var (
	ErrCatalogNotFound     = errors.New("catalogue resource not found")
	ErrInvalidCatalogQuery = errors.New("invalid catalogue query")
)

type CatalogQuery struct {
	Query      string
	Categories []string
	Source     string
	AfterName  string
	AfterID    *uuid.UUID
	Limit      int
}

type CatalogCategory struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	MealCount int64  `json:"meal_count"`
}

type CatalogNutrition struct {
	Calories *float64 `json:"calories"`
	ProteinG *float64 `json:"protein_g"`
	CarbsG   *float64 `json:"carbs_g"`
	FatG     *float64 `json:"fat_g"`
}

type CatalogPortion struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type CatalogImage struct {
	ID          uuid.UUID `json:"id"`
	URL         string    `json:"url"`
	Attribution string    `json:"attribution,omitempty"`
	SourceURL   string    `json:"source_url,omitempty"`
	License     string    `json:"license,omitempty"`
	LicenseURL  string    `json:"license_url,omitempty"`
}

type CatalogMeal struct {
	ID                uuid.UUID        `json:"id"`
	Name              string           `json:"name"`
	SourceName        string           `json:"source_name,omitempty"`
	SourceCode        string           `json:"source_code"`
	SourceRecordID    string           `json:"source_record_id"`
	Categories        []string         `json:"categories"`
	SelectedNutrition CatalogNutrition `json:"nutrition"`
	SelectedPortion   CatalogPortion   `json:"portion"`
	Image             *CatalogImage    `json:"image,omitempty"`
}

type GeneratedCatalogMealInput struct {
	Name               string
	CategoryCodes      []string
	ServingDescription string
	Calories           float64
	ProteinG           float64
	CarbsG             float64
	FatG               float64
}

type CatalogMealPage struct {
	Items      []CatalogMeal `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type CatalogRepository interface {
	SearchMeals(ctx context.Context, query CatalogQuery) ([]model.PrebuiltMeal, error)
	FindMeal(ctx context.Context, id uuid.UUID) (*model.PrebuiltMeal, error)
	ListCategories(ctx context.Context) ([]CatalogCategory, error)
	CategoryCodesExist(ctx context.Context, codes []string) (bool, error)
	CreateGeneratedMeal(ctx context.Context, input GeneratedCatalogMealInput) (*model.PrebuiltMeal, error)
}

type CatalogService interface {
	SearchMeals(ctx context.Context, query CatalogQuery) (CatalogMealPage, error)
	GetMeal(ctx context.Context, id uuid.UUID) (CatalogMeal, error)
	ListCategories(ctx context.Context) ([]CatalogCategory, error)
	CreateGeneratedMeal(ctx context.Context, input GeneratedCatalogMealInput) (CatalogMeal, error)
}

type ObjectURLResolver interface {
	Resolve(objectKey string) string
}
