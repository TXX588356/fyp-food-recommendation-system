package meallogreport

import (
	"context"
	"errors"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type service struct {
	mealLogRepo       interfaces.MealLogRepository
	preferenceService interfaces.PreferenceService
	now               func() time.Time
}

func NewService(repo interfaces.MealLogRepository, prefs interfaces.PreferenceService) interfaces.MealLogReportService {
	return &service{
		mealLogRepo:       repo,
		preferenceService: prefs,
		now:               time.Now,
	}
}

func (s *service) GenerateMonth(ctx context.Context, userID uuid.UUID, month string) (*interfaces.MealLogReportResponse, error) {
	return s.Generate(ctx, userID, "month", month, "", "", "")
}

func (s *service) Generate(ctx context.Context, userID uuid.UUID, period string, month string, week string, weekStart string, weekEnd string) (*interfaces.MealLogReportResponse, error) {
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil || loc == nil {
		loc = time.FixedZone("Asia/Singapore", 8*60*60)
	}

	period = strings.ToLower(strings.TrimSpace(period))
	if period == "" {
		period = "month"
	}

	start, end, label, err := parseReportPeriodInLocation(period, month, week, weekStart, weekEnd, loc)
	if err != nil {
		return nil, err
	}

	logs, err := s.mealLogRepo.ListByUserAndRange(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	// Endpoint can disable generation when no logs exists, but service should still be safe.
	if len(logs) == 0 {
		return emptyReport(start, end, label, period), nil
	}

	prefs, _ := s.preferenceService.GetByUserID(ctx, userID)
	// If preference fail, still return non-budget report instead of failing the whole report

	summary := s.buildSummary(logs, start, end, prefs, period)
	macroSummary := buildMacroSummary(logs)
	report := &interfaces.MealLogReportResponse{
		Period: interfaces.ReportPeriod{
			Start: start,
			End:   end,
			Label: label,
			Kind:  period,
		},
		Summary:           summary,
		MacroSummary:      macroSummary,
		CategoryBreakdown: buildCategoryBreakdown(logs),
		TimePatterns:      buildTimePatterns(logs, loc),
		DailyTrends:       buildDailyTrends(logs, loc),
		TopMeals:          buildTopMeals(logs, 5),
		TopExpensiveMeals: buildTopExpensiveMeals(logs, 5),
		Insights:          buildInsights(logs, summary, macroSummary),
	}

	if len(logs) < 5 {
		msg := fmt.Sprintf("You logged %d meals for this %s. The dataset is small, so the summary may not fully capture your usual eating pattern yet.", len(logs), period)
		report.LowDataWarning = &msg
	}

	return report, nil
}

func parseReportPeriodInLocation(period string, month string, week string, weekStart string, weekEnd string, loc *time.Location) (time.Time, time.Time, string, error) {
	loc = nonNilLocation(loc)

	switch period {
	case "month":
		start, end, err := parseMonthInLocation(month, loc)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}

		return start, end, start.Format("January 2006"), nil
	case "week":
		start, end, err := parseWeekInLocation(week, weekStart, weekEnd, loc)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}

		label := fmt.Sprintf("%s - %s", start.Format("Jan 2"), end.Add(-time.Nanosecond).Format("Jan 2, 2006"))
		return start, end, label, nil
	default:
		return time.Time{}, time.Time{}, "", errors.New("period must be month or week")
	}
}

func nonNilLocation(loc *time.Location) *time.Location {
	if loc != nil {
		return loc
	}

	return time.FixedZone("Asia/Singapore", 8*60*60)
}

func classifyTimeWindow(t time.Time, loc *time.Location) string {
	loc = nonNilLocation(loc)

	hour := t.In(loc).Hour()

	switch {
	case hour >= 5 && hour <= 10:
		return "breakfast"
	case hour >= 11 && hour <= 14:
		return "lunch"
	case hour >= 17 && hour <= 21:
		return "dinner"
	default:
		return "other"
	}
}

