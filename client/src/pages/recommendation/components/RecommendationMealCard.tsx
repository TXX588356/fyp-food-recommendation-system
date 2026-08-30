import { Badge, Box, Button, Group, Text, Title } from '@mantine/core'
import type { KeyboardEvent, Ref } from 'react'

import MealPhoto from '@/pages/shared/MealPhoto'
import MealSourceBadge from '@/pages/shared/MealSourceBadge'
import type { MealCategory, VisibleRecommendationItem } from '../recommendationTypes'
import RecommendationIcon from './RecommendationIcon'

type RecommendationMealCardProps = {
  mealCategory: MealCategory
  item: VisibleRecommendationItem
  index: number
  isSpotlightTarget: boolean
  spotlightRef?: Ref<HTMLButtonElement>
  onOpenDetail: (mealCategory: MealCategory, mealId: string) => void
  onLog: () => void
}

const formatRecommendationScore = (score: number | undefined) => {
  if (typeof score !== 'number' || Number.isNaN(score)) {
    return null
  }
  return Math.round(score).toString()
}

const formatRM = (value: number) => `RM ${value.toFixed(2)}`

export default function RecommendationMealCard({
  mealCategory,
  item,
  index,
  isSpotlightTarget,
  spotlightRef,
  onOpenDetail,
  onLog,
}: RecommendationMealCardProps) {
  const { candidate, filteredReason } = item
  const recommendationScore = formatRecommendationScore(candidate.score)
  const customMealPrice =
    candidate.food.source === 'custom' && typeof candidate.food.price === 'number'
      ? candidate.food.price
      : null

  const openDetailPage = () => {
    if (filteredReason) {
      return
    }

    onOpenDetail(mealCategory, candidate.food.id)
  }

  const handleCardKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      openDetailPage()
    }
  }

  return (
    <Box
      className={`ui-meal-card ui-card ${filteredReason ? 'ui-meal-card-filtered' : 'ui-meal-card-clickable'}`}
      key={`${candidate.food.id}-${candidate.matched_query}-${index}`}
      role={filteredReason ? undefined : 'button'}
      tabIndex={filteredReason ? undefined : 0}
      onClick={openDetailPage}
      onKeyDown={handleCardKeyDown}
    >
      <MealPhoto
        imageUrl={candidate.food.image_url}
        alt={candidate.food.name}
        fallback={<RecommendationIcon name="bowl" size={24} />}
      />
      <Box className="ui-meal-summary">
        <Group gap="xs" align="center">
          <Title order={2}>{candidate.food.name}</Title>
          <MealSourceBadge source={candidate.food.source} />
          {recommendationScore && (
            <Badge className="ui-meal-score-badge">Score {recommendationScore}</Badge>
          )}
        </Group>
        <Text>{Math.round(candidate.food.calories)} kcal</Text>
        <Group gap="xs" align="center">
          <Text className="ui-meal-price">
            {customMealPrice !== null
              ? formatRM(customMealPrice)
              : `${formatRM(candidate.generated_meal.estimated_price_range.min)} - ${formatRM(candidate.generated_meal.estimated_price_range.max)}`}
          </Text>
          <Badge
            variant="gradient"
            gradient={{ from: 'rgba(37, 161, 21, 1)', to: 'rgba(247, 200, 153, 1)', deg: 90 }}
          >
            {customMealPrice !== null ? 'Actual' : 'Estimated'}
          </Badge>
        </Group>
        {filteredReason && (
          <Badge color="red" className="ui-meal-filtered-badge">
            Reason: {filteredReason}
          </Badge>
        )}
      </Box>

      <Button
        ref={isSpotlightTarget ? spotlightRef : undefined}
        className={`ui-meal-log-button ${isSpotlightTarget ? 'ui-spotlight-target' : ''}`}
        data-recommendation-tour={isSpotlightTarget ? 'log-meal' : undefined}
        variant="subtle"
        disabled={filteredReason !== null}
        onClick={(event) => {
          event.stopPropagation()
          onLog()
        }}
      >
        Log
      </Button>
    </Box>
  )
}
