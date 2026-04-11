package httphandlers

import (
	"log"
	"net/http"

	"balance-web/internal/application"
	"balance-web/internal/domain"
	"balance-web/internal/infrastructure/memory"
	"balance-web/internal/infrastructure/websocket"
	"balance-web/web/templates"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// Handlers holds dependencies for HTTP request handlers.
type Handlers struct {
	store        *memory.Store
	timerService *application.TimerService
	hub          *websocket.Hub
	// Track the active session globally (single-user for now)
	activeSessionID string
}

// NewHandlers creates a new Handlers instance with all dependencies.
func NewHandlers(store *memory.Store, timerService *application.TimerService, hub *websocket.Hub) *Handlers {
	return &Handlers{
		store:        store,
		timerService: timerService,
		hub:          hub,
	}
}

// RegisterRoutes registers all HTTP routes on the Echo instance.
func (h *Handlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/", h.IndexHandler)
	e.GET("/health", h.HealthHandler)

	// API routes
	api := e.Group("/api")
	api.GET("/activities", h.GetActivities)
	api.POST("/timer/start", h.StartTimer)
	api.POST("/timer/stop", h.StopTimer)
}

// HealthHandler returns a simple JSON health check response.
func (h *Handlers) HealthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// GetActivities returns all seeded ActivityProfiles from the store as JSON.
func (h *Handlers) GetActivities(c echo.Context) error {
	activities, err := h.store.FindAllActivities()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, activities)
}

// StartTimer handles POST /api/timer/start?activityID=xxx
func (h *Handlers) StartTimer(c echo.Context) error {
	activityID := c.QueryParam("activityID")
	if activityID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "activityID is required")
	}

	// If there's already an active session, stop it first
	if h.activeSessionID != "" {
		_, err := h.timerService.StopSession(h.activeSessionID)
		if err != nil {
			log.Printf("Error auto-stopping previous session: %v", err)
		}
	}

	session, err := h.timerService.StartSession(activityID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	h.activeSessionID = session.ID

	// Look up the activity to include its name in the event payload
	activity, _ := h.store.FindActivityByID(activityID)
	activityName := ""
	activityCategory := ""
	if activity != nil {
		activityName = activity.Name
		activityCategory = string(activity.Category)
	}

	// Calculate the current global CR balance (base) at session start
	allSessions, _ := h.store.FindAllSessions()
	baseBalance := 0
	for _, s := range allSessions {
		if s.Status == domain.SessionStatusCompleted {
			baseBalance += s.CreditsEarned
		}
	}

	// Broadcast TIMER_STARTED event with baseBalance for client-side ticking
	h.hub.Broadcast <- &domain.WSEvent{
		Type: domain.EventTimerStarted,
		Payload: map[string]interface{}{
			"sessionID":        session.ID,
			"activityID":       activityID,
			"activityName":     activityName,
			"activityCategory": activityCategory,
			"startTime":        session.StartTime,
			"baseBalance":      baseBalance,
		},
	}

	return c.NoContent(http.StatusNoContent)
}

// StopTimer handles POST /api/timer/stop
func (h *Handlers) StopTimer(c echo.Context) error {
	if h.activeSessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no active session")
	}

	session, err := h.timerService.StopSession(h.activeSessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	h.activeSessionID = ""

	// Broadcast TIMER_STOPPED immediately
	h.hub.Broadcast <- &domain.WSEvent{
		Type: domain.EventTimerStopped,
		Payload: map[string]interface{}{
			"sessionID":     session.ID,
			"duration":      session.Duration,
			"creditsEarned": session.CreditsEarned,
		},
	}

	// Immediately follow with BALANCE_UPDATED so clients sync to final CR
	allSessions, _ := h.store.FindAllSessions()
	totalBalance := 0
	for _, s := range allSessions {
		if s.Status == domain.SessionStatusCompleted {
			totalBalance += s.CreditsEarned
		}
	}

	h.hub.Broadcast <- &domain.WSEvent{
		Type: domain.EventBalanceUpdated,
		Payload: map[string]interface{}{
			"balance": totalBalance,
		},
	}

	return c.NoContent(http.StatusNoContent)
}

// IndexHandler renders the dashboard with activity data.
func (h *Handlers) IndexHandler(c echo.Context) error {
	activities, err := h.store.FindAllActivities()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Calculate current balance from completed sessions
	allSessions, _ := h.store.FindAllSessions()
	totalBalance := 0
	for _, s := range allSessions {
		if s.Status == domain.SessionStatusCompleted {
			totalBalance += s.CreditsEarned
		}
	}

	balanceStr := "0"
	if totalBalance > 0 {
		balanceStr = formatBalance(totalBalance)
	}

	component := templates.Dashboard(activities, balanceStr)
	return Render(c, http.StatusOK, component)
}

// formatBalance adds comma separators to an integer.
func formatBalance(n int) string {
	if n < 0 {
		return "-" + formatBalance(-n)
	}
	if n < 1000 {
		return itoa(n)
	}
	return formatBalance(n/1000) + "," + padLeft(itoa(n%1000), 3)
}

func itoa(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func padLeft(s string, l int) string {
	for len(s) < l {
		s = "0" + s
	}
	return s
}

// Render is a helper that renders a templ component into an Echo response.
func Render(c echo.Context, statusCode int, t templ.Component) error {
	c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().Writer.WriteHeader(statusCode)
	return t.Render(c.Request().Context(), c.Response().Writer)
}
