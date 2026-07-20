import { Navigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from './auth/useAuth'

export function HomeRedirect() {
    const { isAuthenticated, isAuthLoading, user } = useAuth()

    if (isAuthLoading) {
    	return null 
    }
    

    if (!isAuthenticated) {
        return <Navigate to="/login" replace/>
    }
    return user?.hasCompletedOnboarding 
    ? <Navigate to="/recommendation" replace/>
    : <Navigate to="/onboarding/preferences" replace/>

}

export function ProtectedRoute({ children }: { children: ReactNode }) {
    const { isAuthenticated, isAuthLoading, user } = useAuth()

    if (isAuthLoading) {
        return null 
    }
    
    if (!isAuthenticated) {
        return <Navigate to="/login" replace/>
    }

    if (!user?.hasCompletedOnboarding) {
        return <Navigate to="/onboarding/preferences" replace/>
    }

    return children
}

export function OnboardingRoute({ children }: { children: ReactNode }) {
    const { isAuthenticated, isAuthLoading, user } = useAuth()

		if (isAuthLoading) {
      return null 
    }
    
    if (!isAuthenticated) {
        return <Navigate to="/login" replace/>
    }

    if (user?.hasCompletedOnboarding) {
        return <Navigate to="/recommendation" replace/>
    }

    return children
}