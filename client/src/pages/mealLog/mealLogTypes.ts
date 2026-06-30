export type MealLogSource = 'custom' | 'prebuilt'

export type MealLogInput = {
    source: MealLogSource
    mealId: string
    price: number
    eatenAt: string
}

export type MealLogItem = {
    id: string
    customMealItemId?: string
    prebuiltMealId?: string
    eatenAt: string
    mealName: string
    calories: number
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
