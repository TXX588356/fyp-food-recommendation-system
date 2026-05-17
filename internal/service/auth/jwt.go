package auth

import (
	"errors"
	"fyp/food-rs/types/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type authClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *service) generateToken(user *model.User) (string, error) {
	if s.jwtSecret == "" {
		return "", errors.New("jwt secret is not configured")
	}

	now := time.Now()
	claims := authClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(48 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.jwtSecret))
}
