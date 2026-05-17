import { createBrowserRouter } from 'react-router-dom'

import RegisterPage from './pages/RegisterPage'
import LoginPage from './pages/LoginPage'

// function ProtectedRoute({ children }: { children: React.ReactNode }) {
//     const { isAuthenticated } = useAuth()
//     return isAuthenticated ? children : <Navigate to="/login" replace/>
// }

export const router = createBrowserRouter([
    {path: '/', element: <RegisterPage />,},
    {path: '/register', element: <RegisterPage />,},
    {path: '/login', element: <LoginPage />,},
    // {path: 'recommendation', element: (
    //     <ProtectedRoute>
    //         <RecommendationPage />
    //     </ProtectedRoute>
    // )}
])