func buildCategoryBreakdown(logs []model.MealLog) []interfaces.CategoryMetric {
	// Counter map that stores how many times each cat. appears
	counts := map[string]int{}

	for _, log := range logs {
		for _, category := range log.MealCategory {
			// Each tag deserves one count. Percentages may add up above 100%.
			counts[string(category)]++
		}
	}

	// Convert map into report response items
	result := make([]interfaces.CategoryMetric, 0, len(counts))

	// turns map entry into JSON-friendly structure
	for category, count := range counts {
		percentage := float64(count) / float64(len(logs)) * 100

		result = append(result, interfaces.CategoryMetric{
			Category:   category,
			Count:      count,
			Percentage: math.Round(percentage*100) / 100,
		})
	}

	// Sort from highest to lowest
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

func parseMonthInLocation(value string, loc *time.Location) (time.Time, time.Time, error) {
	loc = nonNilLocation(loc)
	value = strings.TrimSpace(value)

	if value == "" {
		now := time.Now().In(loc)
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0), nil
	}

	parsed, err := time.ParseInLocation("2006-01", value, loc)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("month must use YYYY-MM format")
	}

	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 1, 0), nil
}

func parseWeekInLocation(value string, weekStart string, weekEnd string, loc *time.Location) (time.Time, time.Time, error) {
	loc = nonNilLocation(loc)
	value = strings.TrimSpace(value)
	weekStart = strings.TrimSpace(weekStart)
	weekEnd = strings.TrimSpace(weekEnd)

	if weekStart != "" || weekEnd != "" {
		if weekStart == "" || weekEnd == "" {
			return time.Time{}, time.Time{}, errors.New("weekStart and weekEnd must be provided together")
		}

		start, err := time.ParseInLocation("2006-01-02", weekStart, loc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("weekStart must use YYYY-MM-DD format")
		}

		endDay, err := time.ParseInLocation("2006-01-02", weekEnd, loc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("weekEnd must use YYYY-MM-DD format")
		}

		end := endDay.AddDate(0, 0, 1)
		if !end.After(start) {
			return time.Time{}, time.Time{}, errors.New("weekEnd must be on or after weekStart")
		}

		if end.Sub(start) > 7*24*time.Hour {
			return time.Time{}, time.Time{}, errors.New("weekly report range cannot exceed 7 days")
		}

		return start, end, nil
	}

	var selected time.Time
	if value == "" {
		selected = time.Now().In(loc)
	} else {
		parsed, err := time.ParseInLocation("2006-01-02", value, loc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("week must use YYYY-MM-DD format")
		}
		selected = parsed
	}

	weekdayOffset := (int(selected.Weekday()) + 6) % 7
	start := time.Date(selected.Year(), selected.Month(), selected.Day()-weekdayOffset, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 7), nil
}

func emptyReport(start, end time.Time, label string, period string) *interfaces.MealLogReportResponse {
	return &interfaces.MealLogReportResponse{
		Period: interfaces.ReportPeriod{
			Start: start,
			End:   end,
			Label: label,
			Kind:  period,
		},
		Summary:      interfaces.ReportSummary{},
		MacroSummary: interfaces.MacroSummary{},
		// Keep arrays empty instead of nil so frontend can map safely.
		CategoryBreakdown: []interfaces.CategoryMetric{},
		TimePatterns:      []interfaces.TimeWindowMetric{},
		DailyTrends:       []interfaces.DailyTrendMetric{},
		TopMeals:          []interfaces.RepeatedMealMetric{},
		TopExpensiveMeals: []interfaces.ExpensiveMealMetric{},
		Insights:          []interfaces.ReportInsight{},
	}
}

