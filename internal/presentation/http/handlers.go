package httphandlers

import (
	"net/http"

	"balance-web/internal/infrastructure/memory"
	"balance-web/web/templates"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// Handlers holds dependencies for HTTP request handlers.
type Handlers struct {
	store *memory.Store
}

// NewHandlers creates a new Handlers instance injected with the store.
func NewHandlers(store *memory.Store) *Handlers {
	return &Handlers{
		store: store,
	}
}

// RegisterRoutes registers all HTTP routes on the Echo instance.
func (h *Handlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/", h.IndexHandler)
	e.GET("/health", h.HealthHandler)
	
	// API routes
	api := e.Group("/api")
	api.GET("/activities", h.GetActivities)
}

// HealthHandler returns a simple JSON health check response.
func (h *Handlers) HealthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// GetActivities returns all seeded ActivityProfiles from the store as JSON.
func (h *Handlers) GetActivities(c echo.Context) error {
	activities, err := h.store.FindAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, activities)
}

// IndexHandler renders the main index page using templ.
func (h *Handlers) IndexHandler(c echo.Context) error {
	component := templates.Index()
	return Render(c, http.StatusOK, component)
}

// Render is a helper that renders a templ component into an Echo response.
func Render(c echo.Context, statusCode int, t templ.Component) error {
	c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().Writer.WriteHeader(statusCode)
	return t.Render(c.Request().Context(), c.Response().Writer)
}
