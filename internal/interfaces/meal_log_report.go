package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MealLogReportService interface {
	GenerateMonth(ctx context.Context, userID uuid.UUID, month string) (*MealLogReportResponse, error)
}

type MealLogReportResponse struct {
	Period            ReportPeriod          `json:"period"`
	Summary           ReportSummary         `json:"summary"`
	MacroSummary      MacroSummary          `json:"macroSummary"`
	CategoryBreakdown []CategoryMetric      `json:"categoryBreakdown"`
	TimePatterns      []TimeWindowMetric    `json:"timePatterns"`
	DailyTrends       []DailyTrendMetric    `json:"dailyTrends"`
	TopMeals          []RepeatedMealMetric  `json:"topMeals"`
	TopExpensiveMeals []ExpensiveMealMetric `json:"topExpensiveMeals"`
	Insights          []ReportInsight       `json:"insights"`
	LowDataWarning    *string               `json:"lowDataWarning,omitempty"`
}

type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Label string    `json:"label"`
}

type ReportSummary struct {
	TotalMeals             int      `json:"totalMeals"`
	ActiveLoggingDays      int      `json:"activeLoggingDays"`
	TotalSpent             float64  `json:"totalSpent"`
	AveragePricePerMeal    float64  `json:"averagePricePerMeal"`
	AverageDailySpend      float64  `json:"averageDailySpend"`
	RemainingUsableBudget  *float64 `json:"remainingUsableBudget,omitempty"`
	ProjectedMonthSpend    *float64 `json:"projectedMonthSpend,omitempty"`
	TotalCalories          float64  `json:"totalCalories"`
	AverageCaloriesPerMeal float64  `json:"averageCaloriesPerMeal"`
}

type MacroSummary struct {
	TotalProteinG       float64 `json:"totalProteinG"`
	TotalCarbsG         float64 `json:"totalCarbsG"`
	TotalFatG           float64 `json:"totalFatG"`
	AverageProteinG     float64 `json:"averageProteinG"`
	AverageCarbsG       float64 `json:"averageCarbsG"`
	AverageFatG         float64 `json:"averageFatG"`
	ProteinCalorieShare float64 `json:"proteinCalorieShare"`
	CarbsCalorieShare   float64 `json:"carbsCalorieShare"`
	FatCalorieShare     float64 `json:"fatCalorieShare"`
}

type CategoryMetric struct {
	Category   string  `json:"category"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"` // Can exceed 100% total because tags overlap
}

type TimeWindowMetric struct {
	Window           string `json:"window"` // breakfast, lunch, dinner, other
	Count            int    `json:"count"`  // Number of logs behind this estimate
	UsualTime        string `json:"usualTime"`
	UsualTimeMinutes int    `json:"usualTimeMinutes"`
}

type DailyTrendMetric struct {
	Date          string  `json:"date"`
	DayLabel      string  `json:"dayLabel"`
	MealCount     int     `json:"mealCount"`
	TotalSpent    float64 `json:"totalSpent"`
	TotalCalories float64 `json:"totalCalories"`
}

type RepeatedMealMetric struct {
	MealName string `json:"mealName"`
	Count    int    `json:"count"`
}

type ExpensiveMealMetric struct {
	MealName string    `json:"mealName"`
	Price    float64   `json:"price"`
	EatenAt  time.Time `json:"eatenAt"`
}

type ReportInsight struct {
	Type           string `json:"type"`
	Severity       string `json:"severity"` // positive, note, caution, warning
	Title          string `json:"title"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
}
