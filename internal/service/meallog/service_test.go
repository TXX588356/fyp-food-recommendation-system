package meallog

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// TestMealLogService runs the meal-log service test suite.
func TestMealLogService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Meal Log Service Suite")
}

type testMealLogRepository struct {
	logs map[uuid.UUID]*model.MealLog

	createdLog *model.MealLog
	createErr  error

	listLogs []model.MealLog
	listErr  error
	listFrom time.Time
	listTo   time.Time

	findID     uuid.UUID
	findUserID uuid.UUID
	findErr    error

	updatedLog *model.MealLog
	updateErr  error

	deletedID     uuid.UUID
	deletedUserID uuid.UUID
	deleteErr     error
}

func (r *testMealLogRepository) Create(_ context.Context, mealLog *model.MealLog) (*model.MealLog, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}

	if mealLog.ID == uuid.Nil {
		mealLog.ID = uuid.New()
	}

	r.createdLog = mealLog
	r.logs[mealLog.ID] = mealLog

	return mealLog, nil
}

func (r *testMealLogRepository) ListByUserAndRange(context.Context, uuid.UUID, time.Time, time.Time) ([]model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) ListByUserAndMonth(_ context.Context, _ uuid.UUID, start time.Time, end time.Time) ([]model.MealLog, error) {
	r.listFrom = start
	r.listTo = end
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listLogs, nil
}

func (r *testMealLogRepository) FindByIDAndUser(_ context.Context, id uuid.UUID, userID uuid.UUID) (*model.MealLog, error) {
	r.findID = id
	r.findUserID = userID

	if r.findErr != nil {
		return nil, r.findErr
	}

	log, ok := r.logs[id]
	if !ok || log.UserID != userID {
		return nil, errors.New("meal log not found")
	}

	return log, nil
}

func (r *testMealLogRepository) Update(_ context.Context, mealLog *model.MealLog) (*model.MealLog, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}

	r.updatedLog = mealLog
	r.logs[mealLog.ID] = mealLog

	return mealLog, nil
}

func (r *testMealLogRepository) Delete(_ context.Context, id uuid.UUID, userID uuid.UUID) error {
	r.deletedID = id
	r.deletedUserID = userID

	if r.deleteErr != nil {
		return r.deleteErr
	}

	delete(r.logs, id)

	return nil
}

