export const mainGoalOptions = [
	{ value: 'eat_healthier', label: 'Eat healthier' },
  { value: 'muscle_gain', label: 'Gain muscles / high protein' },
  { value: 'quick_recommendation', label: 'Just get quick meal suggestions' },
]

export const dietaryRestrictionOptions = [
  { value: 'none', label: 'No restrictions' },
  { value: 'non_beef', label: 'Non-beef' },
  { value: 'halal', label: 'Halal' },
  { value: 'vegetarian', label: 'Vegetarian' },
  { value: 'vegan', label: 'Vegan' },
  { value: 'seafood_free', label: 'Seafood allergy' },
  { value: 'nut_free', label: 'Nut allergy' },
  { value: 'low_sugar', label: 'Low sugar' },
  { value: 'low_salt', label: 'Low salt' },
  { value: 'low_fat', label: 'Low fat' },
]

export const healthConcernOptions = [
  { value: 'none', label: 'No' },
  { value: 'gout', label: 'Gout' },
  { value: 'diabetes', label: 'Diabetes' },
  { value: 'high_blood_pressure', label: 'High blood pressure' },
]

export type CityRow = {
  name: string
  country: string
  subcountry: string
  geonameid: string
}

export function parseMalaysiaCitiesCsv(csv: string): CityRow[] {
  const [, ...rows] = csv.trim().split('\n')

  return rows.map((row) => {
    const [name, country, subcountry, geonameid] = row.split(',')

    return {
      name: name.trim(),
      country: country.trim(),
      subcountry: subcountry.trim(),
      geonameid: geonameid.trim(),
    }
  })
}

export const mealCategoryOptions = [
  { value: "american", label: "American" },
  { value: "basics", label: "Basics" },
  { value: "breakfast", label: "Breakfast" },
  { value: "chinese", label: "Chinese" },
  { value: "condiments", label: "Condiments" },
  { value: "desserts", label: "Desserts" },
  { value: "drinks", label: "Drinks" },
  { value: "french", label: "French" },
  { value: "fruits", label: "Fruits" },
  { value: "grains", label: "Grains" },
  { value: "greek", label: "Greek" },
  { value: "healthy", label: "Healthy" },
  { value: "indian", label: "Indian" },
  { value: "italian", label: "Italian" },
  { value: "japanese", label: "Japanese" },
  { value: "korean", label: "Korean" },
  { value: "kuih", label: "Kuih" },
  { value: "legumes", label: "Legumes" },
  { value: "meat", label: "Meat" },
  { value: "mexican", label: "Mexican" },
  { value: "middle_eastern", label: "Middle Eastern" },
  { value: "noodles", label: "Noodles" },
  { value: "nuts", label: "Nuts" },
  { value: "proteins", label: "Proteins" },
  { value: "rice", label: "Rice" },
  { value: "roti", label: "Roti" },
  { value: "seafood", label: "Seafood" },
  { value: "seeds", label: "Seeds" },
  { value: "snacks", label: "Snacks" },
  { value: "soups", label: "Soups" },
  { value: "spanish", label: "Spanish" },
  { value: "thai", label: "Thai" },
  { value: "vegetables", label: "Vegetables" },
  { value: "vietnamese", label: "Vietnamese" },
  { value: "western", label: "Western" }
]

// Disable the corresponding preferred meal types if its related dietary restriction is selected
export const restrictedMealCategories: Record<string, string[]> = {
  seafood_free: ['seafood'],
  nut_free: ['nuts', 'seeds'],
  vegetarian: ['meat', 'seafood'],
  vegan: ['meat', 'seafood', 'dairy'],
}
