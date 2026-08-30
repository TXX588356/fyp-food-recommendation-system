import {
  Alert,
  Anchor,
  Box,
  Button,
  PasswordInput,
  TextInput,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { useState } from 'react'
import axios from 'axios'
import { useNavigate } from 'react-router-dom'
import { replayInputShake } from '@/theme/inputTransitions'

const thumbnailImage = 'https://images.unsplash.com/photo-1606756790138-261d2b21cd75?q=80&w=765&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D'

export default function RegisterPage() {
  // const [name, setName] = useState('')
  // const [email, setEmail] = useState('')
  // const [password, setPassword] = useState('')

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
      const response = await axios.post("/auth/register", values)

      navigate('/login', {replace: true, state: { message: 'Account created. Please log in.' }})
      console.log('Registration successful: ', response.data);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const message = error.response?.data?.error

        const errorMessage = typeof message === 'string'
          ? message
          : 'Registration failed. Please check your details and try again.'

        setSubmitError(errorMessage)
        if (/email/i.test(errorMessage)) {
          form.setFieldError('email', errorMessage)
          replayInputShake('.ui-form .t-input[data-error], .ui-form .t-input.is-error')
        }
      } else {
        setSubmitError('Registration failed. Please check your details and try again.')

      }
      console.error('Reigstration failed: ', error)
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
            <Title order={2} style={{color: '#fff8e8'}}>Build a palate profile that remembers what you actually like.</Title>
          </Box>
        </Box>
        <Box
          component="section"
          className="ui-centered-content"
        >
          <Box className="ui-form-card">
            <Box className="ui-form-topline">
              <span>new profile</span>
              <Anchor href="/login">Log in</Anchor>
            </Box>

            <Title
              order={1}
              className="ui-hero-heading"
            >
              Create your taste account.
            </Title>
            <p className="ui-body-copy">
              Save cravings, dietary preferences, and meal rhythms so recommendations feel less random and more like a regular order.
            </p>

            {submitError && (<Alert color="red">{submitError}</Alert>)}
            <Box 
              component='form'
              onSubmit={form.onSubmit(handleSubmit, handleValidationFailure)}
              className="ui-form"
              noValidate
            >
              <TextInput 
                label='Name' 
                type='text'
                autoComplete='name' 
                placeholder="e.g. John Doe"
                classNames={inputClassNames}
                key={form.key('name')}
                {...form.getInputProps('name')}
              />
              <TextInput 
                label='Email' 
                type='email' 
                autoComplete='email' 
                placeholder="you@example.com"
                classNames={inputClassNames}
                key={form.key('email')}
                {...form.getInputProps('email')}
              />
              <PasswordInput 
                label='Password' 
                autoComplete='new-password' 
                placeholder="At least 6 characters"
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
              >Create Account</Button>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
