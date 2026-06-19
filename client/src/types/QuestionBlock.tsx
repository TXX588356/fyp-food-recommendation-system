import { Box, Title } from "@mantine/core"

export function QuestionBlock({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <Box className="ui-question">
      <Title order={2}>{title}</Title>
      {children}
    </Box>
  )
}
