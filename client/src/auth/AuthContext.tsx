import React, { useState } from 'react'
import { AuthContext } from './auth-context'
import type { AuthContextType, User } from './auth-context'

// Restore the saved user safely. If localStorage contains invalid JSON,
// clear it so the app falls back to a logged-out state.
const safeParseUser = (): User | null => {
    try {
        return JSON.parse(localStorage.getItem('user') || 'null')
    } catch {
        localStorage.removeItem('user')
        return null
    }
}

type LayoutProps = {
  children: React.ReactNode
}
const AuthProvider = ({ children }: LayoutProps) => {
    // Initialize from localStorage so auth survives page refreshes and browser restarts.
    const [token, setToken] = useState(() => localStorage.getItem('token'))
    const [user, setUser] = useState<User | null>(() => safeParseUser())
    // A user is considered authenticated when a token is present.
    const isAuthenticated = Boolean(token)

    // Keep React state and localStorage in sync after a successful login.
    const login = (user: User, token: string) => {
        setToken(token)
        setUser(user)
        localStorage.setItem('token', token)
        localStorage.setItem('user', JSON.stringify(user))
    }

    // Clear both in-memory auth state and persisted auth state.
    const logout = () => {
        setToken(null)
        setUser(null)
        localStorage.removeItem('token')
        localStorage.removeItem('user')
    }

    const value: AuthContextType = {
        isAuthenticated, 
        user,
        login,
        logout,
    }

    return (
        <AuthContext.Provider value={value}>
            {children}
        </AuthContext.Provider>
    )
}

export { AuthProvider }
