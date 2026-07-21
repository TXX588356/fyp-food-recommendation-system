package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(s, ",") + "}", nil
}

func (s *StringArray) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("unsupported string array value %T", value)
	}
	raw = strings.Trim(raw, "{}")
	if raw == "" {
		*s = StringArray{}
		return nil
	}
	*s = strings.Split(raw, ",")
	return nil
}

type PrebuiltMeal struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SourceCode         string
	SourceRecordID     string
	Name               string
	NormalizedName     string
	CategoryCodes      StringArray `gorm:"type:text[]"`
	ServingDescription string
	Calories           *float64
	ProteinG           *float64
	CarbsG             *float64
	FatG               *float64
	FiberG             *float64
	SugarG             *float64
	SodiumMg           *float64
	CholesterolMg      *float64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type MealCategory struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code         string
	Label        string
	DisplayOrder int
	IsActive     bool
	CreatedAt    time.Time
}
