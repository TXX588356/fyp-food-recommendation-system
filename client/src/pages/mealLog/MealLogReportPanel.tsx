import {
  Alert,
  Badge,
  Box,
  Button,
  Group,
  SimpleGrid,
  Stack,
  Text,
} from '@mantine/core'
import {
  Bar,
  BarChart,
  CartesianGrid,
  LabelList,
  Line,
  LineChart,
  ResponsiveContainer,
  Scatter,
  ScatterChart,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { formatKcal, formatRM } from './mealLogHelpers'
import type { MealLogReportResponse } from './mealLogTypes'

type MealLogReportPanelProps = {
  report: MealLogReportResponse
  isOpen: boolean
  onToggle: () => void
}

const formatCategory = (value: string) =>
  value
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')

const formatChartRM = (value: number) => `RM${value.toFixed(0)}`

const formatGram = (value: number) => `${value.toFixed(1)}g`

const formatShortNumber = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value.toFixed(0)}`
}

const buildBudgetOutlook = (summary: MealLogReportResponse['summary']) => {
  if (
    summary.monthlyMealBudget === undefined ||
    summary.projectedMonthSpend === undefined ||
    summary.budgetSpendStatus === undefined
  ) {
    return null
  }

  const projectedDifference = Math.abs(summary.monthlyMealBudget - summary.projectedMonthSpend)
  const isOverspending = summary.budgetSpendStatus === 'overspending'

  return {
    color: isOverspending ? 'red' : 'green',
    label: isOverspending ? 'Overspending' : 'On track',
    headline: isOverspending
      ? `${formatRM(projectedDifference)} over budget`
      : `${formatRM(projectedDifference)} under budget`,
    detail: `${formatRM(summary.projectedMonthSpend)} projected of ${formatRM(summary.monthlyMealBudget)} budget.`,
  }
}

const formatMinutesAsTime = (minutes: number) => {
  if (!Number.isFinite(minutes)) {
    return ''
  }

  const roundedMinutes = Math.round(minutes)
  const hour = Math.floor(roundedMinutes / 60)
  const minute = roundedMinutes % 60

  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

type MealTimingTooltipPayload = {
  payload?: {
    name: string
    count: number
    usualTime: string
  }
}

const MealTimingTooltip = ({
  active,
  payload,
}: {
  active?: boolean
  payload?: MealTimingTooltipPayload[]
}) => {
  if (!active || !payload || payload.length === 0) {
    return null
  }

  const item = payload[0]?.payload
  if (!item) {
    return null
  }

  return (
    <Box className="ui-card" p="sm">
      <Text size="sm" fw={900}>{item.name}</Text>
      <Text size="sm">Usual time: {item.usualTime}</Text>
      <Text size="xs" className="ui-field-copy">{item.count} logs</Text>
    </Box>
  )
}

const EmptyReportState = ({ message }: { message: string }) => (
  <Box className="ui-meal-log-report-empty-state">
    <Text size="sm">{message}</Text>
  </Box>
)

export default function MealLogReportPanel({
  report,
  isOpen,
  onToggle,
}: MealLogReportPanelProps) {
  const categoryChartData = report.categoryBreakdown.slice(0, 8).map((item) => ({
    name: formatCategory(item.category),
    count: item.count,
    percentage: item.percentage,
  }))

  const timeChartData = report.timePatterns
    .filter((item) => item.count > 0)
    .map((item) => ({
      name: formatCategory(item.window),
      count: item.count,
      usualTime: item.usualTime,
      usualTimeMinutes: item.usualTimeMinutes,
    }))

  const expensiveMealChartData = report.topExpensiveMeals.map((item) => ({
    name: item.mealName.length > 18 ? `${item.mealName.slice(0, 18)}...` : item.mealName,
    fullName: item.mealName,
    price: item.price,
    eatenAt: item.eatenAt,
  }))

  const dailyTrendData = report.dailyTrends.map((item) => ({
    day: item.dayLabel,
    meals: item.mealCount,
    spent: item.totalSpent,
    calories: item.totalCalories,
  }))

  const macroChartData = [
    {
      name: 'Protein',
      grams: report.macroSummary.totalProteinG,
      average: report.macroSummary.averageProteinG,
      share: report.macroSummary.proteinCalorieShare,
    },
    {
      name: 'Carbs',
      grams: report.macroSummary.totalCarbsG,
      average: report.macroSummary.averageCarbsG,
      share: report.macroSummary.carbsCalorieShare,
    },
    {
      name: 'Fat',
      grams: report.macroSummary.totalFatG,
      average: report.macroSummary.averageFatG,
      share: report.macroSummary.fatCalorieShare,
    },
  ]

  const hasMacroData = macroChartData.some((item) => item.grams > 0)

  const showBudgetCard = report.summary.remainingUsableBudget !== undefined
  const budgetOutlook = buildBudgetOutlook(report.summary)

  return (
    <Box className="ui-meal-log-report-panel">
      <Group justify="space-between" gap="md">
        <Box>
          <Text fw={900}>Monthly Report - {report.period.label}</Text>
          <Text size="sm" className="ui-field-copy">
            {report.summary.totalMeals} meals | {formatRM(report.summary.totalSpent)} spent
          </Text>
        </Box>

        <Button variant="subtle" className="ui-meal-log-button" onClick={onToggle}>
          {isOpen ? 'Collapse' : 'Expand'}
        </Button>
      </Group>

      {isOpen && (
        <Stack mt="md" gap="md">
          {report.lowDataWarning && (
            <Alert color="yellow">{report.lowDataWarning}</Alert>
          )}

          <SimpleGrid cols={{ base: 1, md: showBudgetCard ? 4 : 3 }} className="ui-meal-log-report-metrics">
            <Box>
              <Text>Average daily spend</Text>
              <strong>{formatRM(report.summary.averageDailySpend)}</strong>
              <Text size="xs" className="ui-field-copy">
                Based on active logging days.
              </Text>
            </Box>

            <Box>
              <Text>Average calories per meal</Text>
              <strong>{formatKcal(report.summary.averageCaloriesPerMeal)}</strong>
            </Box>

            <Box>
              <Text>Average price per meal</Text>
              <strong>{formatRM(report.summary.averagePricePerMeal)}</strong>
            </Box>

            {showBudgetCard && (
              <Box className={`ui-meal-log-budget-outlook ${budgetOutlook?.color === 'red' ? 'ui-meal-log-budget-outlook-risk' : ''}`}>
                <Group justify="space-between" gap="xs">
                  <Text>Budget outlook</Text>
                  {budgetOutlook && (
                    <Badge color={budgetOutlook.color} variant="light">
                      {budgetOutlook.label}
                    </Badge>
                  )}
                </Group>
                <strong>{budgetOutlook?.headline ?? formatRM(report.summary.remainingUsableBudget ?? 0)}</strong>
                <Text size="xs" className="ui-field-copy">
                  {budgetOutlook?.detail ?? 'Current month only.'}
                </Text>
              </Box>
            )}
          </SimpleGrid>

          <SimpleGrid cols={{ base: 1, md: 2 }} className="ui-meal-log-report-sections">
            <Box>
              <Text fw={900}>Category Breakdown</Text>
              {categoryChartData.length === 0 ? (
                <EmptyReportState message="No category tags are available for this month." />
              ) : (
                <Box className="ui-meal-log-report-chart">
                  <ResponsiveContainer width="100%" height={260}>
                    <BarChart data={categoryChartData} layout="vertical" margin={{ left: 12, right: 44 }}>
                      <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                      <XAxis type="number" allowDecimals={false} />
                      <YAxis type="category" dataKey="name" width={120} tick={{ fontSize: 12 }} />
                      <Tooltip
                        formatter={(value, _name, item) => [
                          `${value} meals (${Number(item.payload.percentage).toFixed(0)}%)`,
                          'Count',
                        ]}
                      />
                      <Bar dataKey="count" fill="#24533f" radius={[0, 6, 6, 0]}>
                        <LabelList dataKey="count" position="right" />
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </Box>
              )}
            </Box>

            <Box>
              <Text fw={900}>Usual Meal Timing</Text>
              {timeChartData.length === 0 ? (
                <Text size="sm" className="ui-field-copy">
                  No meal timing pattern available yet.
                </Text>
              ) : (
                <>
                  <Box className="ui-meal-log-report-chart">
                    <ResponsiveContainer width="100%" height={260}>
                      <ScatterChart margin={{ top: 24, right: 28, left: 8, bottom: 12 }}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis
                          type="number"
                          dataKey="usualTimeMinutes"
                          domain={[0, 1439]}
                          ticks={[360, 720, 1080, 1439]}
                          tickFormatter={(value) => formatMinutesAsTime(Number(value))}
                          name="Usual time"
                        />
                        <YAxis
                          type="category"
                          dataKey="name"
                          width={84}
                          tick={{ fontSize: 12 }}
                          name="Meal"
                        />
                        <Tooltip
                          cursor={{ strokeDasharray: '3 3' }}
                          content={<MealTimingTooltip />}
                        />
                        <Scatter data={timeChartData} fill="#e78b00">
                          <LabelList dataKey="usualTime" position="top" />
                        </Scatter>
                      </ScatterChart>
                    </ResponsiveContainer>
                  </Box>

                  <SimpleGrid cols={{ base: 2, md: 4 }} className="ui-meal-log-report-time-summary">
                    {timeChartData.map((item) => (
                      <Box key={item.name}>
                        <Text size="xs" className="ui-field-copy">{item.name}</Text>
                        <Text fw={900}>{item.usualTime}</Text>
                      </Box>
                    ))}
                  </SimpleGrid>
                </>
              )}
            </Box>
          </SimpleGrid>

          <SimpleGrid cols={{ base: 1, md: 2 }} className="ui-meal-log-report-sections">
            <Box>
              <Text fw={900}>Spending By Day</Text>
              {dailyTrendData.length === 0 ? (
                <EmptyReportState message="Log meals on at least one day to show spending trends." />
              ) : (
                <Box className="ui-meal-log-report-chart">
                  <ResponsiveContainer width="100%" height={260}>
                    <LineChart data={dailyTrendData} margin={{ top: 18, right: 24, left: 8, bottom: 10 }}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="day" tick={{ fontSize: 12 }} />
                      <YAxis tickFormatter={formatChartRM} width={54} />
                      <Tooltip formatter={(value) => [formatRM(Number(value)), 'Spent']} />
                      <Line
                        type="monotone"
                        dataKey="spent"
                        stroke="#24533f"
                        strokeWidth={3}
                        dot={{ r: 4 }}
                        activeDot={{ r: 6 }}
                        name="Spent"
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </Box>
              )}
            </Box>

            <Box>
              <Text fw={900}>Calories By Day</Text>
              {dailyTrendData.length === 0 ? (
                <EmptyReportState message="Log meals on at least one day to show calorie trends." />
              ) : (
                <Box className="ui-meal-log-report-chart">
                  <ResponsiveContainer width="100%" height={260}>
                    <LineChart data={dailyTrendData} margin={{ top: 18, right: 24, left: 8, bottom: 10 }}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="day" tick={{ fontSize: 12 }} />
                      <YAxis tickFormatter={(value) => formatShortNumber(Number(value))} width={54} />
                      <Tooltip formatter={(value) => [formatKcal(Number(value)), 'Calories']} />
                      <Line
                        type="monotone"
                        dataKey="calories"
                        stroke="#b45309"
                        strokeWidth={3}
                        dot={{ r: 4 }}
                        activeDot={{ r: 6 }}
                        name="Calories"
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </Box>
              )}
            </Box>
          </SimpleGrid>

          <SimpleGrid cols={{ base: 1, md: 2 }} className="ui-meal-log-report-sections">
            <Box>
              <Text fw={900}>Macro Balance</Text>
              {hasMacroData ? (
                <>
                  <Box className="ui-meal-log-report-chart">
                    <ResponsiveContainer width="100%" height={260}>
                      <BarChart data={macroChartData} margin={{ top: 24, right: 24, left: 8, bottom: 10 }}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="name" />
                        <YAxis tickFormatter={(value) => `${Number(value).toFixed(0)}g`} width={54} />
                        <Tooltip
                          formatter={(value, name, item) => {
                            if (name === 'grams') {
                              return [
                                `${formatGram(Number(value))} total, ${formatGram(item.payload.average)} avg/meal`,
                                'Macro',
                              ]
                            }

                            return [`${Number(value).toFixed(0)}%`, 'Calorie share']
                          }}
                        />
                        <Bar dataKey="grams" fill="#24533f" radius={[6, 6, 0, 0]}>
                          <LabelList
                            dataKey="share"
                            position="top"
                            formatter={(value) => `${Number(value).toFixed(0)}%`}
                          />
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  </Box>

                  <SimpleGrid cols={{ base: 3 }} className="ui-meal-log-report-time-summary">
                    {macroChartData.map((item) => (
                      <Box key={item.name}>
                        <Text size="xs" className="ui-field-copy">{item.name}</Text>
                        <Text fw={900}>{formatGram(item.average)}</Text>
                      </Box>
                    ))}
                  </SimpleGrid>
                </>
              ) : (
                <EmptyReportState message="No macro data available for this month." />
              )}
            </Box>

            <Box>
              <Text fw={900}>Macro Averages</Text>
              <SimpleGrid cols={{ base: 1 }} className="ui-meal-log-report-macro-average-list">
                {macroChartData.map((item) => (
                  <Group
                    className="ui-meal-log-report-list-row"
                    justify="space-between"
                    key={item.name}
                  >
                    <Text size="sm" fw={800}>{item.name}</Text>
                    <Text size="sm" fw={900}>{formatGram(item.average)} per meal</Text>
                  </Group>
                ))}
              </SimpleGrid>
            </Box>
          </SimpleGrid>

          <SimpleGrid cols={{ base: 1, md: 2 }} className="ui-meal-log-report-sections">
            <Box>
              <Text fw={900}>Top 5 Most Expensive Meals</Text>
              {expensiveMealChartData.length === 0 ? (
                <EmptyReportState message="No meal spending data available for this month." />
              ) : (
                <Box className="ui-meal-log-report-chart">
                  <ResponsiveContainer width="100%" height={260}>
                    <BarChart data={expensiveMealChartData} layout="vertical" margin={{ left: 12, right: 56 }}>
                      <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                      <XAxis type="number" tickFormatter={formatChartRM} />
                      <YAxis type="category" dataKey="name" width={120} tick={{ fontSize: 12 }} />
                      <Tooltip
                        formatter={(value) => [formatRM(Number(value)), 'Price']}
                        labelFormatter={(_, payload) => payload?.[0]?.payload?.fullName ?? ''}
                      />
                      <Bar dataKey="price" fill="#b45309" radius={[0, 6, 6, 0]}>
                        <LabelList
                          dataKey="price"
                          position="right"
                          formatter={(value) => formatRM(Number(value))}
                        />
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </Box>
              )}
            </Box>

            <Box>
              <Text fw={900}>Repeated Meals</Text>
              <Stack gap="xs">
                {report.topMeals.length === 0 ? (
                  <Text size="sm" className="ui-field-copy">
                    No repeated meals detected.
                  </Text>
                ) : (
                  report.topMeals.map((item) => (
                    <Group
                      className="ui-meal-log-report-list-row"
                      justify="space-between"
                      key={item.mealName}
                    >
                      <Text size="sm" fw={800}>{item.mealName}</Text>
                      <Text size="sm" fw={900}>{item.count} times</Text>
                    </Group>
                  ))
                )}
              </Stack>
            </Box>
          </SimpleGrid>

        </Stack>
      )}
    </Box>
  )
}
