//go:build integration

package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	"fyp/food-rs/types/model"

	authservice "fyp/food-rs/internal/service/auth"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"golang.org/x/crypto/bcrypt"

	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("Auth integration", func() {
	const jwtSecret = "integration-test-secret"

	newAuthService := func(tx *gorm.DB) interfaces.AuthService {
		userRepo := postgres.NewUserPostgresRepository(tx)
		refreshTokenRepo := postgres.NewRefreshTokenPostgresRepository(tx)

		return authservice.NewService(userRepo, refreshTokenRepo, jwtSecret)
	}

	countUsersByEmail := func(tx *gorm.DB, email string) int64 {
		var count int64
		Expect(tx.Model(&model.User{}).
			Where("email = ?", email).
			Count(&count).Error).
			NotTo(HaveOccurred())

		return count
	}

	countRefreshTokensByUserID := func(tx *gorm.DB, userID uuid.UUID) int64 {
		var count int64
		Expect(tx.Model(&model.RefreshToken{}).
			Where("user_id = ?", userID).
			Count(&count).Error).
			NotTo(HaveOccurred())

		return count
	}

	findRefreshTokenByRawToken := func(tx *gorm.DB, rawToken string) model.RefreshToken {
		hashBytes := sha256.Sum256([]byte(rawToken))
		tokenHash := hex.EncodeToString(hashBytes[:])

		var token model.RefreshToken
		Expect(tx.Where("token_hash = ?", tokenHash).First(&token).Error).
			NotTo(HaveOccurred())

		return token
	}

	createRegisteredUser := func(tx *gorm.DB, email, password string) *interfaces.AuthResult {
		ctx := GinkgoT().Context()
		svc := newAuthService(tx)

		result, err := svc.Register(ctx, interfaces.RegisterInput{
			Name:     "Integration Test User",
			Email:    email,
			Password: password,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())

		return result
	}

	Describe("Register a new user successfully", func() {
		It("should store the user in PostgreSQL, hash the passowrd, and return access + refresh token", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@email.com", uuid.NewString())

			result, err := svc.Register(ctx, interfaces.RegisterInput{
				Name:     "Integration User",
				Email:    email,
				Password: "password123",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.User.Email).To(Equal(email))
			Expect(result.AccessToken).NotTo(BeEmpty())
			Expect(result.RefreshToken).NotTo(BeEmpty())

			var storedUser model.User
			Expect(tx.Where("email = ?", email).First(&storedUser).Error).
				NotTo(HaveOccurred())

			Expect(storedUser.Name).To(Equal("Integration User"))
			Expect(storedUser.PasswordHash).NotTo(Equal("password123"))
			Expect(bcrypt.CompareHashAndPassword(
				[]byte(storedUser.PasswordHash),
				[]byte("password123"),
			)).To(Succeed())

			Expect(countRefreshTokensByUserID(tx, storedUser.ID)).To(Equal(int64(1)))
		})
	})

	Describe("Reject registration with duplicate email", func() {
		It("should return duplicate email error and leaves only one user record", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())

			first, err := svc.Register(ctx, interfaces.RegisterInput{
				Name:     "First User",
				Email:    email,
				Password: "password123",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(first).NotTo(BeNil())

			second, err := svc.Register(ctx, interfaces.RegisterInput{
				Name:     "Second User",
				Email:    email,
				Password: "password456",
			})

			Expect(second).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("email already registered")))
			Expect(countUsersByEmail(tx, email)).To(Equal(int64(1)))
		})
	})

	Describe("Login with valid credentials", func() {
		It("should succeed and valid access plus refresh tokens", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())
			createRegisteredUser(tx, email, "password123")

			result, err := svc.Login(ctx, interfaces.LoginInput{
				Email:    email,
				Password: "password123",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.User.Email).To(Equal(email))
			Expect(result.AccessToken).NotTo(BeEmpty())
			Expect(result.RefreshToken).NotTo(BeEmpty())

			var storedUser model.User
			Expect(tx.Where("email = ?", email).First(&storedUser).Error).
				NotTo(HaveOccurred())

			Expect(countRefreshTokensByUserID(tx, storedUser.ID)).To(Equal(int64(2)))
		})
	})

	Describe("Reject login with incorrect password", func() {
		It("should fail login and return no authentication tokens", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())
			createRegisteredUser(tx, email, "password123")

			result, err := svc.Login(ctx, interfaces.LoginInput{
				Email:    email,
				Password: "wrong-password",
			})

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("invalid email or password"))

			var storedUser model.User
			Expect(tx.Where("email = ?", email).First(&storedUser).Error).
				NotTo(HaveOccurred())

			Expect(countRefreshTokensByUserID(tx, storedUser.ID)).To(Equal(int64(1)))
		})
	})

	Describe("Refresh session using valid refresh token", func() {
		It("should process the old token and return a new access token and refresh token", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())

			registered := createRegisteredUser(tx, email, "password123")

			loginResult, err := svc.Login(ctx, interfaces.LoginInput{
				Email:    email,
				Password: "password123",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(loginResult).NotTo(BeNil())

			oldRefreshToken := loginResult.RefreshToken

			refreshed, err := svc.Refresh(ctx, oldRefreshToken)

			Expect(err).NotTo(HaveOccurred())
			Expect(refreshed).NotTo(BeNil())
			Expect(refreshed.User.Email).To(Equal(email))
			Expect(refreshed.AccessToken).NotTo(BeEmpty())
			Expect(refreshed.RefreshToken).NotTo(BeEmpty())
			Expect(refreshed.RefreshToken).NotTo(Equal(oldRefreshToken))
			Expect(refreshed.RefreshToken).NotTo(Equal(registered.RefreshToken))

			oldTokenRow := findRefreshTokenByRawToken(tx, oldRefreshToken)
			Expect(oldTokenRow.RevokedAt).NotTo(BeNil())
			Expect(oldTokenRow.ReplacedBy).NotTo(BeNil())

			newTokenRow := findRefreshTokenByRawToken(tx, refreshed.RefreshToken)
			Expect(newTokenRow.RevokedAt).To(BeNil())
			Expect(newTokenRow.UserID).To(Equal(oldTokenRow.UserID))
		})
	})

	Describe("Reject reuse of revoked/old refresh token", func() {
		It("should fail refresh and do not issue new tokens", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())
			registered := createRegisteredUser(tx, email, "password123")

			firstRefresh, err := svc.Refresh(ctx, registered.RefreshToken)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstRefresh).NotTo(BeNil())

			secondRefresh, err := svc.Refresh(ctx, registered.RefreshToken)

			Expect(secondRefresh).To(BeNil())
			Expect(err).To(MatchError("invalid refresh token"))
		})
	})

	Describe("Logout and revoke refresh token", func() {
		It("should revoke the refresh token so it can no longer obtain new tokens", func() {
			tx, ctx := beginIntegrationTx()
			svc := newAuthService(tx)

			email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())
			registered := createRegisteredUser(tx, email, "password123")

			err := svc.Logout(ctx, registered.RefreshToken)
			Expect(err).NotTo(HaveOccurred())

			tokenRow := findRefreshTokenByRawToken(tx, registered.RefreshToken)
			Expect(tokenRow.RevokedAt).NotTo(BeNil())

			result, err := svc.Refresh(ctx, registered.RefreshToken)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("invalid refresh token"))
		})
	})
})
