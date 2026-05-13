import {
  Anchor,
  Box,
  Button,
  PasswordInput,
  TextInput,
  Title,
} from '@mantine/core'
import '@/App.css'
import '@/index.css'
import '@mantine/core/styles.css'
import { useMediaQuery } from '@mantine/hooks'
import React, { useState } from 'react'
import axios from 'axios'

const thumbnailImage = 'https://miro.medium.com/v2/resize:fit:1400/1*pQDb49sa3kzRGxV_FYQJjQ.jpeg'

export default function RegisterPage() {
  const isMobile = useMediaQuery('(max-width: 760px)')
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [focusedField, setFocusedField] = useState<string | null>(null)
  const API_BASE_URL = import.meta.env?.VITE_API_BASE_URL
  
  const handleSubmit = async(e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if(password.length < 6) {
      //TODO
    }
    try {
      const response = await axios.post(`${API_BASE_URL}/auth/register`, {
        name, 
        email,
        password,
      })
      console.log('Registration successful: ', response.data);
    } catch (error) {
      console.error('Reigstration failed: ', error)
    }
  }

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
            <Title order={2}>Build a palate profile that remembers what you actually like.</Title>
          </Box>
        </Box>
        <Box
          component="section"
          className="auth-content"
        >
          <Box className="auth-card">
            <Box className="auth-topline">
              <span>new profile</span>
              <Anchor href="/login">Log in</Anchor>
            </Box>

            <Title
              order={1}
              className="auth-heading"
            >
              Create your taste account.
            </Title>
            <p className="auth-copy">
              Save cravings, dietary preferences, and meal rhythms so recommendations feel less random and more like a regular order.
            </p>

            <Box 
              component='form'
              onSubmit={handleSubmit}
              className="auth-form"
            >
              <TextInput 
                label='Name' 
                type='text'
                autoComplete='name' 
                placeholder="e.g. John Doe"
                styles={getInputStyles('name')}
                onChange={(e) => setName(e.target.value)}
                onFocus={() => setFocusedField('name')}
                onBlur={() => setFocusedField(null)}
              />
              <TextInput 
                label='Email' 
                type='email' 
                autoComplete='email' 
                placeholder="you@example.com"
                styles={getInputStyles('email')}
                onChange={(e) => setEmail(e.target.value)}
                onFocus={() => setFocusedField('email')}
                onBlur={() => setFocusedField(null)}
              />
              <PasswordInput 
                label='Password' 
                autoComplete='new-password' 
                placeholder="At least 6 characters"
                styles={getInputStyles('password')}
                onChange={(e) => setPassword(e.target.value)}
                onFocus={() => setFocusedField('password')}
                onBlur={() => setFocusedField(null)}
              />

              <Button
                type='submit'
                fullWidth
                className="auth-button"
              >Create Account</Button>
            </Box>
            <Box className="recommendation-preview">
              <span className="preview-dot" />
              <div>
                <strong>Next up</strong>
                <p>Breakfast bowls ranked by protein, budget, and your saved taste notes.</p>
              </div>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
