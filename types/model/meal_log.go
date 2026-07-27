package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MealLog struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID      `gorm:"column:user_id;type:uuid,not null"`
	CustomMealItemID *uuid.UUID     `gorm:"column:custom_meal_item_id;type:uuid"`
	PrebuiltMealID   *uuid.UUID     `gorm:"column:prebuilt_meal_id;type:uuid"`
	MealName         string         `gorm:"column:meal_name;type:text;not null"`
	Price            float64        `gorm:"column:price;type:numeric(10,2);not null"`
	EatenAt          time.Time      `gorm:"column:eaten_at;type:timestamptz;not null"`
	MealType         string         `gorm:"column:meal_type;type:text;not null"`
	MealCategory     StringArray    `gorm:"column:meal_category;type:text[];not null"`
	Calories         float64        `gorm:"column:calories;type:numeric(10,2);not null"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}
