import { mainGoalOptions } from "@/preferences/options";
import { OptionButton } from "@/types/OptionButton";
import { QuestionBlock } from "@/types/QuestionBlock";
import { Box } from "@mantine/core";

type GoalStepProps = {
  value: string,
  onChange: (value: string) => void
}

export function GoalStep({ value, onChange }: GoalStepProps) {
  return (
    <QuestionBlock title="What is your main goal?">
      <Box className="ui-stack">
        {mainGoalOptions.map((goal) => (
          <OptionButton 
            key={goal.value}
            label={goal.label}
            selected={value === goal.value}
            onClick={() =>onChange(goal.value)}
          />
        ))}
      </Box>
    </QuestionBlock>
  )
}
