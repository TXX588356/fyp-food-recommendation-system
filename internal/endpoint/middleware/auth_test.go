package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuthMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Middleware Suite")
}

var _ = Describe("Auth middleware", func() {
	const jwtSecret = "test-secret"

	buildToken := func(userID string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
			UserID: userID,
			Email:  "test@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   userID,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})

		signed, err := token.SignedString([]byte(jwtSecret))
		Expect(err).NotTo(HaveOccurred())
		return signed
	}

	It("rejects missing authorization header", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := Auth(jwtSecret)(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c)

		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		Expect(rec.Body.String()).To(ContainSubstring("missing authorization header"))
	})

	It("rejects invalid authorization header format", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderAuthorization, "Token abc")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := Auth(jwtSecret)(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c)

		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		Expect(rec.Body.String()).To(ContainSubstring("invalid authorization header"))
	})

	It("stores valid token user ID in context", func() {
		e := echo.New()
		userID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+buildToken(userID.String()))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := Auth(jwtSecret)(func(c *echo.Context) error {
			got, err := UserIDFromContext(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(userID))
			return c.NoContent(http.StatusOK)
		})(c)

		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("rejects token with invalid user ID claim", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+buildToken("not-a-uuid"))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := Auth(jwtSecret)(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c)

		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		Expect(rec.Body.String()).To(ContainSubstring("invalid token user id"))
	})

	It("returns an error when user ID is missing from context", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		userID, err := UserIDFromContext(c)

		Expect(err).To(MatchError("user id missing from context"))
		Expect(userID).To(Equal(uuid.Nil))
	})

	It("returns an error when user ID has wrong context type", func() {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(userIDContextKey, "not-a-uuid")

		userID, err := UserIDFromContext(c)

		Expect(err).To(MatchError("invalid user id in context"))
		Expect(userID).To(Equal(uuid.Nil))
	})
})
