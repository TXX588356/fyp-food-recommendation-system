import { RefreshCw, Soup, Sparkles, TriangleAlert, Utensils } from 'lucide-react'

type IconName = 'refresh' | 'bowl' | 'fork' | 'sparkle' | 'warning'

export default function RecommendationIcon({ name, size = 20 }: { name: IconName; size?: number }) {
  const commonProps = {
    size,
    strokeWidth: 2,
    'aria-hidden': true,
  }

  if (name === 'refresh') {
    return <RefreshCw {...commonProps} />
  }

  if (name === 'bowl') {
    return <Soup {...commonProps} />
  }

  if (name === 'fork') {
    return <Utensils {...commonProps} />
  }

  if (name === 'warning') {
    return <TriangleAlert {...commonProps} />
  }

  return <Sparkles {...commonProps} />
}
