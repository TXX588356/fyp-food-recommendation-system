package auth

import (
	"context"
	"errors"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type service struct {
	userRepo  interfaces.UserRepository
	jwtSecret string
}

func NewService(userRepo interfaces.UserRepository, jwtSecret string) interfaces.AuthService {
	return &service{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
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
	token, err := s.generateToken(createdUser)
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
		Token: token,
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
	token, err := s.generateToken(existingUser)
	if err != nil {
		return nil, err
	}
	// 4. return AuthResult
	return &interfaces.AuthResult{
		User: interfaces.AuthUser{
			ID:                     existingUser.ID.String(),
			Name:                   existingUser.Name,
			Email:                  existingUser.Email,
			HasCompletedOnboarding: existingUser.HasCompletedOnboarding,
		},
		Token: token,
	}, nil
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
