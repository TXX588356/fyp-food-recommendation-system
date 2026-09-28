package model

import (
	"time"

	"github.com/google/uuid"
)

type RecommendationInteraction struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	RecommendationID uuid.UUID  `gorm:"column:recommendation_id;type:uuid;not null"`
	UserID           uuid.UUID  `gorm:"column:user_id;type:uuid;not null"`
	MealID           uuid.UUID  `gorm:"column:meal_id;type:uuid;not null"`
	MealSource       string     `gorm:"column:meal_source;type:text;not null"`
	Position         int        `gorm:"column:position;not null"`
	ShownAt          time.Time  `gorm:"column:shown_at;type:timestamptz;not null"`
	ClickedAt        *time.Time `gorm:"column:clicked_at;type:timestamptz"`
	SelectedAt       *time.Time `gorm:"column:selected_at;type:timestamptz"`
	LoggedAt         *time.Time `gorm:"column:logged_at;type:timestamptz"`
	Rating           *int       `gorm:"column:rating"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}
