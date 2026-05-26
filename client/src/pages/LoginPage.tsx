 import {
  Anchor,
  Box,
  Button,
  PasswordInput,
  TextInput,
  Title,
  Alert,
} from '@mantine/core'
import '@/App.css'
import { useMediaQuery } from '@mantine/hooks'
import { useState } from 'react'
import { useAuth } from '@/auth/useAuth'
import axios from 'axios'
import { useForm } from '@mantine/form'

import { useNavigate, useLocation } from 'react-router-dom'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL
const thumbnailImage = 'https://images.unsplash.com/photo-1606756790138-261d2b21cd75?q=80&w=765&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D'

export default function LoginPage() {
    const isMobile = useMediaQuery('(max-width: 760px)')

    const navigate = useNavigate()
    const location = useLocation()
    // const [email, setEmail] = useState('')
    // const [password, setPassword] = useState('')
    const [isSubmitting, setIsSubmitting] = useState(false)
    const [submitError, setSubmitError] = useState<string | null>(null)
    const successMessage = location.state?.message


    const { login } = useAuth()
    const form = useForm({
      mode: 'uncontrolled',
      initialValues: {email: '', password: ''},

      validate: {
        email: (value) => (/^\S+@\S+$/.test(value) ? null : 'Invalid email'),
        password: (value) => (value.length < 6 ? 'Password must have at least 6 characters' : null), 
      }
    })

    const handleSubmit = async(values: typeof form.values) => {
      setSubmitError(null)
      setIsSubmitting(true)

      try {
        const response = await axios.post(`${API_BASE_URL}/auth/login`, values)

        login(response.data.user, response.data.token)

        if (response.data.user.hasCompletedOnboarding) {
          navigate('/recommendation', { replace: true})
        } else {
          navigate('/onboarding/preferences', { replace: true})
        }
      } catch (error) {
        console.error('Login failed', error)
        setSubmitError('Login failed. Please check your email and password')
      } finally {
        setIsSubmitting(false)
      }
    }

    const inputClassNames = {
      label: 'auth-input-label',
      input: 'auth-input',
      wrapper: 'auth-input-wrapper',
      innerInput: 'auth-password-inner-input',
    }

    const inputStyles = {
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
        background: 'rgba(255, 251, 239, 0.92)',
        color: '#17241e',
        fontFamily: 'inherit',
        paddingInline: 18,
        transition: 'border-color 180ms ease, box-shadow 180ms ease, background 180ms ease',
      },
    }

   return (
      <Box
        className="auth-page"
        
        style={{ overflowY: isMobile ? 'auto' : 'hidden', height: '100vh'}}
       >
        <Box
          component='main'
          className="auth-shell"
        >
          <Box
            component="section"
            className="auth-visual"
          >
            <img src={thumbnailImage} alt="food image" />
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
  
              {successMessage && <Alert color="green">{successMessage}</Alert>}
              {submitError && <Alert color="red">{submitError}</Alert>}
              <Box 
                component='form'
                onSubmit={form.onSubmit(handleSubmit)}
                className="auth-form">
                <TextInput 
                label='Email' 
                autoComplete='email' 
                placeholder="you@exmaple.com"
                classNames={inputClassNames}
                styles={inputStyles}
                key={form.key('email')}
                {...form.getInputProps('email')}
                />
                <PasswordInput 
                  label='Password' 
                  autoComplete='new-password' 
                  placeholder="Your password"
                  classNames={inputClassNames}
                  styles={inputStyles}
                  key={form.key('password')}
                {...form.getInputProps('password')}
                />

                <Button
                  type='submit'
                  fullWidth
                  className="auth-button"
                  color='#00754A'
                  size='md'
                  loading={isSubmitting}
                  disabled={isSubmitting}
                >Login</Button>
              </Box>
            </Box>
          </Box>
        </Box>
      </Box>
    )
  }
