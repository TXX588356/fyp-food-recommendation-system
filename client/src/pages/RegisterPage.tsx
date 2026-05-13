 import {
  Anchor,
  Box,
  Button,
  PasswordInput,
  Stack,
  Text,
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
      color: '#050505',
      fontSize: 16,
      fontWeight: 700,
      marginBottom: 8,
    },
    input: {
      height: 42,
      borderRadius: 999,
      border: focusedField === field ? '2px solid #050505' : '1px solid transparent',
      background: '#fff',
      paddingInline: 18,
      borderColor: focusedField === field ? '#050505' : 'transparent',
    },
  })

  return (
    <Box
      style={{
        minHeight: '100vh',
        width: '100%',
        background: '#fff7bc',
        display: 'flex',
        overflowX: 'hidden',
        overflowY: isMobile ? 'auto' : 'hidden',
      }}
     >
      <Box
        component='main'
        style={{
          width: '100%',
          minHeight: '100vh',
          display: 'grid',
          gridTemplateColumns: isMobile ? '1fr' : '44% 55%',
          background: '#fff7bc',
        }}
      >
        <Box
          component="section"
          style={{
            minHeight: 630,
            overflow: 'hidden',
            ...(isMobile ? { minHeight: 260 } : {}),
          }}
        >
          <img src={thumbnailImage} alt="Thumbnail Image" style={{
            display: 'block',
            objectFit: 'cover',
            width: '100%',
            height: '100%',
          }}
          />
        </Box>
        <Box
          component="section"
          style={{
            padding: isMobile ? '40px 28px 56px' : '72px 64px 48px',
            display: 'flex',
          }}>
          <Box style={{
            width: '100%',
          }}
          >
            <Text>
              Create an account or {' '} 
              <Anchor
                href="/login"
                style={{
                  color: '#050505',
                  fontWeight: 700,
                  textDecoration: 'underline',
                }}
              >login
              </Anchor>{' '} to get started
            </Text>

            <Title
              order={1}
              style={{
                margin: '0 0 28px',
                color: '#050505',
                fontWeight: 700,
                fontSize: 50,
              }}
            >
              Create Account
            </Title>

            <Box 
              component='form'
              onSubmit={handleSubmit}
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 22,
              }}
            >
              <TextInput 
              label='Name' 
              type='text'
              autoComplete='name' 
              styles={getInputStyles('name')}
              onChange={(e) => setName(e.target.value)}
              onFocus={() => setFocusedField('name')}
              onBlur={() => setFocusedField(null)}
              />
              <TextInput 
                label='Email' 
                type='email' 
                autoComplete='email' 
                styles={getInputStyles('email')}
                onChange={(e) => setEmail(e.target.value)}
                onFocus={() => setFocusedField('email')}
                onBlur={() => setFocusedField(null)}
              />
              <PasswordInput 
                label='Password' 
                autoComplete='new-password' 
                styles={getInputStyles('password')}
                onChange={(e) => setPassword(e.target.value)}
                onFocus={() => setFocusedField('password')}
                onBlur={() => setFocusedField(null)}
              />

              <Button
                type='submit'
                fullWidth
                style={{
                  height: 42, 
                  marginTop: 40,
                  borderRadius: 999,
                  background: '#000',
                  color: '#fff',
                  fontSize: 18,
                  fontWeight: 700,
                }}
              >Create Account</Button>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
