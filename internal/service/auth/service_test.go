package auth

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/types/model"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestAuthService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Service Suite")
}

var _ = Describe("Auth service", func() {
	var (
		ctx              context.Context
		userRepo         *mocks.UserRepository
		refreshTokenRepo *mocks.RefreshTokenRepository
		svc              interfaces.AuthService
	)

	BeforeEach(func() {
		ctx = context.Background()
		userRepo = mocks.NewUserRepository(GinkgoT())
		refreshTokenRepo = mocks.NewRefreshTokenRepository(GinkgoT())
		svc = NewService(userRepo, refreshTokenRepo, "test-secret")
	})

	It("should register a new user and create a refresh token", func() {
		userID := uuid.New()

		userRepo.EXPECT().
			FindByEmail(ctx, "ada@example.com").
			Return(nil, gorm.ErrRecordNotFound).
			Once()

		userRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(user *model.User) bool {
				Expect(bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("secret123"))).To(Succeed())
				return user.Name == "Ada Lovelace" && user.Email == "ada@example.com"
			})).
			Return(&model.User{
				ID:           userID,
				Name:         "Ada Lovelace",
				Email:        "ada@example.com",
				PasswordHash: "stored-hash",
			}, nil).
			Once()

		refreshTokenRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(token *model.RefreshToken) bool {
				return token.UserID == userID && token.TokenHash != "" && token.ExpiresAt.After(time.Now())
			})).
			Return(nil).
			Once()

		result, err := svc.Register(ctx, interfaces.RegisterInput{
			Name:     " Ada Lovelace ",
			Email:    " ADA@example.com ",
			Password: "secret123",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.User.ID).To(Equal(userID.String()))
		Expect(result.User.Email).To(Equal("ada@example.com"))
		Expect(result.AccessToken).NotTo(BeEmpty())
		Expect(result.RefreshToken).NotTo(BeEmpty())
	})

	It("should reject duplicate registration email", func() {
		userRepo.EXPECT().
			FindByEmail(ctx, "ada@example.com").
			Return(&model.User{ID: uuid.New(), Email: "ada@example.com"}, nil).
			Once()

		result, err := svc.Register(ctx, interfaces.RegisterInput{
			Name:     "Ada",
			Email:    "ada@example.com",
			Password: "secret123",
		})

		Expect(err).To(MatchError("email already registered, please proceed to login or register using different email"))
		Expect(result).To(BeNil())
	})

	It("should reject invalid register input before checking repository", func() {
		result, err := svc.Register(ctx, interfaces.RegisterInput{
			Name:     "A",
			Email:    "not-email",
			Password: "short",
		})

		Expect(err).To(MatchError("name must be at least 2 characters"))
		Expect(result).To(BeNil())
	})

	It("should login with valid credentials", func() {
		userID := uuid.New()
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
		Expect(err).NotTo(HaveOccurred())

		userRepo.EXPECT().
			FindByEmail(ctx, "ada@example.com").
			Return(&model.User{
				ID:                     userID,
				Name:                   "Ada Lovelace",
				Email:                  "ada@example.com",
				PasswordHash:           string(passwordHash),
				HasCompletedOnboarding: true,
			}, nil).
			Once()

		refreshTokenRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(token *model.RefreshToken) bool {
				return token.UserID == userID && token.TokenHash != ""
			})).
			Return(nil).
			Once()

		result, err := svc.Login(ctx, interfaces.LoginInput{
			Email:    " ADA@example.com ",
			Password: "secret123",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.User.ID).To(Equal(userID.String()))
		Expect(result.User.HasCompletedOnboarding).To(BeTrue())
		Expect(result.AccessToken).NotTo(BeEmpty())
		Expect(result.RefreshToken).NotTo(BeEmpty())
	})

	It("should reject login with wrong password", func() {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
		Expect(err).NotTo(HaveOccurred())

		userRepo.EXPECT().
			FindByEmail(ctx, "ada@example.com").
			Return(&model.User{
				ID:           uuid.New(),
				Email:        "ada@example.com",
				PasswordHash: string(passwordHash),
			}, nil).
			Once()

		result, err := svc.Login(ctx, interfaces.LoginInput{
			Email:    "ada@example.com",
			Password: "wrong123",
		})

		Expect(err).To(MatchError("invalid email or password"))
		Expect(result).To(BeNil())
	})

	It("should refresh tokens by rotating the stored refresh token", func() {
		userID := uuid.New()
		oldTokenID := uuid.New()
		rawRefreshToken := "old-refresh-token"

		refreshTokenRepo.EXPECT().
			FindByHash(ctx, hashRefreshToken(rawRefreshToken)).
			Return(&model.RefreshToken{
				ID:        oldTokenID,
				UserID:    userID,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil).
			Once()

		userRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{
				ID:    userID,
				Name:  "Ada Lovelace",
				Email: "ada@example.com",
			}, nil).
			Once()

		refreshTokenRepo.EXPECT().
			Rotate(ctx, oldTokenID, mock.MatchedBy(func(token *model.RefreshToken) bool {
				return token.UserID == userID && token.TokenHash != "" && token.ExpiresAt.After(time.Now())
			})).
			Return(nil).
			Once()

		result, err := svc.Refresh(ctx, rawRefreshToken)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.User.ID).To(Equal(userID.String()))
		Expect(result.AccessToken).NotTo(BeEmpty())
		Expect(result.RefreshToken).NotTo(BeEmpty())
		Expect(result.RefreshToken).NotTo(Equal(rawRefreshToken))
	})

	It("should reject expired refresh token", func() {
		rawRefreshToken := "old-refresh-token"

		refreshTokenRepo.EXPECT().
			FindByHash(ctx, hashRefreshToken(rawRefreshToken)).
			Return(&model.RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ExpiresAt: time.Now().Add(-time.Hour),
			}, nil).
			Once()

		result, err := svc.Refresh(ctx, rawRefreshToken)

		Expect(err).To(MatchError("invalid refresh token"))
		Expect(result).To(BeNil())
	})

	It("should logout by revoking the refresh token hash", func() {
		rawRefreshToken := "refresh-token"

		refreshTokenRepo.EXPECT().
			RevokeByHash(ctx, hashRefreshToken(rawRefreshToken)).
			Return(nil).
			Once()

		err := svc.Logout(ctx, rawRefreshToken)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should return configured errors from refresh token creation", func() {
		userID := uuid.New()
		createErr := errors.New("refresh token store failed")

		userRepo.EXPECT().
			FindByEmail(ctx, "ada@example.com").
			Return(nil, gorm.ErrRecordNotFound).
			Once()

		userRepo.EXPECT().
			Create(ctx, mock.AnythingOfType("*model.User")).
			Return(&model.User{
				ID:    userID,
				Name:  "Ada",
				Email: "ada@example.com",
			}, nil).
			Once()

		refreshTokenRepo.EXPECT().
			Create(ctx, mock.AnythingOfType("*model.RefreshToken")).
			Return(createErr).
			Once()

		result, err := svc.Register(ctx, interfaces.RegisterInput{
			Name:     "Ada",
			Email:    "ada@example.com",
			Password: "secret123",
		})

		Expect(err).To(MatchError(createErr))
		Expect(result).To(BeNil())
	})
})