func (s *service) buildSummary(
	logs []model.MealLog,
	start time.Time,
	end time.Time,
	prefs *interfaces.PreferenceResponse,
	period string,
) interfaces.ReportSummary {
	totalSpent := 0.0
	totalCalories := 0.0
	activeDays := map[string]bool{}

	loc := start.Location()

	for _, log := range logs {
		totalSpent += log.Price
		totalCalories += log.Calories

		// Active day is based on Asia/Singapore calendar day.
		dayKey := log.EatenAt.In(loc).Format("2006-01-02")
		activeDays[dayKey] = true
	}

	totalMeals := len(logs)

	averagePricePerMeal := 0.0
	if totalMeals > 0 {
		averagePricePerMeal = totalSpent / float64(totalMeals)
	}

	averageDailySpend := 0.0
	if len(activeDays) > 0 {
		averageDailySpend = totalSpent / float64(len(activeDays))
	}

	averageCaloriesPerMeal := 0.0
	if totalMeals > 0 {
		averageCaloriesPerMeal = totalCalories / float64(totalMeals)
	}

	summary := interfaces.ReportSummary{
		TotalMeals:             totalMeals,
		ActiveLoggingDays:      len(activeDays),
		TotalSpent:             math.Round(totalSpent*100) / 100,
		AveragePricePerMeal:    math.Round(averagePricePerMeal*100) / 100,
		AverageDailySpend:      math.Round(averageDailySpend*100) / 100,
		TotalCalories:          math.Round(totalCalories*100) / 100,
		AverageCaloriesPerMeal: math.Round(averageCaloriesPerMeal*100) / 100,
	}

	// Budget widgets are hidden when budget is not configured or month is past.
	if prefs == nil || prefs.MonthlyMealBudget <= 0 {
		return summary
	}

	now := s.now().In(loc)

	if isCurrentReportPeriod(now, start, end) {
		remaining := prefs.MonthlyMealBudget - totalSpent
		if remaining < 0 {
			remaining = 0
		}

		elapsedDays := math.Max(1, math.Ceil(now.Sub(start).Hours()/24))
		totalPeriodDays := math.Ceil(end.Sub(start).Hours() / 24)
		projected := totalSpent / elapsedDays * totalPeriodDays
		budgetLimit := prefs.MonthlyMealBudget
		budgetLabel := "monthly budget"
		if period == "week" {
			daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, start.Location()).Day()
			budgetLimit = prefs.MonthlyMealBudget / float64(daysInMonth) * totalPeriodDays
			budgetLabel = "weekly budget"
		}
		status := "on_track"
		if projected > budgetLimit {
			status = "overspending"
		}

		monthlyBudget := math.Round(prefs.MonthlyMealBudget*100) / 100
		roundedBudgetLimit := math.Round(budgetLimit*100) / 100
		remaining = math.Round(remaining*100) / 100
		projected = math.Round(projected*100) / 100

		summary.MonthlyMealBudget = &monthlyBudget
		summary.BudgetLimit = &roundedBudgetLimit
		summary.BudgetLabel = &budgetLabel
		summary.RemainingUsableBudget = &remaining
		summary.ProjectedMonthSpend = &projected
		summary.ProjectedPeriodSpend = &projected
		summary.BudgetSpendStatus = &status
	}

	return summary
}

func isCurrentReportPeriod(now time.Time, start time.Time, end time.Time) bool {
	now = now.In(start.Location())
	return !now.Before(start) && now.Before(end)
}

func buildMacroSummary(logs []model.MealLog) interfaces.MacroSummary {
	totalProtein := 0.0
	totalCarbs := 0.0
	totalFat := 0.0

	for _, log := range logs {
		totalProtein += log.ProteinG
		totalCarbs += log.CarbsG
		totalFat += log.FatG
	}

	mealCount := float64(len(logs))
	if mealCount == 0 {
		return interfaces.MacroSummary{}
	}

	proteinCalories := totalProtein * 4
	carbsCalories := totalCarbs * 4
	fatCalories := totalFat * 9
	macroCalories := proteinCalories + carbsCalories + fatCalories

	summary := interfaces.MacroSummary{
		TotalProteinG:   math.Round(totalProtein*100) / 100,
		TotalCarbsG:     math.Round(totalCarbs*100) / 100,
		TotalFatG:       math.Round(totalFat*100) / 100,
		AverageProteinG: math.Round((totalProtein/mealCount)*100) / 100,
		AverageCarbsG:   math.Round((totalCarbs/mealCount)*100) / 100,
		AverageFatG:     math.Round((totalFat/mealCount)*100) / 100,
	}

	if macroCalories > 0 {
		summary.ProteinCalorieShare = math.Round((proteinCalories/macroCalories*100)*100) / 100
		summary.CarbsCalorieShare = math.Round((carbsCalories/macroCalories*100)*100) / 100
		summary.FatCalorieShare = math.Round((fatCalories/macroCalories*100)*100) / 100
	}

	return summary
}

