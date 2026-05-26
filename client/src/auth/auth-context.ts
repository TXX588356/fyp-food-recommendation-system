import { createContext } from 'react'

export type User = {
    id?: string
    name?: string
    email?: string
    hasCompletedOnboarding?: boolean
}

export type AuthContextType = {
    isAuthenticated: boolean
    user: User | null
    login: (user: User, token: string) => void
    logout: () => void
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined)
