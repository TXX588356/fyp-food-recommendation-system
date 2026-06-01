import { Alert, Box, Button, Group, Progress, Text, Title } from "@mantine/core"
import { useForm } from "@mantine/form"
import { useState } from "react"
import { useNavigate } from "react-router-dom"
import "@/App.css"
import axios from "axios"
import { useAuth } from "@/auth/useAuth"
import { type PreferencesFormValues } from "@/preferences/types"
import { buildPreferencePayload, createEmptyLocation, filterMealTagsForDietaryRestrictions } from "@/preferences/helpers"
import { GoalStep } from "../steps/MainGoalPage"
import { DietaryRestrictionStep } from "../steps/DietaryRestrictionPage"
import { HealthConcernStep } from "../steps/HealthConcernPage"
import { BudgetLocationStep } from "../steps/BudgetLocationPage"
import { MealPreferenceStep } from "../steps/PreferredMealPage"

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

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

    try {
      const preferencePayload = buildPreferencePayload(form.values)

      await axios.post(`${API_BASE_URL}/preferences`, preferencePayload, {
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
      setSubmitError('Could not save your preferences. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Box className="ui-page">
      <Box component="main" className="ui-page-shell">
        <Box component="section" className="ui-feature-panel">
          <Box className="ui-page-header">
            <Box className="ui-topline">
              <Text>Step {step} of 5</Text>
            </Box>

            <Title order={1}>Let us get to know you more.</Title>
            <Text className="ui-page-copy">{stepDescriptions[currentStepIndex]}</Text>

            <Box className="ui-progress">
              <Progress value={progress} color="#0f6b43" size="sm" radius="xl" />
              <Box className="ui-step-list">
                {stepLabels.map((label, index) => (
                  <Box
                    key={label}
                    className={[
                      'ui-step-pill',
                      index + 1 === step ? 'ui-step-pill-active' : '',
                      index + 1 < step ? 'ui-step-pill-complete' : '',
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
            <GoalStep 
              value={form.values.mainGoal}
              onChange={(value) => form.setFieldValue('mainGoal', value)}
            />
          )}

          {step === 2 && (
            <DietaryRestrictionStep 
              value={form.values.dietaryRestrictions}
              onChange={(value) => {
                form.setFieldValue('dietaryRestrictions', value)
                form.setFieldValue(
                  'preferredMealCategories',
                  filterMealTagsForDietaryRestrictions(
                    form.values.preferredMealCategories,
                    value,
                  ),
                )
              }}
            />
          )}

          {step === 3 && (
            <HealthConcernStep 
              value={form.values.healthConcerns}
              onChange={(value) => form.setFieldValue('healthConcerns', value)}
            />
          )}

          {step === 4 && (
            <BudgetLocationStep 
              monthlyMealBudget={form.values.monthlyMealBudget}
              onMonthlyMealBudgetChange={(value) => 
                form.setFieldValue("monthlyMealBudget", value)
              }
              workSchoolLocation={form.values.workSchoolLocation}
              onWorkSchoolLocationChange={(value) => 
                form.setFieldValue("workSchoolLocation", value)
              }
              homeLocation={form.values.homeLocation}
              onHomeLocationChange={(value) => 
                form.setFieldValue("homeLocation", value)
              }
            />
          )}

          {step === 5 && (
            <MealPreferenceStep 
              value={form.values.preferredMealCategories}
              dietaryRestrictions={form.values.dietaryRestrictions}
              onChange={(value) => 
                form.setFieldValue("preferredMealCategories", value)
              }
            />
          )}

          <Group justify="space-between" className="ui-actions">
            <Button
              variant="subtle"
              className="ui-ghost-button"
              leftSection={<span aria-hidden="true">←</span>}
              onClick={handleBack}
              disabled={step === 1 || isSubmitting}
            >
              Back
            </Button>

            {step < 5 ? (
              <Button
                className="ui-primary-button ui-action-button"
                rightSection={<span aria-hidden="true">→</span>}
                onClick={handleNext}
                disabled={!canContinue()}
              >
                Next
              </Button>
            ) : (
              <Button
                className="ui-primary-button ui-action-button"
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
