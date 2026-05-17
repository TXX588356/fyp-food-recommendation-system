import { createBrowserRouter, Navigate } from 'react-router-dom'

import RegisterPage from './pages/RegisterPage'
import LoginPage from './pages/LoginPage'
import { useAuth } from './auth/AuthContext'

function HomeRedirect() {
    const { isAuthenticated } = useAuth()
    return isAuthenticated ? <Navigate to="/recommendation" replace/> : <Navigate to="/login" />
}

// function ProtectedRoute({ children }: { children: React.ReactNode }) {
//     const { isAuthenticated } = useAuth()
//     return isAuthenticated ? children : <Navigate to="/login" replace/>
// }

export const router = createBrowserRouter([
    {path: '/', element: <HomeRedirect />,},
    {path: '/register', element: <RegisterPage />,},
    {path: '/login', element: <LoginPage />,},
    // {path: 'recommendation', element: (
    //     <ProtectedRoute>
    //         <RecommendationPage />
    //     </ProtectedRoute>
    // )}
])
