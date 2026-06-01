export type LocationValue = {
  state: string
  district: string
}

export type PreferencesFormValues = {
    mainGoal: string
    healthConcerns: string[]
    dietaryRestrictions: string[]
    preferredMealCategories: string[]
    monthlyMealBudget: number | string
    workSchoolLocation: LocationValue
    homeLocation: LocationValue 
}

export type PreferenceData = {
    mainGoal: string
    monthlyMealBudget: number
    dataSharingConsent: boolean | null
    homeLocation: string
    workSchoolLocation: string
    healthConcerns: string[]
    dietaryRestrictions: string[]
    preferredMealTags: string[]
}

export type SettingKey = 
| 'dietary'
| 'meals'
| 'budgetLocation'
| 'health'
| 'goals'
| 'consent'


