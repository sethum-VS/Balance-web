package httphandlers

import (
	"net/http"
	"strings"

	"balance-web/internal/infrastructure/auth"

	"github.com/labstack/echo/v4"
)

// FirebaseAuthMiddleware creates an Echo middleware that verifies Firebase ID tokens.
// It extracts the token from the Authorization header (Bearer <token>) or
// falls back to a ?token= query parameter for WebSocket upgrade requests.
func FirebaseAuthMiddleware(firebaseAuth *auth.FirebaseAuth) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var tokenString string

			// 1. Try Authorization: Bearer <token> header
			authHeader := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// 2. Fallback: ?token=<token> query param (for WebSocket upgrades)
			if tokenString == "" {
				tokenString = c.QueryParam("token")
			}

			// No token found
			if tokenString == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing authentication token",
				})
			}

			// Verify token with Firebase Admin SDK
			uid, err := firebaseAuth.VerifyToken(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid authentication token",
				})
			}

			// Inject the authenticated user ID into the Echo context
			c.Set("user_id", uid)
			return next(c)
		}
	}
}
