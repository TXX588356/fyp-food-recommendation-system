 import {
  Anchor,
  Box,
  Button,
  PasswordInput,
  Stack,
  TextInput,
  Title,
} from '@mantine/core'
import '@/App.css'
import '@/index.css'
import '@mantine/core/styles.css'
import { useMediaQuery } from '@mantine/hooks'
import { useState } from 'react'

const thumbnailImage = 'https://miro.medium.com/v2/resize:fit:1400/1*pQDb49sa3kzRGxV_FYQJjQ.jpeg'

export default function LoginPage() {
    const isMobile = useMediaQuery('(max-width: 760px)')
    const [focusedField, setFocusedField] = useState<string | null>(null)

    const getInputStyles = (field: string) => ({
    label: {
      color: '#20342b',
      fontSize: 13,
      fontWeight: 800,
      letterSpacing: '0.08em',
      marginBottom: 10,
      textTransform: 'uppercase' as const,
    },
    input: {
      height: 54,
      borderRadius: 18,
      border: focusedField === field ? '2px solid #0f6b43' : '1px solid rgba(32, 52, 43, 0.22)',
      background: 'rgba(255, 251, 239, 0.92)',
      color: '#17241e',
      paddingInline: 18,
      boxShadow: focusedField === field ? '0 0 0 5px rgba(15, 107, 67, 0.12)' : 'none',
      transition: 'border-color 180ms ease, box-shadow 180ms ease, background 180ms ease',
    },
  })

   return (
      <Box
        className="auth-page"
        style={{ overflowY: isMobile ? 'auto' : 'hidden' }}
       >
        <Box
          component='main'
          className="auth-shell"
        >
          <Box
            component="section"
            className="auth-visual"
          >
            <img src={thumbnailImage} alt="A table with fresh food and coffee" />
            <Box className="visual-caption">
              <span>Food Recommendation System</span>
              <Title order={2}>Return to the profile that knows your food rhythm.</Title>
            </Box>
          </Box>
          <Box
            component="section"
            className="auth-content"
          >
            <Box className="auth-card">
              <Box className="auth-topline">
                <span>saved profile</span>
                <Anchor href="/register">Create an account</Anchor>
              </Box>
  
              <Title
                order={1}
                className="auth-heading"
              >
                Welcome back.
              </Title>
              <p className="auth-copy">
                Pick up where you left off with saved preferences, previous meal choices, and better ranked recommendations.
              </p>
  
              <Stack gap={22} className="auth-form">
                <TextInput 
                label='Name' 
                autoComplete='name' 
                placeholder="Your account name"
                styles={getInputStyles('name')}
                onFocus={() => setFocusedField('name')}
                onBlur={() => setFocusedField(null)}
                />
                <PasswordInput 
                  label='Password' 
                  autoComplete='new-password' 
                  placeholder="Your password"
                  styles={getInputStyles('password')}
                  onFocus={() => setFocusedField('password')}
                  onBlur={() => setFocusedField(null)}
                />
  
                <Button
                  type='submit'
                  fullWidth
                  className="auth-button"
                >Login</Button>
              </Stack>
              <Box className="recommendation-preview">
                <span className="preview-dot" />
                <div>
                  <strong>Ready today</strong>
                  <p>Your saved taste profile can rank breakfast, lunch, and dinner ideas immediately.</p>
                </div>
              </Box>
            </Box>
          </Box>
        </Box>
      </Box>
    )
  }
