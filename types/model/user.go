package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                     uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name                   string         `gorm:"type:text;not null"`
	Email                  string         `gorm:"type:text;not null;uniqueIndex"`
	PasswordHash           string         `gorm:"column:password_hash;type:text;not null"`
	CreatedAt              time.Time      `gorm:"not null"`
	UpdatedAt              time.Time      `gorm:"not null"`
	DeletedAt              gorm.DeletedAt `gorm:"index"`
	HasCompletedOnboarding bool           `gorm:"column:has_completed_onboarding;not null;default:false"`
}
