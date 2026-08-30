import { Plus, Search, Soup } from 'lucide-react'

type MealAddIconProps = {
  name: 'search' | 'plus' | 'bowl'
}

export default function MealAddIcon({ name }: MealAddIconProps) {
  const commonProps = {
    strokeWidth: 2,
    size: 22,
    'aria-hidden': true,
  }

  if (name === 'search') {
    return <Search {...commonProps} />
  }

  if (name === 'plus') {
    return <Plus {...commonProps} />
  }

  return <Soup {...commonProps} />
}
