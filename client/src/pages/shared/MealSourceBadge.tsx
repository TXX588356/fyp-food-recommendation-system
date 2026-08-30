import { Badge } from '@mantine/core'

type MealSourceBadgeProps = {
  source?: 'prebuilt' | 'custom'
}

export default function MealSourceBadge({ source }: MealSourceBadgeProps) {
  if (source !== 'custom') {
    return null
  }

  return <Badge className="ui-meal-source-badge">Community</Badge>
}
