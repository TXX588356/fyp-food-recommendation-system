 import {
  Anchor,
  Box,
  Button,
  PasswordInput,
  TextInput,
  Title,
  Alert,
} from '@mantine/core'
import { useState } from 'react'
import { useAuth } from '@/auth/useAuth'
import axios from 'axios'
import { useForm } from '@mantine/form'

import { useNavigate, useLocation } from 'react-router-dom'
import { replayInputShake } from '@/theme/inputTransitions'

const thumbnailImage = 'https://images.unsplash.com/photo-1606756790138-261d2b21cd75?q=80&w=765&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D'

export default function LoginPage() {
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
        const response = await axios.post("/auth/login", values)

        login(response.data.user, response.data.accessToken, response.data.refreshToken)

        if (response.data.user.hasCompletedOnboarding) {
          navigate('/recommendation', { replace: true})
        } else {
          navigate('/onboarding/preferences', { replace: true})
        }
      } catch (error) {
        console.error('Login failed', error)
        const message = 'Login failed. Please check your email and password'
        setSubmitError(message)
        form.setFieldError('password', message)
        replayInputShake('.ui-form .t-input[data-error], .ui-form .t-input.is-error')
      } finally {
        setIsSubmitting(false)
      }
    }

    const handleValidationFailure = () => {
      replayInputShake('.ui-form .t-input[data-error], .ui-form .t-input.is-error')
    }

    const inputClassNames = {
      label: 'ui-input-label',
      input: 'ui-input t-input',
      wrapper: 'ui-input-wrapper t-input-wrap',
      innerInput: 'ui-password-inner-input',
    }

   return (
      <Box className="ui-fixed-page">
        <Box
          component='main'
          className="ui-split-shell"
        >
          <Box
            component="section"
            className="ui-visual-pane"
          >
            <img src={thumbnailImage} alt="food image" />
            <Box className="ui-visual-caption">
              <span>Food Recommendation System</span>
              <Title order={2} style={{color: '#fff8e8'}}>Return to the profile that knows your food rhythm.</Title>
            </Box>
          </Box>
          <Box
            component="section"
            className="ui-centered-content"
            
          >
            <Box className="ui-form-card">
              <Box className="ui-form-topline">
                <span>saved profile</span>
                <Anchor href="/register">Create an account</Anchor>
              </Box>
  
              <Title
                order={1}
                className="ui-hero-heading"
              >
                Welcome back.
              </Title>
              <p className="ui-body-copy">
                Pick up where you left off with saved preferences, previous meal choices, and better ranked recommendations.
              </p>
  
              {successMessage && <Alert color="green">{successMessage}</Alert>}
              {submitError && <Alert color="red">{submitError}</Alert>}
                <Box 
                component='form'
                onSubmit={form.onSubmit(handleSubmit, handleValidationFailure)}
                className="ui-form"
                noValidate
              >
                <TextInput 
                label='Email' 
                autoComplete='email' 
                placeholder="you@exmaple.com"
                classNames={inputClassNames}
                key={form.key('email')}
                {...form.getInputProps('email')}
                />
                <PasswordInput 
                  label='Password' 
                  autoComplete='new-password' 
                  placeholder="Your password"
                  classNames={inputClassNames}
                  key={form.key('password')}
                {...form.getInputProps('password')}
                />

                <Button
                  type='submit'
                  fullWidth
                  className="ui-primary-button"
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
