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