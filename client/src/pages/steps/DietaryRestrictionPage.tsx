import { OptionButton } from "@/types/OptionButton"
import { QuestionBlock } from "@/types/QuestionBlock"
import { Box } from "@mantine/core"
import { dietaryRestrictionOptions } from "@/preferences/options"


type DietaryRestrictionStepProps = {
    value: string[]
    onChange: (value: string[]) => void
}

export function DietaryRestrictionStep({ value, onChange }: DietaryRestrictionStepProps) {
  const toggleRestriction = (restriction: string) => {
    if (restriction === 'none') {
      onChange(value.includes('none') ? [] : ['none'])
      return
    }

    const nextValue = value.includes(restriction)
    ? value.filter((item) => item !== restriction)
    : [...value.filter((item) => item !== 'none'), restriction]

    onChange(nextValue)
  }

  return (
    <QuestionBlock title="Do you have any dietary restrictions?">
      <Box className="ui-chip-grid">
        {dietaryRestrictionOptions.map((restriction) => (
          <OptionButton 
            key={restriction.value}
            label={restriction.label}
            selected={value.includes(restriction.value)}
            onClick={() => toggleRestriction(restriction.value)}
          />
        ))}
      </Box>
    </QuestionBlock>
  )
}
