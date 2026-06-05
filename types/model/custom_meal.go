package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomMealItem struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string    `gorm:"type:text;not null"`
	Price          float64   `gorm:"type:numeric(10,2);not null"`
	Calories       float64   `gorm:"type:numeric(10,2);not null"`
	FatG           float64   `gorm:"type:numeric(10,2);not null"`
	ProteinG       float64   `gorm:"type:numeric(10,2);not null"`
	CarbsG         float64   `gorm:"type:numeric(10,2);not null"`
	State          string    `gorm:"type:text"`
	District       string    `gorm:"type:text"`
	RestaurantName string    `gorm:"type:text"`
	CreatedBy      uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	DietaryRestrictionTags []CustomMealDietaryRestrictionTag `gorm:"foreignKey:CustomMealItemID"`
	MealCategoryTags       []CustomMealCategoryTag           `gorm:"foreignKey:CustomMealItemID"`
}

type CustomMealDietaryRestrictionTag struct {
	CustomMealItemID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	DietaryRestrictionTag string    `gorm:"type:text;primaryKey"`
}

type CustomMealCategoryTag struct {
	CustomMealItemID uuid.UUID `gorm:"type:uuid;primaryKey"`
	MealCategory     string    `gorm:"type:text;primaryKey"`
}

// TableName maps custom meals to the migration-created table name.
func (CustomMealItem) TableName() string {
	return "custom_meal_items"
}

// TableName maps dietary restriction tags to the migration-created table name.
func (CustomMealDietaryRestrictionTag) TableName() string {
	return "custom_meal_dietary_restriction_tags"
}

// TableName maps category tags to the migration-created table name.
func (CustomMealCategoryTag) TableName() string {
	return "custom_meal_categories"
}
