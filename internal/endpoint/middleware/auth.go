package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

const userIDContextKey = "userID"

type authClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Auth validates the Authorization: Bearer <token> header
// If valid, it stores the parsed userID into Echo context so handlers can use it
func Auth(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing authorization header",
				})
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid authorization header",
				})
			}

			tokenText := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if tokenText == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing token",
				})
			}

			claims := &authClaims{}

			token, err := jwt.ParseWithClaims(tokenText, claims, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}

				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid or expired token",
				})
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid token user id",
				})
			}

			// Store userID in Echo context for downstream handlers
			c.Set(userIDContextKey, userID)

			return next(c)
		}
	}
}

// UserIDFromContext reads the authenticated user ID set by the Auth middleware
// Handlers should call this instead of parsing JWT again
func UserIDFromContext(c *echo.Context) (uuid.UUID, error) {
	value := c.Get(userIDContextKey)
	if value == nil {
		return uuid.Nil, errors.New("user id missing from context")
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user id in context")
	}

	return userID, nil
}
