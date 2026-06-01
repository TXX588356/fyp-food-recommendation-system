import { parseMalaysiaCitiesCsv, type CityRow } from "@/preferences/options"
import type { LocationValue } from "@/preferences/types"
import { QuestionBlock } from "@/types/QuestionBlock"
import { Box, Group, NumberInput, Select, Text } from "@mantine/core"
import { useEffect, useMemo, useState } from "react"

function LocationSelectGroup({
  label,
  description,
  value,
  onChange,
}: {
  label: string
  description: string
  value: LocationValue
  onChange: (value: LocationValue) => void
}) {
  const [cities, setCities] = useState<CityRow[]>([])
  const [states, setStates] = useState<{ value: string; label: string }[]>([])

  useEffect(() => {
    fetch("/malaysia_cities.csv")
      .then((response) => response.text())
      .then((csv) => {
        const rows = parseMalaysiaCitiesCsv(csv)
        setCities(rows)

        const nextStates = Array.from(
          new Set(rows.map((row) => row.subcountry)),
        )
          .sort()
          .map((state) => ({
            value: state,
            label: state,
          }))

        setStates(nextStates)
      })
      .catch((error) => {
        console.error("Error loading Malaysia cities CSV: ", error)
        setStates([])
        setCities([])
      })
  }, [])

  const districts = useMemo(() => {
    if (!value.state) return []

    return cities
      .filter((city) => city.subcountry === value.state)
      .map((city) => city.name)
      .sort()
      .map((city) => ({
        value: city,
        label: city,
      }))
  }, [value.state, cities])

  return (
    <Box className="ui-field-group">
      <Text className="ui-field-title">{label}</Text>
      <Text className="ui-field-copy">{description}</Text>

      <Group grow align="flex-start">
        <Select
          label="State"
          placeholder="Select state"
          data={states}
          value={value.state || null}
          allowDeselect={false}
          onChange={(nextState) => {
            if (!nextState || nextState === value.state) return

            onChange({
              state: nextState,
              district: "",
            })
          }}
          classNames={{
            label: "ui-input-label",
            input: "ui-input",
          }}
        />

        <Select
          label="District"
          placeholder="Select district"
          data={districts}
          value={value.district || null}
          allowDeselect={false}
          disabled={!value.state || districts.length === 0}
          onChange={(district) =>
            onChange({
              ...value,
              district: district ?? "",
            })
          }
          classNames={{
            label: "ui-input-label",
            input: "ui-input",
          }}
        />
      </Group>
    </Box>
  )
}

type BudgetLocationStepProps = {
  monthlyMealBudget: number | string
  workSchoolLocation: LocationValue
  homeLocation: LocationValue
  onMonthlyMealBudgetChange: (value: number | string) => void
  onWorkSchoolLocationChange: (value: LocationValue) => void
  onHomeLocationChange: (value: LocationValue) => void
}

export function BudgetLocationStep({
  monthlyMealBudget,
  workSchoolLocation,
  homeLocation,
  onMonthlyMealBudgetChange,
  onWorkSchoolLocationChange,
  onHomeLocationChange,
}: BudgetLocationStepProps) {
  return (
    <QuestionBlock title="Tell Us Your Budget and Location">
      <Box className="ui-form-grid">
        <NumberInput
          label="Monthly meal budget"
          prefix="RM "
          min={1}
          clampBehavior="none"
          hideControls
          value={monthlyMealBudget}
          onChange={onMonthlyMealBudgetChange}
          classNames={{
            label: "ui-input-label",
            input: "ui-input",
          }}
        />

        <LocationSelectGroup
          label="Workplace / school location"
          description="Where do you usually need lunch or dinner recommendations?"
          value={workSchoolLocation}
          onChange={onWorkSchoolLocationChange}
        />

        <LocationSelectGroup
          label="Home location"
          description="Where should evening and weekend recommendations be centered?"
          value={homeLocation}
          onChange={onHomeLocationChange}
        />
      </Box>
    </QuestionBlock>
  )
}