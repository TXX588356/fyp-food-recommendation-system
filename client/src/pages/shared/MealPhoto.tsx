import { Box } from '@mantine/core'
import type { ReactNode } from 'react'

type MealPhotoProps = {
  imageUrl?: string
  alt: string
  fallback: ReactNode
}

export default function MealPhoto({ imageUrl, alt, fallback }: MealPhotoProps) {
  return (
    <Box className="ui-meal-photo" aria-hidden={!imageUrl}>
      {imageUrl ? (
        <img
          src={imageUrl}
          alt={alt}
          loading="lazy"
          referrerPolicy="no-referrer"
        />
      ) : (
        fallback
      )}
    </Box>
  )
}
