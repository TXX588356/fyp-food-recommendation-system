import { Anchor, Card, Group, Badge, Image, Text } from "@mantine/core";

export default function RecommendationPage() {
   return (
    <Card shadow="sm" padding="lg" withBorder>
      <Card.Section>
        <Image
          src="https://raw.githubusercontent.com/mantinedev/mantine/master/.demo/images/bg-8.png"
          height={160}
          alt="Norway"
        />
      </Card.Section>

      <Group justify="space-between" mt="md" mb="xs">
        <Text fw={500}>Norway Fjord Adventures</Text>
        <Badge color="pink">On Sale</Badge>
      </Group>

      <Text size="lg" c="dimmed">
        To be implemented 👽
      </Text>
      <Anchor href="/preferences" mt="md">
        Manage preferences
      </Anchor>
    </Card>
  );
}
