import { Box, Group, Select, Text } from '@mantine/core'
import { MapPin } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { formatLocation, parseLocation } from '@/preferences/helpers'
import {
  parseMalaysiaCitiesCsv,
  uniqueSelectOptions,
  uniqueSelectOptionsByValue,
  type CityRow,
} from '@/preferences/options'
import type { LocationValue } from '@/preferences/types'
import type { RecommendationLocationOption } from '../recommendationTypes'

type DynamicLocationSelectProps = {
  selectedLocation: LocationValue
  savedLocations: RecommendationLocationOption[]
  onChange: (value: LocationValue) => void
}

export default function DynamicLocationSelect({
  selectedLocation,
  savedLocations,
  onChange,
}: DynamicLocationSelectProps) {
  const [cities, setCities] = useState<CityRow[]>([])
  const [states, setStates] = useState<RecommendationLocationOption[]>([])

  useEffect(() => {
    fetch('/malaysia_cities.csv')
      .then((response) => response.text())
      .then((csv) => {
        const rows = parseMalaysiaCitiesCsv(csv)
        setCities(rows)
        setStates(uniqueSelectOptions(rows.map((row) => row.subcountry)))
      })
      .catch((error) => {
        console.error('Error loading Malaysia cities CSV: ', error)
        setCities([])
        setStates([])
      })
  }, [])

  const districts = useMemo(() => {
    if (!selectedLocation.state) return []

    return uniqueSelectOptions(
      cities
        .filter((city) => city.subcountry === selectedLocation.state)
        .map((city) => city.name),
    )
  }, [cities, selectedLocation.state])

  const selectedLocationValue = formatLocation(selectedLocation)
  const savedLocationOptions = useMemo(
    () => uniqueSelectOptionsByValue(savedLocations),
    [savedLocations],
  )
  const selectedSavedLocationValue = savedLocationOptions.some((location) => location.value === selectedLocationValue)
    ? selectedLocationValue
    : null

  return (
    <Box className="ui-recommendation-location">
      <Group className="ui-recommendation-location-title" gap={6}>
        <MapPin size={15} strokeWidth={2.3} aria-hidden="true" />
        <Text>Recommendation Location</Text>
      </Group>

      <Group className="ui-recommendation-location-fields" align="flex-start">
        <Select
          label="Saved"
          placeholder="Use saved location"
          data={savedLocationOptions}
          value={selectedSavedLocationValue}
          size="xs"
          onChange={(value) => {
            if (!value) return
            onChange(parseLocation(value))
          }}
          classNames={{
            label: 'ui-input-label',
            input: 'ui-input',
          }}
        />

        <Select
          label="State"
          placeholder="Select state"
          data={states}
          value={selectedLocation.state || null}
          allowDeselect={false}
          size="xs"
          onChange={(state) => {
            if (!state || state === selectedLocation.state) return
            onChange({ state, district: '' })
          }}
          classNames={{
            label: 'ui-input-label',
            input: 'ui-input',
          }}
        />

        <Select
          label="District"
          placeholder="Select district"
          data={districts}
          value={selectedLocation.district || null}
          allowDeselect={false}
          disabled={!selectedLocation.state || districts.length === 0}
          size="xs"
          onChange={(district) => {
            if (!district || district === selectedLocation.district) return
            onChange({ ...selectedLocation, district: district ?? '' })
          }}
          classNames={{
            label: 'ui-input-label',
            input: 'ui-input',
          }}
        />
      </Group>
    </Box>
  )
}
