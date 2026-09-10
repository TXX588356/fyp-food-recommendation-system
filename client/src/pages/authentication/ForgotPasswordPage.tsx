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
import axios from 'axios'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { replayInputShake } from '@/theme/inputTransitions'

const thumbnailImage = 'https://images.unsplash.com/photo-1606756790138-261d2b21cd75?q=80&w=765&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D'

export default function ForgotPasswordPage() {
  const navigate = useNavigate()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  const form = useForm({
    mode: 'uncontrolled',
    initialValues: { email: '', newPassword: '', confirmPassword: '' },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value) ? null : 'Invalid email'),
      newPassword: (value) => (value.length < 6 ? 'Password must have at least 6 characters' : null),
      confirmPassword: (value, values) => (
        value === values.newPassword ? null : 'Passwords do not match'
      ),
    },
  })

  const inputClassNames = {
    label: 'ui-input-label',
    input: 'ui-input t-input',
    wrapper: 'ui-input-wrapper t-input-wrap',
    innerInput: 'ui-password-inner-input',
  }

  const handleValidationFailure = () => {
    replayInputShake('.ui-form .t-input[data-error], .ui-form .t-input.is-error')
  }

  const handleSubmit = async (values: typeof form.values) => {
    setSubmitError(null)
    setIsSubmitting(true)

    try {
      await axios.post('/auth/reset-password', {
        email: values.email,
        newPassword: values.newPassword,
      })

      navigate('/login', {
        replace: true,
        state: { message: 'Password updated. Please log in with your new password.' },
      })
    } catch (error) {
      console.error('Password reset failed', error)
      const message = axios.isAxiosError(error) && typeof error.response?.data?.error === 'string'
        ? error.response.data.error
        : 'Unable to reset password. Request a new reset link and try again.'

      setSubmitError(message)
      if (/email/i.test(message)) {
        form.setFieldError('email', message)
      } else {
        form.setFieldError('newPassword', message)
      }
      replayInputShake('.ui-form .t-input[data-error], .ui-form .t-input.is-error')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Box className="ui-fixed-page">
      <Box component="main" className="ui-split-shell">
        <Box component="section" className="ui-visual-pane">
          <img src={thumbnailImage} alt="food image" />
          <Box className="ui-visual-caption">
            <span>Food Recommendation System</span>
            <Title order={2} style={{ color: '#fff8e8' }}>Get back to the profile that keeps your meals familiar.</Title>
          </Box>
        </Box>

        <Box component="section" className="ui-centered-content">
          <Box className="ui-form-card">
            <Box className="ui-form-topline">
              <span>account recovery</span>
              <Anchor href="/login">Log in</Anchor>
            </Box>

            <Title order={1} className="ui-hero-heading">
              Reset your password.
            </Title>
            <p className="ui-body-copy">
              Enter the email attached to your food profile and choose a new password.
            </p>

            {submitError && <Alert color="red">{submitError}</Alert>}

            <Box
              component="form"
              onSubmit={form.onSubmit(handleSubmit, handleValidationFailure)}
              className="ui-form"
              noValidate
            >
              <TextInput
                label="Email"
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                classNames={inputClassNames}
                key={form.key('email')}
                {...form.getInputProps('email')}
              />
              <PasswordInput
                label="New password"
                autoComplete="new-password"
                placeholder="At least 6 characters"
                classNames={inputClassNames}
                key={form.key('newPassword')}
                {...form.getInputProps('newPassword')}
              />
              <PasswordInput
                label="Confirm password"
                autoComplete="new-password"
                placeholder="Repeat your new password"
                classNames={inputClassNames}
                key={form.key('confirmPassword')}
                {...form.getInputProps('confirmPassword')}
              />

              <Button
                type="submit"
                fullWidth
                className="ui-primary-button"
                color="#00754A"
                size="md"
                loading={isSubmitting}
                disabled={isSubmitting}
              >
                Reset Password
              </Button>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
