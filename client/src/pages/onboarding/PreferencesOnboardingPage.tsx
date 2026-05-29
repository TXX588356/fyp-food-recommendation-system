import { Alert, Box, Button, Group, NumberInput, Progress, Select, Text, Title, UnstyledButton } from "@mantine/core"
import { useForm } from "@mantine/form"
import React, { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import "@/App.css"
import axios from "axios"
import { useAuth } from "@/auth/useAuth"


const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

type CityRow = {
  name: string
  country: string
  subcountry: string
  geonameid: string
}

function parseMalaysiaCitiesCsv(csv: string): CityRow[] {
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

type LocationValue = {
  state: string
  district: string
}

type PreferencesFormValues = {
    mainGoal: string
    healthConcerns: string[]
    dietaryRestrictions: string[]
    preferredMealCategories: string[]
    monthlyMealBudget: number | string
    workSchoolLocation: LocationValue
    homeLocation: LocationValue 
}

const createEmptyLocation = (): LocationValue => ({
  state: '',
  district: '',
})

const mainGoals = [
  { value: 'eat_healthier', label: 'Eat healthier'},
  { value: 'muscle_gain', label: 'Gain muscles / high protein'},
  { value: 'quick_recommendation', label: 'Just get quick meal suggestions'},
]

const dietaryRestrictions = [
  { value: 'none', label: 'No restrictions'},
  { value: 'non_beef', label: 'Non-beef'},
  { value: 'halal', label: 'Halal'},
  { value: 'vegetarian', label: 'Vegetarian'},
  { value: 'vegan', label: 'Vegan'},
  { value: 'seafood_free', label: 'Seafood allergy'},
  { value: 'nut_free', label: 'Nut allergy'},
  { value: 'low_sugar', label: 'Low sugar'},
  { value: 'low_salt', label: 'Low salt'},
  { value: 'low_fat', label: 'Low fat'},
]

const healthConcerns = [
  { value: 'none', label: 'No'},
  { value: 'gout', label: 'Gout'},
  { value: 'diabetes', label: 'Diabetes'},
  { value: 'high_blood_pressure', label: 'High blood pressure'},
]

const categories = [
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
const restrictedMealCategories: Record<string, string[]> = {
  seafood_free: ['seafood'],
  nut_free: ['nuts', 'seeds'],
  non_beef: ['meat'],
  vegetarian: ['meat', 'seafood'],
  vegan: ['meat', 'seafood', 'dairy'],
}

const stepLabels = [
  'Goal',
  'Diet',
  'Health',
  'Budget & Location',
  'Meals',
]

const stepDescriptions = [
  'Choose the outcome that should shape your first recommendations.',
  'Tell us what should never show up in your meal suggestions.',
  'Add any health signals that should influence ranking and filtering.',
  'Set the budget and places that make recommendations practical.',
  'Pick the categories you want the system to favor more often.',
]

function LocationSelectGroup({
  label, 
  description, 
  value, 
  onChange,
}: {
  label: string
  description: string
  value: LocationValue
  onChange: (value: LocationValue) => void
}) {
  const [cities, setCities] = useState<CityRow[]>([])
  const [state, setState] = useState<{ value: string, label: string }[]>([])
  const [districts, setDistricts] = useState<{ value: string; label: string }[]>([])

  useEffect(() => {
    fetch('/malaysia_cities.csv')
      .then(response => response.text())
      .then((csv) => {
        const rows = parseMalaysiaCitiesCsv(csv)
        setCities(rows)

        const nextStates = Array.from(
          new Set(rows.map((row) => row.subcountry))
        )
        .sort()
        .map((state) => ({
          value: state,
          label: state,
        }))

        setState(nextStates)
      })
      .catch((error) => {
        console.error('Error loading Malaysia cities CSV: ', error)
        setState([])
        setCities([])
      })
  }, [])

  useEffect(() => {
    if (!value.state) {
      setDistricts([])
      return
    }

    const nextDistricts = cities
      .filter((city) => city.subcountry === value.state)
      .map((city) => city.name)
      .sort()
      .map((city) => ({
        value: city,
        label: city,
      }))

    setDistricts(nextDistricts)
    
  }, [value.state, cities])

  return (
    <Box className="preference-location-group">
      <Text className="preference-location-title">{label}</Text>
      <Text className="preference-location-copy">{description}</Text>

      <Group grow align="flex-start">
        <Select 
          label="State"
          placeholder="Select state"
          data={state}
          value={value.state || null}
          allowDeselect={false}
          onChange={(nextState) => {
            if (!nextState) return

            if (nextState === value.state) {
              return
            }

            onChange({
              state: nextState, 
              district: '',
            })

          }}
          classNames={{
            label: 'preference-input-label',
            input: 'preference-input',
          }}
        />
        <Select 
          label="District"
          placeholder="Select district"
          data={districts}
          value={value.district || null}
          allowDeselect={false}
          disabled={!value.state || districts.length === 0}
          onChange={(district) => 
            onChange({
              ...value,
              district: district ?? '',
            })
          }
          classNames={{
            label: 'preference-input-label',
            input: 'preference-input',
          }}
        />
      </Group>


    </Box>
  )
}

export default function PreferencesOnboardingPage() {
  const navigate = useNavigate()
  const { user, updateUser } = useAuth()
  const [step, setStep] = useState(1)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  const form = useForm<PreferencesFormValues>({
    initialValues: {
      mainGoal: '',
      dietaryRestrictions: [],
      healthConcerns: [],
      preferredMealCategories: [],
      monthlyMealBudget: '',
      workSchoolLocation: createEmptyLocation(),
      homeLocation: createEmptyLocation(),
    }
  })

  const progress = (step / 5) * 100
  const currentStepIndex = step - 1
  const disabledMealCategories = useMemo(
    () => new Set(
      form.values.dietaryRestrictions.flatMap(
        (restriction) => restrictedMealCategories[restriction] ?? [],
      ),
    ),
    [form.values.dietaryRestrictions],
  )

  useEffect(() => {
    const nextMealCategories = form.values.preferredMealCategories.filter(
      (category) => !disabledMealCategories.has(category),
    )

    if (nextMealCategories.length !== form.values.preferredMealCategories.length) {
      form.setFieldValue('preferredMealCategories', nextMealCategories)
    }
  }, [form.values.dietaryRestrictions, form.values.preferredMealCategories, disabledMealCategories])

  const toggleArrayValue = (
    field: 'dietaryRestrictions' | 'healthConcerns' | 'preferredMealCategories',
    value: string,
  ) => {
    // get current selected items
    const currentValues = form.values[field]

    if (field === 'dietaryRestrictions' && value === 'none') {
      form.setFieldValue(field, currentValues.includes(value) ? [] : [value])
      return
    }

    if (field === 'healthConcerns' && value === 'none') {
      form.setFieldValue(field, currentValues.includes(value) ? [] : [value])
      return
    }

    // check if value already exists in checked values list; if yes, remove it (uncheck)
    // if no, add the new value, except for none cases
    const nextValues = currentValues.includes(value) // check whether the clicked option is already selected
      ? currentValues.filter((item) => item != value) // if yes, remove it from the array
      : [...currentValues.filter((item) => item !== 'none'), value] // if no, remove special 'none' case, then append the new selection

    form.setFieldValue(field, nextValues)
  }

  const canContinue = () => {
    if (step === 1) return Boolean(form.values.mainGoal)
    if (step === 2) return form.values.dietaryRestrictions.length > 0
    if (step === 3) return form.values.healthConcerns.length > 0
    if (step === 4) {
      return (
        Number(form.values.monthlyMealBudget) > 0 &&
        form.values.workSchoolLocation.state != '' && form.values.workSchoolLocation.district != '' &&
        form.values.homeLocation.state != '' && form.values.homeLocation.district != ''
      )
    }
    if (step === 5) return form.values.preferredMealCategories.length > 0

    return false
  }

  const handleNext = () => {
    if(!canContinue()) return
    setStep((currentStep) => Math.min(currentStep + 1, 5))
  }

  const handleBack = () => {
    setStep((currentStep) => Math.max(currentStep - 1, 1))
  }

  const handleSubmit = async () => {
    if (!canContinue()) return

    setSubmitError(null)
    setIsSubmitting(true)

    const token = localStorage.getItem('token')

    const formatLocation = (location: LocationValue) =>
      `${location.district}, ${location.state}`

    const payload = {
      mainGoal: form.values.mainGoal,
      dietaryRestrictions: form.values.dietaryRestrictions.filter(
        (item) => item !== 'none',
      ),
      healthConcerns: form.values.healthConcerns.filter(
        (item) => item !== 'none',
      ),
      preferredMealTags: form.values.preferredMealCategories,
      monthlyMealBudget: Number(form.values.monthlyMealBudget),
      workSchoolLocation: formatLocation(form.values.workSchoolLocation),
      homeLocation: formatLocation(form.values.homeLocation),
    }

    try {
      await axios.post(`${API_BASE_URL}/preferences`, payload, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      updateUser({
        ...user,
        hasCompletedOnboarding: true,
      })
      navigate('/recommendation', { replace: true })
    } catch (error) {
      console.error('Preference onboarding failed', error)
      setSubmitError('Could not save your prefereces. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Box className="preference-page">
      <Box component="main" className="preference-shell">
        <Box component="section" className="preference-panel">
          <Box className="preference-header">
            <Box className="preference-topline">
              <Text>Step {step} of 5</Text>
            </Box>

            <Title order={1}>Let us get to know you more.</Title>
            <Text className="preference-intro">{stepDescriptions[currentStepIndex]}</Text>

            <Box className="preference-progress">
              <Progress value={progress} color="#0f6b43" size="sm" radius="xl" />
              <Box className="preference-step-list">
                {stepLabels.map((label, index) => (
                  <Box
                    key={label}
                    className={[
                      'preference-step-pill',
                      index + 1 === step ? 'preference-step-pill-active' : '',
                      index + 1 < step ? 'preference-step-pill-complete' : '',
                    ].join(' ')}
                  >
                    <span>{index + 1 < step ? '✓' : index + 1}</span>
                    {label}
                  </Box>
                ))}
              </Box>
            </Box>
          </Box>

          {submitError && (
            <Alert color="red" mb="lg">{submitError}</Alert>
          )}

          {step === 1 && (
            <QuestionBlock title="What is your main goal?">
              <Box className="preference-stack">
                {mainGoals.map((goal) => (
                  <OptionButton 
                    key={goal.value}
                    label={goal.label}
                    selected={form.values.mainGoal === goal.value}
                    onClick={() => form.setFieldValue('mainGoal', goal.value)}
                  />
                ))}
              </Box>
            </QuestionBlock>
          )}

          {step === 2 && (
            <QuestionBlock title="Do you have any dietary restrictions?">
              <Box className="preference-chip-grid">
                {dietaryRestrictions.map((restriction) => (
                  <OptionButton 
                    key={restriction.value}
                    label={restriction.label}
                    selected={form.values.dietaryRestrictions.includes(restriction.value)}
                    onClick={() => toggleArrayValue('dietaryRestrictions', restriction.value)}
                  />
                ))}
              </Box>
            </QuestionBlock>
          )}

          {step === 3 && (
            <QuestionBlock title="Do you want recommendations based on any condition or health concern?">
              <Box className="preference-stack preference-stack-compact">
                {healthConcerns.map((concern) => (
                  <OptionButton 
                    key={concern.value}
                    label={concern.label}
                    selected={form.values.healthConcerns.includes(concern.value)}
                    onClick={() => toggleArrayValue('healthConcerns', concern.value)}
                    wide
                  />
                ))}
              </Box>
            </QuestionBlock>
          )}

          {step === 4 && (
            <QuestionBlock title="Tell Us Your Budget and Location">
              <Box className="preference-form-grid">
                <NumberInput 
                  label="Monthly meal budget"
                  prefix="RM "
                  min={1}
                  hideControls
                  value={form.values.monthlyMealBudget}
                  onChange={(value) => form.setFieldValue('monthlyMealBudget', value)}
                  classNames={{
                    label: 'preference-input-label',
                    input: 'preference-input'
                  }}
                />

                <LocationSelectGroup
                  label="Workplace / school location"
                  description="Where do you usually need lunch or dinner recommendations?"
                  value={form.values.workSchoolLocation}
                  onChange={(value) => form.setFieldValue('workSchoolLocation', value)}
                />

                <LocationSelectGroup
                  label="Home location"
                  description="Where should evening and weekend recommendations be centered?"
                  value={form.values.homeLocation}
                  onChange={(value) => form.setFieldValue('homeLocation', value)}
                />
              </Box>
            </QuestionBlock>
          )}

          {step === 5 && (
            <QuestionBlock title="Which meal types do you want to see more often?">
              <Box className="preference-chip-grid">
                {categories.map((category) => (
                  <OptionButton 
                    key={category.value}
                    label={category.label}
                    selected={form.values.preferredMealCategories.includes(category.value)}
                    disabled={disabledMealCategories.has(category.value)}
                    onClick={() => {toggleArrayValue('preferredMealCategories', category.value)}}
                  />
                ))}
              </Box>
            </QuestionBlock>
          )}

          <Group justify="space-between" className="preference-actions">
            <Button
              variant="subtle"
              className="preference-back-button"
              leftSection={<span aria-hidden="true">←</span>}
              onClick={handleBack}
              disabled={step === 1 || isSubmitting}
            >
              Back
            </Button>

            {step < 5 ? (
              <Button
                className="preference-next-button"
                rightSection={<span aria-hidden="true">→</span>}
                onClick={handleNext}
                disabled={!canContinue()}
              >
                Next
              </Button>
            ) : (
              <Button
                className="preference-next-button"
                rightSection={<span aria-hidden="true">→</span>}
                onClick={handleSubmit}
                loading={isSubmitting}
                disabled={!canContinue() || isSubmitting}
              >
                Get Started
              </Button>
            )}
          </Group>
        </Box>
      </Box>
    </Box>
  )
}

function QuestionBlock({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <Box className="preference-question">
      <Title order={2}>{title}</Title>
      {children}
    </Box>
  )
}

function OptionButton({
  label,
  selected,
  onClick,
  wide = false,
  disabled = false,
}: {
  label: string,
  selected: boolean,
  onClick: () => void,
  wide?: boolean,
  disabled?: boolean
}) {
  return (
     <UnstyledButton
      className={[
        'preference-option',
        selected ? 'preference-option-selected' : '',
        wide ? 'preference-option-wide' : '',
        disabled ? 'preference-option-disabled' : '',
      ].join(' ')}
      disabled={disabled}
      onClick={onClick}
    >
      <span>{label}</span>
    </UnstyledButton>
  )
}
