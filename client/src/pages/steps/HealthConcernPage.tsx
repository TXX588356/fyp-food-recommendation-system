import { healthConcernOptions } from "@/preferences/options";
import { OptionButton } from "@/types/OptionButton";
import { QuestionBlock } from "@/types/QuestionBlock";
import { Box } from "@mantine/core";

type HealthConcernStepProps = {
    value: string[]
    onChange: (value: string[]) => void
}

export function HealthConcernStep({ value, onChange }: HealthConcernStepProps) {
  const toggleHealthConcern = (concern: string) => {
    if (concern === 'none') {
      onChange(value.includes('none') ? [] : ['none'])
      return
    }

    const nextValue = value.includes(concern)
    ? value.filter((item) => item !== concern)
    : [...value.filter((item) => item !== 'none'), concern]

    onChange(nextValue)
  }

  return (
    <QuestionBlock title="Do you want recommendations based on any condition or health concern?">
      <Box className="ui-stack ui-stack-compact">
        {healthConcernOptions.map((concern) => (
          <OptionButton 
            key={concern.value}
            label={concern.label}
            selected={value.includes(concern.value)}
            onClick={() => toggleHealthConcern(concern.value)}
            wide
          />
        ))}
      </Box>
    </QuestionBlock>
  )
}
