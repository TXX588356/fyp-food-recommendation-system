import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { MantineProvider } from '@mantine/core'
import axios from 'axios'
import '@mantine/core/styles.css'
import '@/index.css'
import '@/App.css'
import '@/theme/inputTransitions'
import App from '@/App.tsx'
import { AuthProvider } from '@/auth/AuthContext';

const appFontFamily = '"Avenir Next", "Gill Sans", Candara, Calibri, sans-serif'

axios.defaults.baseURL = '/api'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AuthProvider>
      <MantineProvider theme={{ fontFamily: appFontFamily }}>
          <App />
        </MantineProvider>
    </AuthProvider>
  </StrictMode>,
)
