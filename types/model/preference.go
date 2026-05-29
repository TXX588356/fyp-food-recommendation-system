package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserPreference struct {
	UserID             uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	MainGoal           string    `gorm:"column:main_goal;type:text;not null"`
	MonthlyMealBudget  float64   `gorm:"column:monthly_meal_budget;type:numeric(10,2);not null"`
	DataSharingConsent *bool     `gorm:"column:data_sharing_consent;type:boolean"`
	HomeLocation       string    `gorm:"column:home_location;type:text;not null"`
	WorkSchoolLocation string    `gorm:"column:work_school_location;type:text;not null"`

	// Load all related rows from these 3 tables when UserPreference is loaded
	HealthConcerns      []UserHealthConcern      `gorm:"foreignKey:UserID;references:UserID"`
	DietaryRestrictions []UserDietaryRestriction `gorm:"foreignKey:UserID;references:UserID"`
	MealPreferences     []UserMealPreference     `gorm:"foreignKey:UserID;references:UserID"`

	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type UserHealthConcern struct {
	UserID    uuid.UUID      `gorm:"column:user_id;type:uuid;primaryKey"`
	Concern   string         `gorm:"column:concern;type:text;primaryKey"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type UserDietaryRestriction struct {
	UserID      uuid.UUID      `gorm:"column:user_id;type:uuid;primaryKey"`
	Restriction string         `gorm:"column:restriction;type:text;primaryKey"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type UserMealPreference struct {
	UserID        uuid.UUID      `gorm:"column:user_id;type:uuid;primaryKey"`
	PreferenceTag string         `gorm:"column:preference_tag;type:text;primaryKey"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
