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
  { value: 'seafood_free', label: 'Seafood-free' },
  { value: 'nut_free', label: 'Nut-free' },
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
  // Regional cuisines
  { value: 'malaysian', label: 'Malaysian' },
  { value: 'singaporean', label: 'Singaporean' },
  { value: 'indonesian', label: 'Indonesian' },
  { value: 'chinese', label: 'Chinese' },
  { value: 'indian', label: 'Indian' },
  { value: 'thai', label: 'Thai' },
  { value: 'vietnamese', label: 'Vietnamese' },
  { value: 'japanese', label: 'Japanese' },
  { value: 'korean', label: 'Korean' },
  { value: 'middle_eastern', label: 'Middle Eastern' },
  { value: 'american', label: 'American' },
  { value: 'mexican', label: 'Mexican' },
  { value: 'italian', label: 'Italian' },
  { value: 'french', label: 'French' },
  { value: 'greek', label: 'Greek' },
  { value: 'spanish', label: 'Spanish' },
  { value: 'western', label: 'Western' },

  // Meal and dish types
  { value: 'breakfast', label: 'Breakfast' },
  { value: 'rice_dishes', label: 'Rice Dishes' },
  { value: 'noodle_dishes', label: 'Noodle Dishes' },
  { value: 'soups', label: 'Soups' },
  { value: 'stews', label: 'Stews' },
  { value: 'curries', label: 'Curries' },
  { value: 'stir_fries', label: 'Stir-Fries' },
  { value: 'grilled_roasted', label: 'Grilled & Roasted' },
  { value: 'fried_foods', label: 'Fried Foods' },
  { value: 'salads', label: 'Salads' },
  { value: 'sandwiches_wraps', label: 'Sandwiches & Wraps' },
  { value: 'breads_flatbreads', label: 'Breads & Flatbreads' },
  { value: 'porridge', label: 'Porridge' },
  { value: 'dumplings', label: 'Dumplings' },
  { value: 'snacks', label: 'Snacks' },
  { value: 'desserts', label: 'Desserts' },
  { value: 'beverages', label: 'Beverages' },
  { value: 'condiments_sauces', label: 'Condiments & Sauces' },

  // Primary food groups
  { value: 'poultry', label: 'Poultry' },
  { value: 'beef', label: 'Beef' },
  { value: 'pork', label: 'Pork' },
  { value: 'lamb', label: 'Lamb' },
  { value: 'seafood', label: 'Seafood' },
  { value: 'eggs', label: 'Eggs' },
  { value: 'tofu_soy', label: 'Tofu & Soy' },
  { value: 'legumes', label: 'Legumes' },
  { value: 'vegetables', label: 'Vegetables' },
  { value: 'fruits', label: 'Fruits' },
  { value: 'grains', label: 'Grains' },
  { value: 'dairy', label: 'Dairy' },
  { value: 'nuts_seeds', label: 'Nuts & Seeds' },
]

// Disable the corresponding preferred meal types if its related dietary restriction is selected
export const restrictedMealCategories: Record<string, string[]> = {
  seafood_free: ['seafood'],
  nut_free: ['nuts_seeds'],
  vegetarian: ['poultry', 'pork', 'lamb', 'beef', 'seafood'],
  vegan: ['poultry', 'beef', 'pork', 'lamb', 'seafood', 'eggs', 'dairy'],
  halal: ['pork'],
  non_beef: ['beef']
}
