package httphandlers

import (
	"net/http"
	"strings"

	"balance-web/internal/infrastructure/auth"
	"balance-web/internal/infrastructure/turso"

	"github.com/labstack/echo/v4"
)

// FirebaseAuthMiddleware creates an Echo middleware that verifies Firebase ID tokens.
// It extracts the token from three sources in priority order:
//  1. Authorization: Bearer <token> header (iOS API calls)
//  2. ?token=<token> query parameter (WebSocket upgrades)
//  3. session_token HTTP cookie (browser SSR requests)
//
// On failure, returns 401 JSON — suitable for API routes.
func FirebaseAuthMiddleware(firebaseAuth *auth.FirebaseAuth, store *turso.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenString := extractToken(c)

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
			if strings.TrimSpace(uid) == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid authentication token",
				})
			}

			if store == nil || store.DB == nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "user provisioning store unavailable",
				})
			}

			if err := turso.EnsureUserProvisioned(store.DB, uid); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "failed to provision user",
				})
			}

			// Inject the authenticated user ID into the Echo context
			c.Set("user_id", uid)
			return next(c)
		}
	}
}

// PageAuthMiddleware creates an Echo middleware for browser page navigation.
// Instead of returning 401 JSON on failure, it redirects to /login.
func PageAuthMiddleware(firebaseAuth *auth.FirebaseAuth, store *turso.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenString := extractToken(c)

			if tokenString == "" {
				return c.Redirect(http.StatusFound, "/login")
			}

			uid, err := firebaseAuth.VerifyToken(tokenString)
			if err != nil {
				return c.Redirect(http.StatusFound, "/login")
			}
			if strings.TrimSpace(uid) == "" {
				return c.Redirect(http.StatusFound, "/login")
			}

			if store == nil || store.DB == nil {
				return c.String(http.StatusInternalServerError, "failed to provision user")
			}

			if err := turso.EnsureUserProvisioned(store.DB, uid); err != nil {
				return c.String(http.StatusInternalServerError, "failed to provision user")
			}

			c.Set("user_id", uid)
			return next(c)
		}
	}
}

// extractToken checks three sources for a Firebase ID token:
//  1. Authorization: Bearer <token> header
//  2. ?token=<token> query parameter
//  3. session_token HTTP cookie
func extractToken(c echo.Context) string {
	// 1. Try Authorization: Bearer <token> header
	authHeader := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 2. Fallback: ?token=<token> query param (for WebSocket upgrades)
	if token := c.QueryParam("token"); token != "" {
		return token
	}

	// 3. Fallback: session_token cookie (for browser SSR / HTMX requests)
	cookie, err := c.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}
