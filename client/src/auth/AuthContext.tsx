import React, { useEffect, useState } from 'react'
import axios from 'axios'
import { AuthContext } from './auth-context'
import type { AuthContextType, User } from './auth-context'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

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
    const [isAuthLoading, setIsAuthLoading] = useState(true)

    const clearAuth = () => {
        setToken(null)
        setUser(null)
        localStorage.removeItem('token')
        localStorage.removeItem('refreshToken')
        localStorage.removeItem('user')
    }


    // Check saved session when the app first loads
    // job: 
    // Browser opens app -> check existing refresh token -> refresh/verify session immediately
    // -> load latest user -> route guards decide where to send user.
    useEffect(() => {
			const bootstrapAuth = async () => {
				const refreshToken = localStorage.getItem('refreshToken')

				if (!refreshToken) {
					clearAuth()
					setIsAuthLoading(false)
					return
				}

				try {
					const response = await axios.post(`${API_BASE_URL}/auth/refresh`, {
						refreshToken,
					})

					const nextAccessToken = response.data.accessToken
					const nextRefreshToken = response.data.refreshToken
					const nextUser = response.data.user

					setToken(nextAccessToken)
					setUser(nextUser)

					localStorage.setItem('token', nextAccessToken)
					localStorage.setItem('refreshToken', nextRefreshToken)
					localStorage.setItem('user', JSON.stringify(nextUser))
				} catch {
					clearAuth()
				} finally {
					setIsAuthLoading(false)
				}
			}

			bootstrapAuth()
    }, [] )


		// Run once to register a global response handler
		// Job:
		// User is already inside app -> some API req gets 401 -> try refresh token
		// -> retry original req -> if refresh fails, clearAuth()
		// handles token expiry during active usage
    useEffect(() => {
			const interceptor = axios.interceptors.response.use(
				(response) => response,
					async (error) => {
						const originalRequest = error.config as typeof error.config & { _retry?: boolean }
						const refreshToken = localStorage.getItem('refreshToken')

						if (
								error.response?.status !== 401 ||
								originalRequest?._retry ||
								!refreshToken ||
								originalRequest?.url?.includes('/auth/refresh')
						) {
							return Promise.reject(error)
							}

						originalRequest._retry = true

						try {
							const response = await axios.post(`${API_BASE_URL}/auth/refresh`, {
									refreshToken,
							})
							const nextAccessToken = response.data.accessToken
							const nextRefreshToken = response.data.refreshToken
							const nextUser = response.data.user

							setToken(nextAccessToken)
							setUser(nextUser)
							localStorage.setItem('token', nextAccessToken)
							localStorage.setItem('refreshToken', nextRefreshToken)
							localStorage.setItem('user', JSON.stringify(nextUser))

							originalRequest.headers = originalRequest.headers ?? {}
							originalRequest.headers.Authorization = `Bearer ${nextAccessToken}`

							return axios(originalRequest)
						} catch (refreshError) {
							clearAuth()
							return Promise.reject(refreshError)
						}
				},
			)

			return () => axios.interceptors.response.eject(interceptor)
    }, [])

    // Keep React state and localStorage in sync after a successful login.
    const login = (user: User, accessToken: string, refreshToken?: string) => {
			setIsAuthLoading(false)
        setToken(accessToken)
        setUser(user)
        localStorage.setItem('token', accessToken)
        if (refreshToken) {
            localStorage.setItem('refreshToken', refreshToken)
        }
        localStorage.setItem('user', JSON.stringify(user))
    }

    const updateUser = (nextUser: User) => {
        setUser(nextUser)
        localStorage.setItem('user', JSON.stringify(nextUser))
    }

    // Clear both in-memory auth state and persisted auth state.
    const logout = () => {
			const refreshToken = localStorage.getItem('refreshToken')
			if (refreshToken) {
					void axios.post(`${API_BASE_URL}/auth/logout`, { refreshToken }).catch(() => undefined)
			}

			clearAuth()
			setIsAuthLoading(false)

    }

    const value: AuthContextType = {
			isAuthenticated, 
			isAuthLoading,
			user,
			login,
			updateUser,
			logout,
    }

    return (
			<AuthContext.Provider value={value}>
					{children}
			</AuthContext.Provider>
    )
}

export { AuthProvider }
