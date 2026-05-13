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
import { useState } from 'react'

const thumbnailImage = 'https://miro.medium.com/v2/resize:fit:1400/1*pQDb49sa3kzRGxV_FYQJjQ.jpeg'

export default function LoginPage() {
    const isMobile = useMediaQuery('(max-width: 760px)')
    const [focusedField, setFocusedField] = useState<string | null>(null)

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
                <Anchor
                  href="/register"
                  style={{
                    color: '#050505',
                    fontWeight: 700,
                    textDecoration: 'underline',
                  }}
                >Create an account
                </Anchor> or login to get started
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
                Welcome!
              </Title>
  
              <Stack gap={22}>
                <TextInput 
                label='Name' 
                autoComplete='name' 
                styles={getInputStyles('name')}
                onFocus={() => setFocusedField('name')}
                onBlur={() => setFocusedField(null)}
                />
                <PasswordInput 
                  label='Password' 
                  autoComplete='new-password' 
                  styles={getInputStyles('password')}
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
                >Login</Button>
              </Stack>
            </Box>
          </Box>
        </Box>
      </Box>
    )
  }
