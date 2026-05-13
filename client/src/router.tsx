import { BrowserRouter, createBrowserRouter, Routes, Route } from 'react-router-dom'

import RegisterPage from './pages/RegisterPage'
import LoginPage from './pages/LoginPage'
import { AuthProvider } from './auth/AuthContext'


export const router = createBrowserRouter([
    {path: '/', element: <RegisterPage />,},
    {path: '/register', element: <RegisterPage />,},
    {path: '/login', element: <LoginPage />,},
])

export default function Router() {
    return (
        <BrowserRouter>
        <AuthProvider>
            <Routes>
                {/* TODO <Route> */}
                    <Route path="/" element={<RegisterPage />} />
                    <Route path="/register" element={<RegisterPage />} />
                    <Route path="/login" element={<LoginPage />} />
                {/* </Route> */} 
            </Routes>
        </AuthProvider>
        </BrowserRouter>
    )
}