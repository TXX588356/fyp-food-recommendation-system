import { createContext } from 'react'

export type User = {
    id?: string
    name?: string
    email?: string
    hasCompletedOnboarding?: boolean
}

export type AuthContextType = {
    isAuthenticated: boolean
    isAuthLoading: boolean
    user: User | null
    login: (user: User, accessToken: string, refreshToken?: string) => void
    updateUser: (user: User) => void
    logout: () => void
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined)