func buildTimePatterns(logs []model.MealLog, loc *time.Location) []interfaces.TimeWindowMetric {
	timeSums := map[string]int{
		"breakfast": 0,
		"lunch":     0,
		"dinner":    0,
		"other":     0,
	}
	counts := map[string]int{
		"breakfast": 0,
		"lunch":     0,
		"dinner":    0,
		"other":     0,
	}

	for _, log := range logs {
		eatenAt := log.EatenAt.In(loc)
		window := strings.TrimSpace(log.MealType)
		if _, ok := counts[window]; !ok {
			window = classifyTimeWindow(eatenAt, loc)
		}
		minutes := eatenAt.Hour()*60 + eatenAt.Minute()

		timeSums[window] += minutes
		counts[window]++
	}

	result := make([]interfaces.TimeWindowMetric, 0, len(counts))
	for _, window := range []string{"breakfast", "lunch", "dinner", "other"} {
		count := counts[window]

		usualMinutes := 0
		usualTime := ""
		if count > 0 {
			usualMinutes = timeSums[window] / count
			usualTime = fmt.Sprintf("%02d:%02d", usualMinutes/60, usualMinutes%60)
		}

		result = append(result, interfaces.TimeWindowMetric{
			Window:           window,
			Count:            count,
			UsualTime:        usualTime,
			UsualTimeMinutes: usualMinutes,
		})
	}

	return result
}

func buildDailyTrends(logs []model.MealLog, loc *time.Location) []interfaces.DailyTrendMetric {
	byDate := map[string]interfaces.DailyTrendMetric{}

	for _, log := range logs {
		eatenAt := log.EatenAt.In(loc)
		dateKey := eatenAt.Format("2006-01-02")
		item := byDate[dateKey]
		item.Date = dateKey
		item.DayLabel = eatenAt.Format("Jan 2")
		item.MealCount++
		item.TotalSpent = math.Round((item.TotalSpent+log.Price)*100) / 100
		item.TotalCalories = math.Round((item.TotalCalories+log.Calories)*100) / 100
		byDate[dateKey] = item
	}

	result := make([]interfaces.DailyTrendMetric, 0, len(byDate))
	for _, item := range byDate {
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})

	return result
}

func buildTopMeals(logs []model.MealLog, limit int) []interfaces.RepeatedMealMetric {
	counts := map[string]int{}

	for _, log := range logs {
		name := strings.TrimSpace(log.MealName)
		if name == "" {
			continue
		}

		counts[name]++
	}

	result := make([]interfaces.RepeatedMealMetric, 0, len(counts))
	for name, count := range counts {
		if count < 2 {
			continue // Only repeated meals are useful here.
		}

		result = append(result, interfaces.RepeatedMealMetric{
			MealName: name,
			Count:    count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].MealName < result[j].MealName
		}

		return result[i].Count > result[j].Count
	})

	if len(result) > limit {
		return result[:limit]
	}

	return result
}

