import React, { useEffect, useState } from 'react'
import axios from 'axios'
import { AuthContext } from './auth-context'
import type { AuthContextType, User } from './auth-context'


type RefreshResponse = {
    accessToken: string
    refreshToken: string
    user: User
}

let refreshSessionPromise: Promise<RefreshResponse> | null = null

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

    const applyAuthSession = (session: RefreshResponse) => {
        setToken(session.accessToken)
        setUser(session.user)

        localStorage.setItem('token', session.accessToken)
        localStorage.setItem('refreshToken', session.refreshToken)
        localStorage.setItem('user', JSON.stringify(session.user))
    }

    const refreshSession = async () => {
        if (refreshSessionPromise) {
            return refreshSessionPromise
        }

        const refreshToken = localStorage.getItem('refreshToken')
        if (!refreshToken) {
            return Promise.reject(new Error('missing refresh token'))
        }

        refreshSessionPromise = axios.post<RefreshResponse>("/auth/refresh", {
            refreshToken,
        }).then((response) => response.data)
            .finally(() => {
                refreshSessionPromise = null
            })

        return refreshSessionPromise
    }

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
					const nextSession = await refreshSession()
					applyAuthSession(nextSession)
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
			const requestInterceptor = axios.interceptors.request.use((config) => {
				if (
					config.url?.includes('/auth/login') ||
					config.url?.includes('/auth/register') ||
					config.url?.includes('/auth/refresh')
				) {
					return config
				}

				const currentToken = localStorage.getItem('token')
				if (currentToken) {
					config.headers = config.headers ?? {}
					config.headers.Authorization = `Bearer ${currentToken}`
				}

				return config
			})

			const responseInterceptor = axios.interceptors.response.use(
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
							const nextSession = await refreshSession()
							applyAuthSession(nextSession)

							originalRequest.headers = originalRequest.headers ?? {}
							originalRequest.headers.Authorization = `Bearer ${nextSession.accessToken}`

							return axios(originalRequest)
						} catch (refreshError) {
							clearAuth()
							return Promise.reject(refreshError)
						}
				},
			)

			return () => {
				axios.interceptors.request.eject(requestInterceptor)
				axios.interceptors.response.eject(responseInterceptor)
			}
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
					void axios.post("/auth/logout", { refreshToken }).catch(() => undefined)
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
