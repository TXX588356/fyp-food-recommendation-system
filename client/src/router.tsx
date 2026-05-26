import { createBrowserRouter } from 'react-router-dom'

import RegisterPage from './pages/RegisterPage'
import LoginPage from './pages/LoginPage'
import RecommendationPage from './pages/RecommendationPage'
import PreferencesOnboardingPage from './pages/onboarding/PreferencesOnboardingPage'
import PreferencePage from './pages/PreferencePage'
import { HomeRedirect, ProtectedRoute, OnboardingRoute } from './RouteGuards'

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
    {path: '/onboarding/preferences', element: (
    <OnboardingRoute>
        <PreferencesOnboardingPage />
    </OnboardingRoute>

    )}


])
