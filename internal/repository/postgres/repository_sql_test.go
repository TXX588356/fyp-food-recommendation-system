package postgres

import (
	"context"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type sqlRecorder struct {
	logger.Interface
	statements []string
}

func newDryRunDB() (*gorm.DB, *sqlRecorder) {
	recorder := &sqlRecorder{Interface: logger.Discard}
	db, err := gorm.Open(pgdriver.Open("postgres://repository:repository@localhost/repository"), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		Logger:                 recorder,
	})
	Expect(err).NotTo(HaveOccurred())
	return db, recorder
}

func (r *sqlRecorder) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	if sql != "" {
		r.statements = append(r.statements, sql)
	}
}

var _ = Describe("user repository SQL", func() {
	It("should create users in the users table", func() {
		db, recorder := newDryRunDB()
		repository := NewUserPostgresRepository(db)

		_, err := repository.Create(context.Background(), &model.User{
			ID:           uuid.New(),
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: "hash",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`INSERT INTO "users"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"email"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"password_hash"`))
	})

	It("should find users by email", func() {
		db, recorder := newDryRunDB()
		repository := NewUserPostgresRepository(db)

		_, _ = repository.FindByEmail(context.Background(), "test@example.com")

		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`FROM "users"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`email = 'test@example.com'`))
		Expect(recorder.statements[0]).To(ContainSubstring(`LIMIT 1`))
	})

	It("should update onboarding status and timestamp", func() {
		db, recorder := newDryRunDB()
		repository := NewUserPostgresRepository(db)
		userID := uuid.New()

		err := repository.UpdateOnboardingStatus(context.Background(), userID, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`UPDATE "users"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"has_completed_onboarding"=true`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"updated_at"=now()`))
		Expect(recorder.statements[0]).To(ContainSubstring(`id = '` + userID.String() + `'`))
	})
})

var _ = Describe("refresh token repository SQL", func() {
	It("should create refresh tokens", func() {
		db, recorder := newDryRunDB()
		repository := NewRefreshTokenPostgresRepository(db)

		err := repository.Create(context.Background(), &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			TokenHash: "token-hash",
			ExpiresAt: time.Now().Add(time.Hour),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`INSERT INTO "refresh_tokens"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"token_hash"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`token-hash`))
	})

	It("should find refresh tokens by hash", func() {
		db, recorder := newDryRunDB()
		repository := NewRefreshTokenPostgresRepository(db)

		_, _ = repository.FindByHash(context.Background(), "token-hash")

		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`FROM "refresh_tokens"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`token_hash = 'token-hash'`))
		Expect(recorder.statements[0]).To(ContainSubstring(`LIMIT 1`))
	})

	It("should revoke active tokens by hash", func() {
		db, recorder := newDryRunDB()
		repository := NewRefreshTokenPostgresRepository(db)

		err := repository.RevokeByHash(context.Background(), "token-hash")

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`UPDATE "refresh_tokens"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"revoked_at"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`token_hash = 'token-hash' AND revoked_at IS NULL`))
	})
})

var _ = Describe("preference repository SQL", func() {
	It("should preload preference association tables when finding by user ID", func() {
		db, recorder := newDryRunDB()
		repository := NewPreferencePostgresRepository(db)
		userID := uuid.New()

		_, _ = repository.FindByUserID(context.Background(), userID)

		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`FROM "user_preferences"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`user_id = '` + userID.String() + `'`))
	})

	It("should update data sharing consent and timestamp", func() {
		db, recorder := newDryRunDB()
		repository := NewPreferencePostgresRepository(db)
		userID := uuid.New()

		err := repository.UpdateDataSharingConsent(context.Background(), userID, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`UPDATE "user_preferences"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"data_sharing_consent"=true`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"updated_at"=now()`))
		Expect(recorder.statements[0]).To(ContainSubstring(`user_id = '` + userID.String() + `'`))
	})

})

var _ = Describe("meal log repository SQL", func() {
	It("should create meal logs", func() {
		db, recorder := newDryRunDB()
		repository := NewMealLogPostgresRepository(db)

		_, err := repository.Create(context.Background(), &model.MealLog{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			MealName:     "Nasi Lemak",
			Price:        5.50,
			EatenAt:      time.Now(),
			MealType:     "lunch",
			MealCategory: model.StringArray{"rice_dishes"},
			Calories:     600,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`INSERT INTO "meal_logs"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`"meal_name"`))
	})

	It("should list meal logs by user and date range in descending eaten time", func() {
		db, recorder := newDryRunDB()
		repository := NewMealLogPostgresRepository(db)
		userID := uuid.New()
		start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)

		_, err := repository.ListByUserAndRange(context.Background(), userID, start, end)

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`FROM "meal_logs"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`user_id = '` + userID.String() + `'`))
		Expect(recorder.statements[0]).To(ContainSubstring(`eaten_at >=`))
		Expect(recorder.statements[0]).To(ContainSubstring(`eaten_at <`))
		Expect(recorder.statements[0]).To(ContainSubstring(`deleted_at IS NULL`))
		Expect(recorder.statements[0]).To(ContainSubstring(`ORDER BY eaten_at DESC`))
	})

	It("should find meal logs by ID and owner", func() {
		db, recorder := newDryRunDB()
		repository := NewMealLogPostgresRepository(db)
		logID := uuid.New()
		userID := uuid.New()

		_, _ = repository.FindByIDAndUser(context.Background(), logID, userID)

		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`id = '` + logID.String() + `' AND user_id = '` + userID.String() + `'`))
		Expect(recorder.statements[0]).To(ContainSubstring(`deleted_at IS NULL`))
		Expect(recorder.statements[0]).To(ContainSubstring(`LIMIT 1`))
	})

	It("should delete meal logs by ID and owner", func() {
		db, recorder := newDryRunDB()
		repository := NewMealLogPostgresRepository(db)
		logID := uuid.New()
		userID := uuid.New()

		err := repository.Delete(context.Background(), logID, userID)

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`UPDATE "meal_logs" SET "deleted_at"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`id = '` + logID.String() + `' AND user_id = '` + userID.String() + `'`))
	})
})

