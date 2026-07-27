export type MealLogSource = 'custom' | 'prebuilt'
export type MealLogType = 'breakfast' | 'lunch' | 'dinner' | 'other'

export type MealLogInput = {
    source: MealLogSource
    mealId: string
    price: number
    eatenAt: string
    mealType: MealLogType
}

export type MealLogUpdateInput = {
    price: number
    eatenAt: string
    mealType: MealLogType
}

export type MealLogItem = {
    id: string
    customMealItemId?: string
    prebuiltMealId?: string
    eatenAt: string
    mealType: MealLogType
    mealName: string
    calories: number
    proteinG: number
    carbsG: number
    fatG: number
    price: number
    mealCategory: string[]
}

export type MealLogSummary = {
    month: string
    totalMealsEaten: number
    totalSpent: number
    totalCalories: number
    budgetRemaining?: number
    showBudgetRemaining: boolean
}

export type MealLogMonthResponse = {
    summary: MealLogSummary
    items: MealLogItem[]
}

export type LoggableMeal = {
    source: MealLogSource
    mealId: string
    name: string
    calories: number
}

export type ReportPeriod = {
    start: string
    end: string
    label: string
}

export type ReportSummary = {
    totalMeals: number
    activeLoggingDays: number
    totalSpent: number
    averagePricePerMeal: number
    averageDailySpend: number
    remainingUsableBudget?: number
    projectedMonthSpend?: number
    totalCalories: number
    averageCaloriesPerMeal: number
}

export type MacroSummary = {
    totalProteinG: number
    totalCarbsG: number
    totalFatG: number
    averageProteinG: number
    averageCarbsG: number
    averageFatG: number
    proteinCalorieShare: number
    carbsCalorieShare: number
    fatCalorieShare: number
}

export type CategoryMetric = {
    category: string
    count: number
    percentage: number
}

export type TimeWindowMetric = {
    window: MealLogType
    count: number
    usualTime: string
    usualTimeMinutes: number
}

export type DailyTrendMetric = {
    date: string
    dayLabel: string
    mealCount: number
    totalSpent: number
    totalCalories: number
}

export type RepeatedMealMetric = {
    mealName: string
    count: number
}

export type ExpensiveMealMetric = {
    mealName: string
    price: number
    eatenAt: string
}

export type ReportInsight = {
    type: string
    severity: 'positive' | 'note' | 'caution' | 'warning'
    title: string
    evidence: string
    recommendation: string
}

export type MealLogReportResponse = {
    period: ReportPeriod
    summary: ReportSummary
    macroSummary: MacroSummary
    categoryBreakdown: CategoryMetric[]
    timePatterns: TimeWindowMetric[]
    dailyTrends: DailyTrendMetric[]
    topMeals: RepeatedMealMetric[]
    topExpensiveMeals: ExpensiveMealMetric[]
    insights: ReportInsight[]
    lowDataWarning?: string
    generatedSummary?: string
}
