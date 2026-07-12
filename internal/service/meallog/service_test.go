package meallog

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestMealLogService runs the meal-log service test suite.
func TestMealLogService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Meal Log Service Suite")
}

type testMealLogRepository struct {
	logs map[uuid.UUID]*model.MealLog

	findID     uuid.UUID
	findUserID uuid.UUID
	findErr    error

	updatedLog *model.MealLog
	updateErr  error

	deletedID     uuid.UUID
	deletedUserID uuid.UUID
	deleteErr     error
}

func (r *testMealLogRepository) Create(context.Context, *model.MealLog) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) ListByUserAndMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) ListByUserAndRange(context.Context, uuid.UUID, time.Time, time.Time) ([]model.MealLog, error) {
	return nil, nil
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
	})

	storedLog := func() *model.MealLog {
		customMealID := uuid.New()

		return &model.MealLog{
			ID:               logID,
			UserID:           userID,
			CustomMealItemID: &customMealID,
			MealName:         "Chicken Rice",
			Calories:         650,
			Price:            8.50,
			EatenAt:          time.Date(2026, 7, 12, 8, 30, 0, 0, time.UTC),
			MealCategory:     model.StringArray{"rice_dishes", "poultry"},
		}
	}

	Describe("Update", func() {
		It("should update meal log price and eaten time without changing meal details", func() {
			repo.logs[logID] = storedLog()
			nextEatenAt := time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC)

			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price:   9.75,
				EatenAt: nextEatenAt,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.ID).To(Equal(logID.String()))
			Expect(response.MealName).To(Equal("Chicken Rice"))
			Expect(response.Calories).To(Equal(650.0))
			Expect(response.MealCategory).To(Equal([]string{"rice_dishes", "poultry"}))
			Expect(response.Price).To(Equal(9.75))
			Expect(response.EatenAt).To(Equal(nextEatenAt))

			Expect(repo.findID).To(Equal(logID))
			Expect(repo.findUserID).To(Equal(userID))
			Expect(repo.updatedLog).NotTo(BeNil())
			Expect(repo.updatedLog.Price).To(Equal(9.75))
			Expect(repo.updatedLog.EatenAt).To(Equal(nextEatenAt))
		})

		It("should reject negative price before loading the meal log", func() {
			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price:   -1,
				EatenAt: time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
			})

			Expect(err).To(MatchError("price cannot be negative"))
			Expect(response).To(BeNil())
			Expect(repo.findID).To(Equal(uuid.Nil))
			Expect(repo.updatedLog).To(BeNil())
		})

		It("should reject missing eaten time before loading the meal log", func() {
			response, err := svc.Update(ctx, userID, logID, interfaces.MealLogUpdateInput{
				Price: 9.75,
			})

			Expect(err).To(MatchError("eaten time is required"))
			Expect(response).To(BeNil())
			Expect(repo.findID).To(Equal(uuid.Nil))
			Expect(repo.updatedLog).To(BeNil())
		})

		It("should return an error when the meal log is not visible to the user", func() {
			repo.logs[logID] = storedLog()
			otherUserID := uuid.New()

			response, err := svc.Update(ctx, otherUserID, logID, interfaces.MealLogUpdateInput{
				Price:   9.75,
				EatenAt: time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
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
