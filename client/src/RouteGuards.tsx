import { Navigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from './auth/useAuth'

export function HomeRedirect() {
    const { isAuthenticated } = useAuth()
    return isAuthenticated ? <Navigate to="/recommendation" replace/> : <Navigate to="/login" />
}

export function ProtectedRoute({ children }: { children: ReactNode }) {
    const { isAuthenticated } = useAuth()
    return isAuthenticated ? children : <Navigate to="/login" replace/>
}

export function OnboardingRoute({ children }: { children: ReactNode }) {
    const { isAuthenticated, user } = useAuth()

    if (!isAuthenticated) {
        return <Navigate to="/login" replace/>
    }

    if (user?.hasCompletedOnboarding) {
        return <Navigate to="/recommendation" replace/>
    }

    return children
}