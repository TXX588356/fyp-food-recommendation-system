//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	catalogservice "fyp/food-rs/internal/service/catalog"
	custommealservice "fyp/food-rs/internal/service/custommeal"
	"fyp/food-rs/internal/service/mealsearch"
	preferenceservice "fyp/food-rs/internal/service/preference"
	recommendationservice "fyp/food-rs/internal/service/recommendation"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

type fakeGeminiClient struct {
	meals              []interfaces.GeneratedMeal
	autocomplete       interfaces.CustomMealAutocompleteResponse
	matchDecisions     []interfaces.MealMatchDecision
	generateInputs     []interfaces.MealPromptInput
	matchTasks         []interfaces.MealMatchTask
	autocompleteInputs []interfaces.CustomMealAutocompleteInput
}

func (f *fakeGeminiClient) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	f.generateInputs = append(f.generateInputs, input)
	return interfaces.GeminiMealsResponse{Meals: f.meals}, nil
}

func (f *fakeGeminiClient) ResolveMatches(ctx context.Context, tasks []interfaces.MealMatchTask) ([]interfaces.MealMatchDecision, error) {
	f.matchTasks = append(f.matchTasks, tasks...)
	return f.matchDecisions, nil
}

func (f *fakeGeminiClient) AutocompleteCustomMeal(ctx context.Context, input interfaces.CustomMealAutocompleteInput) (interfaces.CustomMealAutocompleteResponse, error) {
	f.autocompleteInputs = append(f.autocompleteInputs, input)
	return f.autocomplete, nil
}

func (f *fakeGeminiClient) ExplainMealRecommendation(ctx context.Context, input interfaces.MealDetailExplanationInput) (string, error) {
	return "fake explanation", nil
}

type fakeRestaurantSearcher struct{}

func (fakeRestaurantSearcher) SearchRestaurants(ctx context.Context, input interfaces.RestaurantSearchInput) (interfaces.RestaurantSearchResult, error) {
	return interfaces.RestaurantSearchResult{
		Status:      interfaces.RestaurantLookupUnavailable,
		Restaurants: nil,
	}, nil
}

type recommendationIntegrationResolver struct{}

func (recommendationIntegrationResolver) Resolve(objectKey string) string {
	return objectKey
}

