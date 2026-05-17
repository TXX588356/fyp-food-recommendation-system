import {
  Alert,
  Anchor,
  Box,
  Button,
  PasswordInput,
  TextInput,
  Title,
} from '@mantine/core'
import '@/App.css'
import { useMediaQuery } from '@mantine/hooks'
import { useForm } from '@mantine/form'
import { useState } from 'react'
import axios from 'axios'
import { useNavigate } from 'react-router-dom'

const thumbnailImage = 'https://images.unsplash.com/photo-1606756790138-261d2b21cd75?q=80&w=765&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D'

export default function RegisterPage() {
  const isMobile = useMediaQuery('(max-width: 760px)')
  // const [name, setName] = useState('')
  // const [email, setEmail] = useState('')
  // const [password, setPassword] = useState('')
  const API_BASE_URL = import.meta.env?.VITE_API_BASE_URL

  const form = useForm({
    mode: 'uncontrolled',
    initialValues: { name: '', email:'', password: ''},

    validate: {
        name: (value) => (value.trim().length < 2 ? 'Name must have at least 2 characters' : null),
        email: (value) => (/^\S+@\S+$/.test(value) ? null : 'Invalid email'),
        password: (value) => (value.length < 6 ? 'Password must have at least 6 characters' : null), 
      },
    })

    const navigate = useNavigate()
    const [isSubmitting, setIsSubmitting] = useState(false)
    const [submitError, setSubmitError] = useState<string | null>(null)

  const handleSubmit = async(values: typeof form.values) => {
    setSubmitError(null)
    setIsSubmitting(true)
    //e.preventDefault()
    try {
      const response = await axios.post(`${API_BASE_URL}/auth/register`, values)

      navigate('/login', {replace: true, state: { message: 'Account created. Please log in.' }})
      console.log('Registration successful: ', response.data);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const message = error.response?.data?.error

        setSubmitError(typeof message === 'string' ? message : 'Registration failed. Please check your details and try again.')
      } else {
        setSubmitError('Registration failed. Please check your details and try again.')

      }
      console.error('Reigstration failed: ', error)
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

            {submitError && (<Alert color="red">{submitError}</Alert>)}
            <Box 
              component='form'
              onSubmit={form.onSubmit(handleSubmit)}
              className="auth-form"
              noValidate
            >
              <TextInput 
                label='Name' 
                type='text'
                autoComplete='name' 
                placeholder="e.g. John Doe"
                classNames={inputClassNames}
                styles={inputStyles}
                key={form.key('name')}
                {...form.getInputProps('name')}
              />
              <TextInput 
                label='Email' 
                type='email' 
                autoComplete='email' 
                placeholder="you@example.com"
                classNames={inputClassNames}
                styles={inputStyles}
                key={form.key('email')}
                {...form.getInputProps('email')}
              />
              <PasswordInput 
                label='Password' 
                autoComplete='new-password' 
                placeholder="At least 6 characters"
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
              >Create Account</Button>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
