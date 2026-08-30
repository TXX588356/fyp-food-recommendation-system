import { Box, Button, Group, Text, Title } from '@mantine/core'
import type { KeyboardEvent } from 'react'

import type { ExistingMeal } from '../customMealTypes'
import MealPhoto from '@/pages/shared/MealPhoto'
import MealSourceBadge from '@/pages/shared/MealSourceBadge'
import MealAddIcon from './MealAddIcon'

type ExistingMealCardProps = {
  meal: ExistingMeal
  onOpenDetail: (meal: ExistingMeal) => void
  onLog: (meal: ExistingMeal) => void
  onKeyDown: (event: KeyboardEvent<HTMLDivElement>, meal: ExistingMeal) => void
}

export default function ExistingMealCard({
  meal,
  onOpenDetail,
  onLog,
  onKeyDown,
}: ExistingMealCardProps) {
  return (
    <Box
      className="ui-meal-card ui-card ui-meal-card-clickable"
      role="button"
      tabIndex={0}
      onClick={() => onOpenDetail(meal)}
      onKeyDown={(event) => onKeyDown(event, meal)}
    >
      <MealPhoto
        imageUrl={meal.imageUrl}
        alt={meal.name}
        fallback={<MealAddIcon name="bowl" />}
      />

      <Box className="ui-meal-summary">
        <Group gap="xs" align="center">
          <Title order={2}>{meal.name}</Title>
          <MealSourceBadge source={meal.source} />
        </Group>
        <Text>{Math.round(meal.calories)} kcal</Text>
        <Text className="ui-meal-price">{meal.priceLabel}</Text>
      </Box>

      <Button
        className="ui-meal-log-button"
        variant="subtle"
        onClick={(event) => {
          event.stopPropagation()
          onLog(meal)
        }}
      >
        Log
      </Button>
    </Box>
  )
}