var _ = Describe("Meal log service", func() {
	var (
		ctx    context.Context
		repo   *testMealLogRepository
		svc    interfaces.MealLogService
		userID uuid.UUID
		logID  uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		userID = uuid.New()
		logID = uuid.New()
		repo = &testMealLogRepository{
			logs: map[uuid.UUID]*model.MealLog{},
		}
		svc = NewService(repo, nil, nil, nil)
		svc.(*service).now = func() time.Time {
			return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
		}
	})

	storedLog := func() *model.MealLog {
		customMealID := uuid.New()

		return &model.MealLog{
			ID:               logID,
			UserID:           userID,
			CustomMealItemID: &customMealID,
			MealName:         "Chicken Rice",
			Calories:         650,
			ProteinG:         32,
			CarbsG:           78,
			FatG:             21,
			Price:            8.50,
			EatenAt:          time.Date(2026, 7, 12, 8, 30, 0, 0, time.UTC),
			MealType:         "breakfast",
			MealCategory:     model.StringArray{"rice_dishes", "poultry"},
		}
	}

	Describe("Create", func() {
		floatPtr := func(value float64) *float64 {
			return &value
		}

		It("should store and return the selected meal type for a custom meal log", func() {
			customMealID := uuid.New()
			eatenAt := time.Date(2026, 7, 12, 11, 15, 0, 0, time.UTC)
			customMealService := mocks.NewCustomMealService(GinkgoT())

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, customMealID).
				Return(&interfaces.CustomMealResponse{
					ID:               customMealID.String(),
					Name:             "Chicken Rice",
					Calories:         650,
					ProteinG:         32,
					CarbsG:           78,
					FatG:             21,
					MealCategoryTags: []string{"rice_dishes", "poultry"},
				}, nil)

			svc = NewService(repo, customMealService, nil, nil)

			response, err := svc.Create(ctx, userID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourceCustom,
				MealID:   customMealID.String(),
				Price:    8.50,
				EatenAt:  eatenAt,
				MealType: "breakfast",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.MealType).To(Equal("breakfast"))
			Expect(response.MealName).To(Equal("Chicken Rice"))
			Expect(response.Calories).To(Equal(650.0))
			Expect(response.ProteinG).To(Equal(32.0))
			Expect(response.CarbsG).To(Equal(78.0))
			Expect(response.FatG).To(Equal(21.0))
			Expect(response.MealCategory).To(Equal([]string{"rice_dishes", "poultry"}))

			Expect(repo.createdLog).NotTo(BeNil())
			Expect(repo.createdLog.UserID).To(Equal(userID))
			Expect(repo.createdLog.CustomMealItemID).NotTo(BeNil())
			Expect(*repo.createdLog.CustomMealItemID).To(Equal(customMealID))
			Expect(repo.createdLog.MealType).To(Equal("breakfast"))
			Expect(repo.createdLog.ProteinG).To(Equal(32.0))
			Expect(repo.createdLog.CarbsG).To(Equal(78.0))
			Expect(repo.createdLog.FatG).To(Equal(21.0))
			Expect(repo.createdLog.EatenAt).To(Equal(eatenAt))
			Expect(repo.createdLog.Price).To(Equal(8.50))
		})

		It("should store and return a prebuilt meal log", func() {
			prebuiltMealID := uuid.New()
			eatenAt := time.Date(2026, 7, 12, 19, 15, 0, 0, time.UTC)
			catalogService := mocks.NewCatalogService(GinkgoT())

			catalogService.EXPECT().
				GetMeal(mock.Anything, prebuiltMealID).
				Return(interfaces.CatalogMeal{
					ID:         prebuiltMealID,
					Name:       "Nasi Lemak",
					Categories: []string{"rice_dishes", "fried_foods"},
					SelectedNutrition: interfaces.CatalogNutrition{
						Calories: floatPtr(720),
						ProteinG: floatPtr(24),
						CarbsG:   floatPtr(86),
						FatG:     floatPtr(32),
					},
				}, nil)

			svc = NewService(repo, nil, catalogService, nil)

			response, err := svc.Create(ctx, userID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMealID.String(),
				Price:    12.50,
				EatenAt:  eatenAt,
				MealType: "dinner",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.PrebuiltMealID).NotTo(BeNil())
			Expect(*response.PrebuiltMealID).To(Equal(prebuiltMealID.String()))
			Expect(response.MealName).To(Equal("Nasi Lemak"))
			Expect(response.Calories).To(Equal(720.0))
			Expect(response.ProteinG).To(Equal(24.0))
			Expect(response.CarbsG).To(Equal(86.0))
			Expect(response.FatG).To(Equal(32.0))
			Expect(response.MealCategory).To(Equal([]string{"rice_dishes", "fried_foods"}))

			Expect(repo.createdLog).NotTo(BeNil())
			Expect(repo.createdLog.PrebuiltMealID).NotTo(BeNil())
			Expect(*repo.createdLog.PrebuiltMealID).To(Equal(prebuiltMealID))
			Expect(repo.createdLog.CustomMealItemID).To(BeNil())
		})

		It("should reject invalid meal ID before loading meal details", func() {
			response, err := svc.Create(ctx, userID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourceCustom,
				MealID:   "not-a-uuid",
				Price:    8,
				EatenAt:  time.Date(2026, 7, 12, 11, 15, 0, 0, time.UTC),
				MealType: "lunch",
			})

			Expect(err).To(MatchError("invalid meal id"))
			Expect(response).To(BeNil())
			Expect(repo.createdLog).To(BeNil())
		})

		It("should reject unsupported log source", func() {
			response, err := svc.Create(ctx, userID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSource("manual"),
				MealID:   uuid.New().String(),
				Price:    8,
				EatenAt:  time.Date(2026, 7, 12, 11, 15, 0, 0, time.UTC),
				MealType: "lunch",
			})

			Expect(err).To(MatchError("unsupported meal log source"))
			Expect(response).To(BeNil())
			Expect(repo.createdLog).To(BeNil())
		})

		It("should reject future eaten time before loading meal details", func() {
			response, err := svc.Create(ctx, userID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourceCustom,
				MealID:   uuid.New().String(),
				Price:    8,
				EatenAt:  time.Date(2026, 8, 15, 12, 1, 0, 0, time.UTC),
				MealType: "lunch",
			})

			Expect(err).To(MatchError("eaten time cannot be in the future"))
			Expect(response).To(BeNil())
			Expect(repo.createdLog).To(BeNil())
		})
	})

	Describe("GetMonth", func() {
		It("should include budget remaining for current month", func() {
			preferenceService := mocks.NewPreferenceService(GinkgoT())
			svc = NewService(repo, nil, nil, preferenceService)
			svc.(*service).now = func() time.Time {
				return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
			}

			repo.listLogs = []model.MealLog{
				{
					ID:       uuid.New(),
					UserID:   userID,
					MealName: "Chicken Rice",
					Price:    8.50,
					Calories: 650,
					EatenAt:  time.Date(2026, 7, 12, 11, 15, 0, 0, time.UTC),
					MealType: "lunch",
				},
				{
					ID:       uuid.New(),
					UserID:   userID,
					MealName: "Nasi Lemak",
					Price:    12.50,
					Calories: 720,
					EatenAt:  time.Date(2026, 7, 13, 8, 15, 0, 0, time.UTC),
					MealType: "breakfast",
				},
			}

			preferenceService.EXPECT().
				GetByUserID(mock.Anything, userID).
				Return(&interfaces.PreferenceResponse{MonthlyMealBudget: 100}, nil).
				Once()

			response, err := svc.GetMonth(ctx, userID, "2026-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.Summary.Month).To(Equal("2026-07"))
			Expect(response.Summary.TotalMealsEaten).To(Equal(2))
			Expect(response.Summary.TotalSpent).To(Equal(21.0))
			Expect(response.Summary.TotalCalories).To(Equal(1370.0))
			Expect(response.Summary.ShowBudgetRemaining).To(BeTrue())
			Expect(response.Summary.BudgetRemaining).NotTo(BeNil())
			Expect(*response.Summary.BudgetRemaining).To(Equal(79.0))
			Expect(response.Items).To(HaveLen(2))
		})

		It("should hide budget remaining for past month", func() {
			preferenceService := mocks.NewPreferenceService(GinkgoT())
			svc = NewService(repo, nil, nil, preferenceService)
			svc.(*service).now = func() time.Time {
				return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
			}

			repo.listLogs = []model.MealLog{
				{
					ID:       uuid.New(),
					UserID:   userID,
					MealName: "Chicken Rice",
					Price:    8.50,
					Calories: 650,
					EatenAt:  time.Date(2026, 7, 12, 11, 15, 0, 0, time.UTC),
					MealType: "lunch",
				},
			}

			response, err := svc.GetMonth(ctx, userID, "2026-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.Summary.ShowBudgetRemaining).To(BeFalse())
			Expect(response.Summary.BudgetRemaining).To(BeNil())
		})

		It("should reject invalid month format", func() {
			response, err := svc.GetMonth(ctx, userID, "2026/07")

			Expect(err).To(MatchError("month must use YYYY-MM format"))
			Expect(response).To(BeNil())
		})
	})

	Describe("Update", func() {
		It("should update meal log price and eaten time without changing meal details", func() {
			repo.logs[logID] = storedLog()
			nextEatenAt := time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC)

			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price:    9.75,
				EatenAt:  nextEatenAt,
				MealType: "lunch",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.ID).To(Equal(logID.String()))
			Expect(response.MealName).To(Equal("Chicken Rice"))
			Expect(response.Calories).To(Equal(650.0))
			Expect(response.ProteinG).To(Equal(32.0))
			Expect(response.CarbsG).To(Equal(78.0))
			Expect(response.FatG).To(Equal(21.0))
			Expect(response.MealCategory).To(Equal([]string{"rice_dishes", "poultry"}))
			Expect(response.Price).To(Equal(9.75))
			Expect(response.EatenAt).To(Equal(nextEatenAt))
			Expect(response.MealType).To(Equal("lunch"))

			Expect(repo.findID).To(Equal(logID))
			Expect(repo.findUserID).To(Equal(userID))
			Expect(repo.updatedLog).NotTo(BeNil())
			Expect(repo.updatedLog.Price).To(Equal(9.75))
			Expect(repo.updatedLog.EatenAt).To(Equal(nextEatenAt))
			Expect(repo.updatedLog.MealType).To(Equal("lunch"))
		})

		It("should reject negative price before loading the meal log", func() {
			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price:    -1,
				EatenAt:  time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
				MealType: "lunch",
			})

			Expect(err).To(MatchError("price cannot be negative"))
			Expect(response).To(BeNil())
			Expect(repo.findID).To(Equal(uuid.Nil))
			Expect(repo.updatedLog).To(BeNil())
		})

		It("should reject missing eaten time before loading the meal log", func() {
			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price:    9.75,
				MealType: "lunch",
			})

			Expect(err).To(MatchError("eaten time is required"))
			Expect(response).To(BeNil())
			Expect(repo.findID).To(Equal(uuid.Nil))
			Expect(repo.updatedLog).To(BeNil())
		})

		It("should reject future eaten time before loading the meal log", func() {
			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price:    9.75,
				EatenAt:  time.Date(2026, 8, 15, 12, 1, 0, 0, time.UTC),
				MealType: "lunch",
			})

			Expect(err).To(MatchError("eaten time cannot be in the future"))
			Expect(response).To(BeNil())
			Expect(repo.findID).To(Equal(uuid.Nil))
			Expect(repo.updatedLog).To(BeNil())
		})

		It("should return an error when the meal log is not visible to the user", func() {
			repo.logs[logID] = storedLog()
			otherUserID := uuid.New()

			response, err := svc.Update(ctx, otherUserID, logID, interfaces.MealLogUpdateInput{
				Price:    9.75,
				EatenAt:  time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
				MealType: "lunch",
			})

			Expect(err).To(MatchError("meal log not found"))
			Expect(response).To(BeNil())
			Expect(repo.findID).To(Equal(logID))
			Expect(repo.findUserID).To(Equal(otherUserID))
			Expect(repo.updatedLog).To(BeNil())
		})
	})

	Describe("Delete", func() {
		It("should delete a meal log owned by the user", func() {
			repo.logs[logID] = storedLog()

			err := svc.Delete(ctx, userID, logID)

			Expect(err).NotTo(HaveOccurred())
			Expect(repo.findID).To(Equal(logID))
			Expect(repo.findUserID).To(Equal(userID))
			Expect(repo.deletedID).To(Equal(logID))
			Expect(repo.deletedUserID).To(Equal(userID))
			Expect(repo.logs).NotTo(HaveKey(logID))
		})

		It("should not delete a meal log owned by another user", func() {
			repo.logs[logID] = storedLog()
			otherUserID := uuid.New()

			err := svc.Delete(ctx, otherUserID, logID)

			Expect(err).To(MatchError("meal log not found"))
			Expect(repo.findID).To(Equal(logID))
			Expect(repo.findUserID).To(Equal(otherUserID))
			Expect(repo.deletedID).To(Equal(uuid.Nil))
			Expect(repo.logs).To(HaveKey(logID))
		})
	})
})