var _ = Describe("Recommendation integration", func() {
	floatPtr := func(value float64) *float64 {
		return &value
	}

	createTestUser := func(tx *gorm.DB) model.User {
		user := model.User{
			Name:         "Recommendation User",
			Email:        fmt.Sprintf("integration-%s@gmail.com", uuid.NewString()),
			PasswordHash: "hashed-password",
		}

		Expect(tx.Create(&user).Error).NotTo(HaveOccurred())
		Expect(user.ID).NotTo(BeZero())

		return user
	}

	completePreferences := func(tx *gorm.DB, userID uuid.UUID, input interfaces.PreferenceInput) {
		ctx := GinkgoT().Context()

		preferenceSvc := preferenceservice.NewService(
			postgres.NewPreferencePostgresRepository(tx),
			postgres.NewUserPostgresRepository(tx),
		)

		_, err := preferenceSvc.CompleteOnboarding(ctx, userID, input)
		Expect(err).NotTo(HaveOccurred())
	}

	defaultPreferences := func() interfaces.PreferenceInput {
		consent := true

		return interfaces.PreferenceInput{
			MainGoal:            "eat_healthier",
			MonthlyMealBudget:   600,
			DataSharingConsent:  &consent,
			HomeLocation:        "Bukit Jalil",
			WorkSchoolLocation:  "KL Sentral",
			HealthConcerns:      []string{"high_blood_pressure"},
			DietaryRestrictions: []string{"halal"},
			PreferredMealTags:   []string{"malaysian", "rice_dishes"},
		}
	}

	createCatalogMeal := func(tx *gorm.DB, name string, tags []string, calories, protein, carbs, fat float64) model.PrebuiltMeal {
		meal := model.PrebuiltMeal{
			SourceCode:         "integration",
			SourceRecordID:     uuid.NewString(),
			Name:               name,
			CategoryCodes:      model.StringArray(tags),
			ServingDescription: "1 serving",
			Calories:           floatPtr(calories),
			ProteinG:           floatPtr(protein),
			CarbsG:             floatPtr(carbs),
			FatG:               floatPtr(fat),
		}

		Expect(tx.Create(&meal).Error).NotTo(HaveOccurred())
		Expect(meal.ID).NotTo(BeZero())

		return meal
	}

	createRecentMealLog := func(tx *gorm.DB, userID uuid.UUID, meal model.PrebuiltMeal, eatenAt time.Time) {
		log := model.MealLog{
			UserID:         userID,
			PrebuiltMealID: &meal.ID,
			MealName:       meal.Name,
			Price:          10,
			EatenAt:        eatenAt,
			MealType:       "lunch",
			MealCategory:   meal.CategoryCodes,
			Calories:       500,
			ProteinG:       20,
			CarbsG:         60,
			FatG:           15,
		}

		Expect(tx.Create(&log).Error).NotTo(HaveOccurred())
	}

	newRecommendationService := func(tx *gorm.DB, fakeGemini *fakeGeminiClient) interfaces.RecommendationService {
		customMealSvc := custommealservice.NewService(
			postgres.NewCustomMealPostgresRepository(tx),
		)

		catalogSvc := catalogservice.NewService(
			postgres.NewCatalogPostgresRepository(tx),
			recommendationIntegrationResolver{},
		)

		catalogCandidateSearcher := catalogservice.NewCandidateSearcher(catalogSvc)
		candidateSearcher := mealsearch.NewCandidateSearcher(customMealSvc, catalogCandidateSearcher)

		preferenceSvc := preferenceservice.NewService(
			postgres.NewPreferencePostgresRepository(tx),
			postgres.NewUserPostgresRepository(tx),
		)

		return recommendationservice.NewService(
			fakeGemini,
			candidateSearcher,
			fakeGemini,
			catalogSvc,
			fakeGemini,
			postgres.NewMealLogPostgresRepository(tx),
			postgres.NewCustomMealPostgresRepository(tx),
			preferenceSvc,
			fakeGemini,
			fakeRestaurantSearcher{},
		)
	}

	generatedMeal := func(name string, minPrice, maxPrice float64, flags map[string]string) interfaces.GeneratedMeal {
		return interfaces.GeneratedMeal{
			Name: name,
			EstimatedPriceRange: interfaces.PriceRange{
				Min: minPrice,
				Max: maxPrice,
			},
			SodiumLevel: "LOW",
			SugarLevel:  "LOW",
			PurineRisk:  "LOW",
			HealthFlags: flags,
		}
	}

	Describe("Generate recommendations using user preferences", func() {
		It("should retrieve user preferences and uses them when scoring candidates", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)

			prefs := defaultPreferences()
			prefs.PreferredMealTags = []string{"malaysian", "rice_dishes"}
			completePreferences(tx, user.ID, prefs)

			createCatalogMeal(tx, "Nasi Lemak", []string{"malaysian", "rice_dishes"}, 550, 20, 65, 22)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Nasi Lemak", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:      "lunch",
				CurrentMonthSpent: 100,
				PerMealBudget:     12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(1))
			Expect(result.Candidates[0].Food.Name).To(Equal("Nasi Lemak"))
			Expect(result.Candidates[0].ScoreBreakdown.Preference).To(Equal(30.0))
			Expect(fakeGemini.generateInputs).To(HaveLen(1))
			Expect(fakeGemini.generateInputs[0].Goal).To(Equal("eat_healthier"))
			Expect(fakeGemini.generateInputs[0].DietaryRestrictions).To(ConsistOf("halal"))
			Expect(fakeGemini.generateInputs[0].HealthConcerns).To(ConsistOf("high_blood_pressure"))
			Expect(fakeGemini.generateInputs[0].PreferredMealTags).To(ConsistOf("malaysian", "rice_dishes"))
		})
	})

	Describe("Use recent meal history during recommendation", func() {
		It("should apply a recency penalty to a recently consumed meal", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			completePreferences(tx, user.ID, defaultPreferences())

			nasi := createCatalogMeal(tx, "Nasi Lemak", []string{"malaysian", "rice_dishes"}, 550, 20, 65, 22)
			createCatalogMeal(tx, "Chicken Porridge", []string{"malaysian", "porridge"}, 400, 25, 45, 8)
			createRecentMealLog(tx, user.ID, nasi, time.Now().AddDate(0, 0, -1))

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Nasi Lemak", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
					generatedMeal("Chicken Porridge", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(2))

			var nasiCandidate interfaces.MatchedMealCandidate
			var porridgeCandidate interfaces.MatchedMealCandidate

			for _, candidate := range result.Candidates {
				switch candidate.Food.Name {
				case "Nasi Lemak":
					nasiCandidate = candidate
				case "Chicken Porridge":
					porridgeCandidate = candidate
				}
			}

			Expect(nasiCandidate.Food.Name).To(Equal("Nasi Lemak"))
			Expect(porridgeCandidate.Food.Name).To(Equal("Chicken Porridge"))
			Expect(nasiCandidate.ScoreBreakdown.RecencyPenalty).
				To(BeNumerically("<", porridgeCandidate.ScoreBreakdown.RecencyPenalty))
		})
	})

	Describe("Resolve exact generated meal match from catalog", func() {
		It("should select the exact catalog record without calling the Gemini matcher", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			completePreferences(tx, user.ID, defaultPreferences())

			createCatalogMeal(tx, "Nasi Lemak", []string{"malaysian", "rice_dishes"}, 550, 20, 65, 22)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Nasi Lemak", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(1))
			Expect(result.Candidates[0].Food.Name).To(Equal("Nasi Lemak"))
			Expect(fakeGemini.matchTasks).To(BeEmpty())
		})
	})

	Describe("Resolve ambiguous candidate using Gemini matcher", func() {
		It("should use the fake Gemini matching decision to select the correct candidate", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			completePreferences(tx, user.ID, defaultPreferences())

			createCatalogMeal(tx, "Chicken Noodle Soup", []string{"chinese", "noodle_dishes"}, 430, 25, 50, 10)
			selected := createCatalogMeal(tx, "Chicken Fried Noodles", []string{"chinese", "noodle_dishes"}, 420, 28, 73, 20)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Chicken Noodles", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
				matchDecisions: []interfaces.MealMatchDecision{
					{
						MealIndex:   0,
						Decision:    "MATCH",
						CandidateID: selected.ID.String(),
					},
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(fakeGemini.matchTasks).NotTo(BeEmpty())
			Expect(result.Candidates).To(HaveLen(1))
			Expect(result.Candidates[0].Food.ID).To(Equal(selected.ID.String()))
			Expect(result.Candidates[0].Food.Name).To(Equal("Chicken Fried Noodles"))

		})
	})

	Describe("Create missing generated meal in catalog", func() {
		It("should create the missing generated meal and adds it to the candidate list", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			completePreferences(tx, user.ID, defaultPreferences())

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Generated Tempeh Bowl", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
				autocomplete: interfaces.CustomMealAutocompleteResponse{
					Calories:         480,
					ProteinG:         30,
					CarbsG:           55,
					FatG:             12,
					MealCategoryTags: []string{"indonesian", "tofu_soy"},
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(1))
			Expect(result.Candidates[0].Food.Name).To(Equal("Generated Tempeh Bowl"))
			Expect(result.Candidates[0].Food.Source).To(Equal("prebuilt"))
			Expect(fakeGemini.autocompleteInputs).To(HaveLen(1))
			Expect(fakeGemini.autocompleteInputs[0].Name).To(Equal("Generated Tempeh Bowl"))

			var stored model.PrebuiltMeal
			Expect(tx.Where("name = ?", "Generated Tempeh Bowl").First(&stored).Error).
				NotTo(HaveOccurred())
			Expect(stored.CategoryCodes).To(ConsistOf("indonesian", "tofu_soy"))
		})
	})

	Describe("Filter meal that violates dietary restriction", func() {
		It("should remove beef candidates for a non-beef user", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)

			prefs := defaultPreferences()
			prefs.DietaryRestrictions = []string{"non_beef"}
			prefs.PreferredMealTags = []string{"malaysian"}
			completePreferences(tx, user.ID, prefs)

			createCatalogMeal(tx, "Beef Rendang", []string{"malaysian", "beef"}, 650, 35, 30, 35)
			createCatalogMeal(tx, "Chicken Rice", []string{"malaysian", "poultry"}, 520, 28, 62, 14)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Beef Rendang", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
					generatedMeal("Chicken Rice", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "dinner",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(1))
			Expect(result.Candidates[0].Food.Name).To(Equal("Chicken Rice"))
			Expect(result.FilteredOut).To(HaveLen(1))
			Expect(result.FilteredOut[0].Candidate.Food.Name).To(Equal("Beef Rendang"))
			Expect(result.FilteredOut[0].Reason).To(ContainSubstring("beef"))
		})
	})

	Describe("Filter candidate marked AVOID for health concern", func() {
		It("should remove unsafe candidates from the final list", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			completePreferences(tx, user.ID, defaultPreferences())

			createCatalogMeal(tx, "Salty Soup", []string{"soups"}, 350, 18, 40, 8)
			createCatalogMeal(tx, "Grilled Chicken", []string{"poultry"}, 430, 35, 20, 12)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Salty Soup", 8, 10, map[string]string{"high_blood_pressure": "AVOID"}),
					generatedMeal("Grilled Chicken", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "dinner",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(1))
			Expect(result.Candidates[0].Food.Name).To(Equal("Grilled Chicken"))
			Expect(result.FilteredOut).To(HaveLen(1))
			Expect(result.FilteredOut[0].Reason).To(Equal("AVOID for high_blood_pressure"))
		})
	})

	Describe("Apply budget fit during ranking", func() {
		It("should rank the within-budget candidate higher", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)

			prefs := defaultPreferences()
			prefs.PreferredMealTags = []string{"rice_dishes"}
			completePreferences(tx, user.ID, prefs)

			createCatalogMeal(tx, "Budget Rice Bowl", []string{"rice_dishes"}, 500, 25, 60, 12)
			createCatalogMeal(tx, "Premium Rice Bowl", []string{"rice_dishes"}, 500, 25, 60, 12)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Budget Rice Bowl", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
					generatedMeal("Premium Rice Bowl", 20, 24, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 10,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(2))
			Expect(result.Candidates[0].Food.Name).To(Equal("Budget Rice Bowl"))
			Expect(result.Candidates[0].ScoreBreakdown.BudgetFit).
				To(BeNumerically(">", result.Candidates[1].ScoreBreakdown.BudgetFit))
		})
	})

	Describe("Apply preferred meal tags during ranking", func() {
		It("should give a higher preference score to candidates matching preferred tags", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)

			prefs := defaultPreferences()
			prefs.PreferredMealTags = []string{"porridge"}
			completePreferences(tx, user.ID, prefs)

			createCatalogMeal(tx, "Chicken Porridge", []string{"porridge"}, 420, 26, 48, 8)
			createCatalogMeal(tx, "Chicken Soup", []string{"soups"}, 420, 26, 48, 8)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Chicken Porridge", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
					generatedMeal("Chicken Soup", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(2))
			Expect(result.Candidates[0].Food.Name).To(Equal("Chicken Porridge"))
			Expect(result.Candidates[0].ScoreBreakdown.Preference).
				To(BeNumerically(">", result.Candidates[1].ScoreBreakdown.Preference))
		})
	})

	Describe("Rank final candidates by total score", func() {
		It("should order final recommendations from highest to lowest score", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)

			prefs := defaultPreferences()
			prefs.MainGoal = "quick_recommendation"
			prefs.PreferredMealTags = []string{"porridge"}
			completePreferences(tx, user.ID, prefs)

			createCatalogMeal(tx, "Best Porridge", []string{"porridge"}, 420, 25, 45, 8)
			createCatalogMeal(tx, "Okay Soup", []string{"soups"}, 420, 25, 45, 8)
			createCatalogMeal(tx, "Expensive Soup", []string{"soups"}, 420, 25, 45, 8)

			fakeGemini := &fakeGeminiClient{
				meals: []interfaces.GeneratedMeal{
					generatedMeal("Expensive Soup", 25, 30, map[string]string{"high_blood_pressure": "SAFE"}),
					generatedMeal("Okay Soup", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
					generatedMeal("Best Porridge", 8, 10, map[string]string{"high_blood_pressure": "SAFE"}),
				},
			}
			svc := newRecommendationService(tx, fakeGemini)

			result, err := svc.GenerateRecommendationResult(ctx, user.ID, interfaces.RecommendationRequestInput{
				MealCategory:  "lunch",
				PerMealBudget: 12,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Candidates).To(HaveLen(3))

			Expect(result.Candidates[0].Score).
				To(BeNumerically(">=", result.Candidates[1].Score))
			Expect(result.Candidates[1].Score).
				To(BeNumerically(">=", result.Candidates[2].Score))
			Expect(result.Candidates[0].Food.Name).To(Equal("Best Porridge"))
		})
	})
})
