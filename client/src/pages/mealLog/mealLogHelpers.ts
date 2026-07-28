import type { MealLogItem, MealLogType } from "./mealLogTypes"

export const formatRM = (value: number) => `RM ${value.toFixed(2)}`
export const formatKcal = (value: number) => `${Math.round(value)} kcal`

export const toMonthKey = (date: Date) => {
    const year = date.getFullYear()
    const month = `${date.getMonth() + 1}`.padStart(2, '0')
    return `${year}-${month}`
}

export const toDateKey = (date: Date) => {
    const year = date.getFullYear()
    const month = `${date.getMonth() + 1}`.padStart(2, '0')
    const day = `${date.getDate()}`.padStart(2, '0')
    return `${year}-${month}-${day}`
}

export const parseDateKey = (dateKey: string) => {
    const [year, month, day] = dateKey.split('-').map(Number)
    return new Date(year, month - 1, day)
}

export const getWeekRangeForDate = (date: Date) => {
    const start = new Date(date)
    const weekdayOffset = (start.getDay() + 6) % 7
    start.setDate(start.getDate() - weekdayOffset)

    const end = new Date(start)
    end.setDate(start.getDate() + 6)

    return {
        start: toDateKey(start),
        end: toDateKey(end),
    }
}

export const shiftDateKeyByDays = (dateKey: string, days: number) => {
    const date = parseDateKey(dateKey)
    date.setDate(date.getDate() + days)
    return toDateKey(date)
}

export const parseMonthKey = (monthKey: string) => {
    const [year, month] = monthKey.split('-').map(Number)
    return new Date(year, month - 1, 1)
}

export const formatMonthTitle = (monthKey: string) => {
    const date = parseMonthKey(monthKey)

    return date.toLocaleDateString('en-US', {
        month: 'long',
        year: 'numeric',
    })
}

export const getPreviousMonthKey = (monthKey: string) => {
    const date = parseMonthKey(monthKey)
    date.setMonth(date.getMonth() - 1)
    return toMonthKey(date)
} 

export const getNextMonthKey = (monthKey: string) => {
    const date = parseMonthKey(monthKey)
    date.setMonth(date.getMonth() + 1)
    return toMonthKey(date)
}

export const toDateTimeLocalValue = (date: Date) => {
    const year = date.getFullYear()
    const month = `${date.getMonth() + 1}`.padStart(2, '0')
    const day = `${date.getDate()}`.padStart(2, '0') 
    const hour = `${date.getHours()}`.padStart(2, '0') 
    const minute = `${date.getMinutes()}`.padStart(2, '0') 

    return `${year}-${month}-${day}T${hour}:${minute}`
}

export const fromDateTimeLocalValue = (value: string) => {
    const date = new Date(value)

    return date.toISOString()
}

export const groupLogsByDay = (items: MealLogItem[]) => {
    return items.reduce<Record<string, MealLogItem[]>>((groups, item) => {
        const day = new Date(item.eatenAt).getDate().toString()

        return {
            ...groups, 
            [day]: [...(groups[day] ?? []), item],
        }
    }, {})
}

export const sumCalories = (items: MealLogItem[]) => 
    items.reduce((total, item) => total + item.calories, 0)

export const sumPrice = (items: MealLogItem[]) =>
    items.reduce((total, item) => total + item.price, 0)

export const formatMealTime = (value: string) => 
    new Date(value).toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
    })

export const inferMealTypeFromDate = (date: Date): MealLogType => {
    const hour = date.getHours()

    if (hour >= 5 && hour <= 10) {
        return 'breakfast'
    }

    if (hour >= 11 && hour <= 14) {
        return 'lunch'
    }

    if (hour >= 17 && hour <= 21) {
        return 'dinner'
    }

    return 'other'
}

export const formatMealType = (value: MealLogType) => {
    if (value === 'other') {
        return 'Other'
    }

    return value.charAt(0).toUpperCase() + value.slice(1)
}