func buildTopExpensiveMeals(logs []model.MealLog, limit int) []interfaces.ExpensiveMealMetric {
	result := make([]interfaces.ExpensiveMealMetric, 0, len(logs))

	for _, log := range logs {
		// Keep individual log entries because the same meal can have different prices.
		result = append(result, interfaces.ExpensiveMealMetric{
			MealName: log.MealName,
			Price:    math.Round(log.Price*100) / 100,
			EatenAt:  log.EatenAt,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Price > result[j].Price
	})

	if len(result) > limit {
		return result[:limit]
	}

	return result
}

func buildInsights(logs []model.MealLog, summary interfaces.ReportSummary, macroSummary interfaces.MacroSummary) []interfaces.ReportInsight {
	insights := []interfaces.ReportInsight{}

	if summary.TotalMeals < 5 {
		insights = append(insights, interfaces.ReportInsight{
			Type:           "low_data",
			Severity:       "note",
			Title:          "Not enough logs for a strong pattern yet",
			Evidence:       fmt.Sprintf("You logged %d meals this month.", summary.TotalMeals),
			Recommendation: "Log more meals before treating this report as your usual eating pattern.",
		})
	}

	categoryCounts := countCategories(logs)

	if summary.TotalMeals >= 5 && macroSummary.AverageProteinG > 0 && macroSummary.AverageProteinG < 15 {
		insights = append(insights, interfaces.ReportInsight{
			Type:           "macro_balance",
			Severity:       "caution",
			Title:          "Protein looks low per meal",
			Evidence:       fmt.Sprintf("Average protein is %.1fg per logged meal.", macroSummary.AverageProteinG),
			Recommendation: "Consider meals with clearer protein sources such as eggs, chicken, fish, tofu, beans, or lean meat.",
		})
	}

	if count := categoryCounts["fried_foods"]; count > 0 {
		percentage := float64(count) / float64(len(logs)) * 100

		if percentage >= 30 {
			insights = append(insights, interfaces.ReportInsight{
				Type:           "balance",
				Severity:       "caution",
				Title:          "Fried food appears often",
				Evidence:       fmt.Sprintf("Fried food appears in %.0f%% of logged meals.", percentage),
				Recommendation: "Consider replacing some fried meals with grilled, soup, or vegetable-forward meals.",
			})
		}
	}

	plantForwardCount := categoryCounts["vegetables"] + categoryCounts["fruits"] + categoryCounts["salads"]
	if plantForwardCount > 0 {
		percentage := float64(plantForwardCount) / float64(len(logs)) * 100

		if percentage >= 25 {
			insights = append(insights, interfaces.ReportInsight{
				Type:           "balance",
				Severity:       "positive",
				Title:          "Plant-forward meals are showing up",
				Evidence:       fmt.Sprintf("Plant-forward categories appear in %.0f%% of logged meals.", percentage),
				Recommendation: "Keep this pattern consistent across the month.",
			})
		}
	}

	if summary.ProjectedMonthSpend != nil {
		severity := "note"
		title := "Current spending pace is projected"
		recommendation := "Compare this with your monthly budget before choosing higher-cost meals."
		projectedLabel := "monthly spend"
		if summary.BudgetLabel != nil && *summary.BudgetLabel == "weekly budget" {
			projectedLabel = "weekly spend"
			recommendation = "Compare this with your weekly budget before choosing higher-cost meals."
		}

		if summary.BudgetSpendStatus != nil && *summary.BudgetSpendStatus == "overspending" {
			severity = "warning"
			title = "Current pace is overspending"
			recommendation = "Choose lower-cost meals for the rest of the month to get closer to your budget."
			if projectedLabel == "weekly spend" {
				recommendation = "Choose lower-cost meals for the rest of the week to get closer to your budget."
			}
		} else if summary.BudgetSpendStatus != nil && *summary.BudgetSpendStatus == "on_track" {
			severity = "positive"
			title = "Current pace is on track"
			recommendation = "Keep roughly the same daily spending pace to stay within your budget."
		}

		insights = append(insights, interfaces.ReportInsight{
			Type:           "budget_projection",
			Severity:       severity,
			Title:          title,
			Evidence:       fmt.Sprintf("At the current pace, projected %s is RM%.2f.", projectedLabel, *summary.ProjectedMonthSpend),
			Recommendation: recommendation,
		})
	}

	if len(insights) > 5 {
		return insights[:5]
	}

	return insights
}

func countCategories(logs []model.MealLog) map[string]int {
	counts := map[string]int{}

	for _, log := range logs {
		for _, category := range log.MealCategory {
			counts[string(category)]++
		}
	}

	return counts
}
