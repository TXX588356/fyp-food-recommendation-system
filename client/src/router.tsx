import { createBrowserRouter } from 'react-router-dom'

import RegisterPage from './pages/RegisterPage'
import LoginPage from './pages/LoginPage'
import RecommendationPage from './pages/RecommendationPage'
import PreferencesOnboardingPage from './pages/onboarding/PreferencesOnboardingPage'
import PreferencePage from './pages/preferences/PreferencePage'
import { HomeRedirect, ProtectedRoute, OnboardingRoute } from './RouteGuards'
import DietaryRestrictionsPreferencePage from './pages/preferences/DietaryRestrictionsPreferencePage'
import MainGoalPreferencePage from './pages/preferences/MainGoalPreferencePage'
import HealthConcernsPreferencePage from './pages/preferences/HealthConcernsPreferencePage'
import PreferredMealPreferencePage from './pages/preferences/PreferredMealPreferencePage'
import BudgetLocationPreferencePage from './pages/preferences/BudgetLocationPreferencePage'
import ConsentPreferencePage from './pages/preferences/ConsentPreferencePage'

export const router = createBrowserRouter([
  {path: '/', element: <HomeRedirect />,},
  {path: '/register', element: <RegisterPage />,},
  {path: '/login', element: <LoginPage />,},
  {path: 'recommendation', element: (
    <ProtectedRoute>
      <RecommendationPage />
    </ProtectedRoute>
  )},
  {path: '/preferences', element: (
    <ProtectedRoute>
      <PreferencePage />
    </ProtectedRoute>
  )},
  {path: '/preferences/dietaryRestrictions', element: (
    <ProtectedRoute>
      <DietaryRestrictionsPreferencePage />
    </ProtectedRoute>
  )},
  {path: '/preferences/healthConcerns', element: (
    <ProtectedRoute>
      <HealthConcernsPreferencePage />
    </ProtectedRoute>
  )},
  {path: '/preferences/goal', element: (
    <ProtectedRoute>
      <MainGoalPreferencePage />
    </ProtectedRoute>
  )},
  {path: '/preferences/mealPreferences', element: (
    <ProtectedRoute>
      <PreferredMealPreferencePage />
    </ProtectedRoute>
  )},
  {path: '/preferences/budget-location', element: (
    <ProtectedRoute>
      <BudgetLocationPreferencePage />
    </ProtectedRoute>
  )},
  {path: '/preferences/consent', element: (
    <ProtectedRoute>
      <ConsentPreferencePage />
    </ProtectedRoute>
  )},
  {path: '/onboarding/preferences', element: (
    <OnboardingRoute>
      <PreferencesOnboardingPage />
    </OnboardingRoute>
  )}
])
