package interfaces

import (
	"context"
)

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type RefreshInput struct {
	RefreshToken string `json:"refreshToken"`
}

type ResetPasswordInput struct {
	Email       string `json:"email"`
	NewPassword string `json:"newPassword"`
}

type AuthUser struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Email                  string `json:"email"`
	HasCompletedOnboarding bool   `json:"hasCompletedOnboarding"`
}

type AuthResult struct {
	User         AuthUser `json:"user"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
}

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, rawRefreshToken string) (*AuthResult, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	ResetPassword(ctx context.Context, input ResetPasswordInput) error
}
