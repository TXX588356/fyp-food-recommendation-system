package auth

import (
	"context"
	"errors"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type service struct {
	userRepo         interfaces.UserRepository
	refreshTokenRepo interfaces.RefreshTokenRepository
	jwtSecret        string
}

func NewService(userRepo interfaces.UserRepository, refreshTokenRepo interfaces.RefreshTokenRepository, jwtSecret string) interfaces.AuthService {
	return &service{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        jwtSecret,
	}
}

func (s *service) Register(ctx context.Context, input interfaces.RegisterInput) (*interfaces.AuthResult, error) {
	// TODO:
	// 1. validate input
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	// 2. check existing user by email
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if existingUser != nil && err == nil {
		return nil, errors.New("email already registered, please proceed to login or register using different email")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 3. hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	// 4. create model.User
	user := &model.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	// 5. generate JWT
	accessToken, err := s.generateAccessToken(createdUser)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshTokenHash, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = s.refreshTokenRepo.Create(ctx, &model.RefreshToken{
		UserID:    createdUser.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	})
	if err != nil {
		return nil, err
	}
	// 6. return AuthResult
	return &interfaces.AuthResult{
		User: interfaces.AuthUser{
			ID:                     createdUser.ID.String(),
			Name:                   createdUser.Name,
			Email:                  createdUser.Email,
			HasCompletedOnboarding: createdUser.HasCompletedOnboarding,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) Login(ctx context.Context, input interfaces.LoginInput) (*interfaces.AuthResult, error) {
	// TODO:
	if err := validateLoginInput(input); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	// 1. find user by email
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("invalid email or password")
	}

	if err != nil {
		return nil, err
	}

	// 2. compare password hash
	hashedPassword := []byte(existingUser.PasswordHash) //hash retrieved from db
	userPassword := []byte(input.Password)              // plain-text password provided by user

	err = bcrypt.CompareHashAndPassword(hashedPassword, userPassword)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}
	// 3. generate JWT
	accessToken, err := s.generateAccessToken(existingUser)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshTokenHash, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = s.refreshTokenRepo.Create(ctx, &model.RefreshToken{
		UserID:    existingUser.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	})
	if err != nil {
		return nil, err
	}
	// 4. return AuthResult
	return &interfaces.AuthResult{
		User: interfaces.AuthUser{
			ID:                     existingUser.ID.String(),
			Name:                   existingUser.Name,
			Email:                  existingUser.Email,
			HasCompletedOnboarding: existingUser.HasCompletedOnboarding},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) Refresh(ctx context.Context, rawRefreshToken string) (*interfaces.AuthResult, error) {
	hash := hashRefreshToken(rawRefreshToken)

	oldToken, err := s.refreshTokenRepo.FindByHash(ctx, hash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if oldToken.RevokedAt != nil || time.Now().After(oldToken.ExpiresAt) {
		return nil, errors.New("invalid refresh token")
	}

	user, err := s.userRepo.FindByID(ctx, oldToken.UserID)
	if err != nil {
		return nil, err
	}

	newRawRefreshToken, newHash, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	newRefreshToken := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	}

	err = s.refreshTokenRepo.Rotate(ctx, oldToken.ID, newRefreshToken)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &interfaces.AuthResult{
		User: interfaces.AuthUser{
			ID:                     user.ID.String(),
			Name:                   user.Name,
			Email:                  user.Email,
			HasCompletedOnboarding: user.HasCompletedOnboarding},
		AccessToken:  accessToken,
		RefreshToken: newRawRefreshToken,
	}, nil
}

func (s *service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := hashRefreshToken(rawRefreshToken)
	return s.refreshTokenRepo.RevokeByHash(ctx, hash)
}

func validateRegisterInput(input interfaces.RegisterInput) error {
	required := map[string]string{
		"name":     input.Name,
		"email":    input.Email,
		"password": input.Password,
	}

	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}

	if len(strings.TrimSpace(input.Name)) < 2 {
		return errors.New("name must be at least 2 characters")
	}

	if _, err := mail.ParseAddress(strings.TrimSpace(input.Email)); err != nil {
		return errors.New("invalid email")

	}

	if len(input.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	return nil
}

func validateLoginInput(input interfaces.LoginInput) error {
	required := map[string]string{
		"email":    input.Email,
		"password": input.Password,
	}

	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}

	if _, err := mail.ParseAddress(strings.TrimSpace(input.Email)); err != nil {
		return errors.New("invalid email")

	}

	if len(input.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	return nil
}
