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
	userRepo interfaces.UserRepository
}

func NewService(userRepo interfaces.UserRepository) interfaces.AuthService {
	return &service{
		userRepo: userRepo,
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
		return nil, errors.New("email already registered")
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
			ID:    createdUser.ID.String(),
			Name:  createdUser.Name,
			Email: createdUser.Email,
		},
		Token: token,
	}, nil
}

func (s *service) Login(ctx context.Context, input interfaces.LoginInput) (*interfaces.AuthResult, error) {
	// TODO:
	// 1. find user by email
	// 2. compare password hash
	// 3. generate JWT
	// 4. return AuthResult
	return nil, nil
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