var _ = Describe("custom meal repository SQL", func() {
	It("should list meals owned by the user with optional name filtering", func() {
		db, recorder := newDryRunDB()
		repository := NewCustomMealPostgresRepository(db)
		userID := uuid.New()

		_, err := repository.ListOwnedByUser(context.Background(), userID, "bowl")

		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.statements).NotTo(BeEmpty())
		sql := stringsJoined(recorder.statements)
		Expect(sql).To(ContainSubstring(`FROM "custom_meal_items"`))
		Expect(sql).To(ContainSubstring(`created_by = '` + userID.String() + `'`))
		Expect(sql).To(ContainSubstring(`LOWER(name) LIKE LOWER('%bowl%')`))
	})

	It("should list shared meals from consenting users only", func() {
		db, recorder := newDryRunDB()
		repository := NewCustomMealPostgresRepository(db)
		userID := uuid.New()

		_, err := repository.ListSharedFromOtherUsers(context.Background(), userID, "rice")

		Expect(err).NotTo(HaveOccurred())
		sql := stringsJoined(recorder.statements)
		Expect(sql).To(ContainSubstring(`JOIN user_preferences`))
		Expect(sql).To(ContainSubstring(`custom_meal_items.created_by <> '` + userID.String() + `'`))
		Expect(sql).To(ContainSubstring(`user_preferences.data_sharing_consent = true`))
		Expect(sql).To(ContainSubstring(`LOWER(custom_meal_items.name) LIKE LOWER('%rice%')`))
	})

	It("should find a meal visible to the current user", func() {
		db, recorder := newDryRunDB()
		repository := NewCustomMealPostgresRepository(db)
		userID := uuid.New()
		mealID := uuid.New()

		_, _ = repository.FindVisibleByID(context.Background(), userID, mealID)

		sql := stringsJoined(recorder.statements)
		Expect(sql).To(ContainSubstring(`LEFT JOIN user_preferences`))
		Expect(sql).To(ContainSubstring(`custom_meal_items.id = '` + mealID.String() + `'`))
		Expect(sql).To(ContainSubstring(`custom_meal_items.created_by = '` + userID.String() + `'`))
		Expect(sql).To(ContainSubstring(`user_preferences.data_sharing_consent = true`))
	})

	It("should delete only owned meals", func() {
		db, recorder := newDryRunDB()
		repository := NewCustomMealPostgresRepository(db)
		userID := uuid.New()
		mealID := uuid.New()

		_ = repository.DeleteOwned(context.Background(), userID, mealID)

		Expect(recorder.statements).NotTo(BeEmpty())
		Expect(recorder.statements[0]).To(ContainSubstring(`UPDATE "custom_meal_items" SET "deleted_at"`))
		Expect(recorder.statements[0]).To(ContainSubstring(`id = '` + mealID.String() + `' AND created_by = '` + userID.String() + `'`))
	})
})

func stringsJoined(values []string) string {
	result := ""
	for _, value := range values {
		result += value + "\n"
	}
	return result
}

var _ interfaces.UserRepository = (*userRepository)(nil)
var _ interfaces.RefreshTokenRepository = (*refreshTokenRepository)(nil)
var _ interfaces.PreferenceRepository = (*preferenceRepository)(nil)
var _ interfaces.MealLogRepository = (*mealLogRepository)(nil)
var _ interfaces.CustomMealRepository = (*customMealRepository)(nil)
