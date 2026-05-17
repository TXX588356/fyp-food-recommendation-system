import React, { createContext, useContext, useState } from 'react';

type User = {
    name?: string
    email?: string
}
type AuthContextType = {
    isAuthenticated: boolean
    user: User | null
    login: (user: User, token: string) => void
    logout: () => void
}

const safeParseUser = (): User | null => {
    try {
        return JSON.parse(localStorage.getItem('user') || 'null')
    } catch {
        localStorage.removeItem('user')
        return null
    }
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

type LayoutProps = {
  children: React.ReactNode
}
const AuthProvider = ({ children }: LayoutProps) => {
    const [token, setToken] = useState(() => localStorage.getItem('token'))
    const [user, setUser] = useState<User | null>(() => safeParseUser())
    const isAuthenticated = Boolean(token)

    // const [isAuthenticated, setIsAuthenticated] = useState(
    //     localStorage.getItem('isAuthenticated') === 'true'
    // )


    const login = (user: User, token: string) => {
        setToken(token)
        setUser(user)
        localStorage.setItem('token', token)
        localStorage.setItem('user', JSON.stringify(user))
    }

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

// custom hook to use the auth context
const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export {AuthProvider, useAuth }
