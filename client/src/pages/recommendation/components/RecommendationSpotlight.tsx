import { Box, Button, Group, Text } from '@mantine/core'
import type { ReactNode } from 'react'
import { X } from 'lucide-react'

import type { SpotlightRect } from '../recommendationTypes'

const CARD_WIDTH = 360
const CARD_ESTIMATED_HEIGHT = 230
const VIEWPORT_GAP = 16
const MOBILE_VIEWPORT_WIDTH = 680

const clamp = (value: number, min: number, max: number) => (
  Math.min(Math.max(value, min), Math.max(min, max))
)

type RecommendationSpotlightProps = {
  targetRect: SpotlightRect | null
  title: string
  description: string
  icon: ReactNode
  primaryLabel: string
  onDismiss: () => void
  onPrimary: () => void
}

export default function RecommendationSpotlight({
  targetRect,
  title,
  description,
  icon,
  primaryLabel,
  onDismiss,
  onPrimary,
}: RecommendationSpotlightProps) {
  const paddedRect = targetRect
    ? {
        top: Math.max(targetRect.top - 10, 0),
        left: Math.max(targetRect.left - 10, 0),
        width: targetRect.width + 20,
        height: targetRect.height + 20,
      }
    : null
  const useMobileLayout = window.innerWidth <= MOBILE_VIEWPORT_WIDTH
  const cardStyle = paddedRect && !useMobileLayout
    ? {
        top: clamp(
          paddedRect.top + paddedRect.height + 18,
          VIEWPORT_GAP,
          window.innerHeight - CARD_ESTIMATED_HEIGHT - VIEWPORT_GAP,
        ),
        left: clamp(
          paddedRect.left - 236,
          VIEWPORT_GAP,
          window.innerWidth - CARD_WIDTH - VIEWPORT_GAP,
        ),
        right: 'auto',
        bottom: 'auto',
      }
    : undefined

  return (
    <>
      {paddedRect ? (
        <>
          <Box
            className="ui-spotlight-scrim"
            style={{ top: 0, left: 0, right: 0, height: paddedRect.top }}
            onClick={onDismiss}
          />
          <Box
            className="ui-spotlight-scrim"
            style={{
              top: paddedRect.top,
              left: 0,
              width: paddedRect.left,
              height: paddedRect.height,
            }}
            onClick={onDismiss}
          />
          <Box
            className="ui-spotlight-scrim"
            style={{
              top: paddedRect.top,
              left: paddedRect.left + paddedRect.width,
              right: 0,
              height: paddedRect.height,
            }}
            onClick={onDismiss}
          />
          <Box
            className="ui-spotlight-scrim"
            style={{
              top: paddedRect.top + paddedRect.height,
              left: 0,
              right: 0,
              bottom: 0,
            }}
            onClick={onDismiss}
          />
        </>
      ) : (
        <Box className="ui-spotlight-scrim ui-spotlight-scrim-full" onClick={onDismiss} />
      )}

      <Box
        className="ui-spotlight-card"
        style={cardStyle}
        role="dialog"
        aria-live="polite"
        aria-label="Log meal guide"
      >
        <button
          type="button"
          className="ui-spotlight-close"
          onClick={onDismiss}
          aria-label="Dismiss guide"
        >
          <X size={16} aria-hidden="true" />
        </button>

        <Group gap="xs" align="center">
          <Box className="ui-spotlight-icon">
            {icon}
          </Box>
          <Text fw={900}>{title}</Text>
        </Group>

        <Text size="sm" className="ui-spotlight-copy">
          {description}
        </Text>

        <Group justify="flex-end" gap="xs">
          <Button size="xs" variant="subtle" onClick={onDismiss}>
            Skip
          </Button>
          <Button
            size="xs"
            className="ui-primary-button ui-spotlight-action"
            onClick={onPrimary}
          >
            {primaryLabel}
          </Button>
        </Group>
      </Box>
    </>
  )
}
