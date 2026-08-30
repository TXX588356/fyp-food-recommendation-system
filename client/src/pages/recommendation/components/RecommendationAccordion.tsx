import {
  Accordion,
  Alert,
  Badge,
  Box,
  Button,
  SegmentedControl,
  Text,
  Title,
} from '@mantine/core'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import { formatLocation } from '@/preferences/helpers'
import type { LocationValue } from '@/preferences/types'
import type {
  CategoryState,
  FilteredMealCandidate,
  MealCategory,
  MatchedMealCandidate,
  RecommendationTourStep,
  VisibleRecommendationItem,
} from '../recommendationTypes'
import RecommendationIcon from './RecommendationIcon'

type MealCategoryOption = {
  value: MealCategory
  label: string
}

type RecommendationAccordionProps = {
  activeCategory: MealCategory | null
  mealCategoryOptions: MealCategoryOption[]
  candidatesByCategory: CategoryState<MatchedMealCandidate[]>
  filteredOutByCategory: CategoryState<FilteredMealCandidate[]>
  loadingByCategory: CategoryState<boolean>
  generatedByCategory: CategoryState<boolean>
  errorsByCategory: CategoryState<string | null>
  displayMode: 'filtered' | 'all'
  selectedLocation: LocationValue
  activeTourStep: RecommendationTourStep | null
  onCategoryChange: (value: string | null) => void
  onDisplayModeChange: (value: 'filtered' | 'all') => void
  onGenerateRecommendations: (mealCategory: MealCategory, force?: boolean) => void
  renderCandidateCard: (
    mealCategory: MealCategory,
    item: VisibleRecommendationItem,
    index: number,
  ) => ReactNode
}

export default function RecommendationAccordion({
  activeCategory,
  mealCategoryOptions,
  candidatesByCategory,
  filteredOutByCategory,
  loadingByCategory,
  generatedByCategory,
  errorsByCategory,
  displayMode,
  selectedLocation,
  activeTourStep,
  onCategoryChange,
  onDisplayModeChange,
  onGenerateRecommendations,
  renderCandidateCard,
}: RecommendationAccordionProps) {
  return (
    <Accordion
      value={activeCategory}
      onChange={onCategoryChange}
      className="ui-meal-accordion"
      chevronPosition="right"
    >
      {mealCategoryOptions.map((option) => {
        const candidates = candidatesByCategory[option.value]
        const filteredOut = filteredOutByCategory[option.value]
        const isLoading = loadingByCategory[option.value]
        const hasGenerated = generatedByCategory[option.value]
        const error = errorsByCategory[option.value]
        const loadingText = `Generating and sorting ${option.label.toLowerCase()} meals...`
        const visibleItems =
          displayMode === 'filtered'
            ? candidates.map((candidate) => ({
                candidate,
                filteredReason: null,
              }))
            : [
                ...candidates.map((candidate) => ({
                  candidate,
                  filteredReason: null,
                })),
                ...filteredOut.map((item) => ({
                  candidate: item.candidate,
                  filteredReason: item.reason,
                })),
              ]

        return (
          <Accordion.Item value={option.value} key={option.value} className="ui-meal-accordion-item">
            <Accordion.Control className="ui-meal-accordion-control">
              <Box className="ui-meal-accordion-heading">
                <Box className="ui-meal-accordion-icon">
                  <RecommendationIcon name="fork" size={20} />
                </Box>
                <Title order={2}>{option.label}</Title>
                <Badge className="ui-meal-accordion-count">
                  {isLoading ? 'Generating' : hasGenerated ? `${candidates.length} matches` : 'Folded'}
                </Badge>
              </Box>
            </Accordion.Control>

            <Accordion.Panel className="ui-meal-accordion-panel">
              {error && (
                <Alert color="red" icon={<RecommendationIcon name="warning" size={20} />}>
                  {error}
                </Alert>
              )}

              <Box className="ui-recommendation-filter-row">
                <SegmentedControl
                  className="ui-recommendation-filter-toggle"
                  value={displayMode}
                  onChange={(value) => onDisplayModeChange(value as 'filtered' | 'all')}
                  data={[
                    { label: 'Show recommended only', value: 'filtered' },
                    { label: 'Show all results', value: 'all' },
                  ]}
                />
              </Box>

              {isLoading && (
                <Box className="ui-recommendation-state ui-card">
                  <RecommendationIcon name="sparkle" size={28} />
                  <Text fw={900}>
                    <span className="t-shimmer" data-text={loadingText}>
                      {loadingText}
                    </span>
                  </Text>
                </Box>
              )}

              {!isLoading && hasGenerated && visibleItems.length === 0 && (
                <Box className="ui-recommendation-state ui-card">
                  <RecommendationIcon name="bowl" size={30} />
                  <Text fw={900}>No dataset matches found.</Text>
                  <Text className="ui-field-copy">
                    Try regenerating or add your custom meal.
                  </Text>
                </Box>
              )}

              {!isLoading && !hasGenerated && (
                <Box className="ui-recommendation-state ui-card">
                  <RecommendationIcon name="bowl" size={30} />
                  <Text fw={900}>Preparing this mealtime.</Text>
                </Box>
              )}

              {!isLoading && visibleItems.length > 0 && (
                <Box className="ui-recommendation-grid">
                  {visibleItems.map((item, index) => renderCandidateCard(option.value, item, index))}
                </Box>
              )}

              <Button
                className={`ui-primary-button ui-recommendation-regenerate ${
                  activeTourStep === 'regenerate' && option.value === activeCategory ? 'ui-spotlight-target' : ''
                }`}
                data-recommendation-tour={
                  activeTourStep === 'regenerate' && option.value === activeCategory ? 'regenerate' : undefined
                }
                leftSection={<RecommendationIcon name="refresh" size={18} />}
                onClick={() => onGenerateRecommendations(option.value, true)}
                loading={isLoading}
              >
                Regenerate {option.label.toLowerCase()}
              </Button>

              <Button
                component={Link}
                to={`/meals/add/${option.value}?location=${encodeURIComponent(formatLocation(selectedLocation))}`}
                className={`ui-ghost-button ui-recommendation-add-meal ${
                  activeTourStep === 'addMeal' && option.value === activeCategory ? 'ui-spotlight-target' : ''
                }`}
                data-recommendation-tour={
                  activeTourStep === 'addMeal' && option.value === activeCategory ? 'add-meal' : undefined
                }
                variant="subtle"
              >
                Add other meal to {option.label.toLowerCase()}
              </Button>
            </Accordion.Panel>
          </Accordion.Item>
        )
      })}
    </Accordion>
  )
}
