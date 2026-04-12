package httphandlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"balance-web/internal/application"
	"balance-web/internal/domain"
	"balance-web/internal/infrastructure/turso"
	"balance-web/internal/infrastructure/websocket"
	"balance-web/web/templates"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// Handlers holds dependencies for HTTP request handlers.
type Handlers struct {
	store        *turso.Store
	timerService *application.TimerService
	hub          *websocket.Hub
	// Track the active session globally (single-user for now)
	activeSessionID string
}

// NewHandlers creates a new Handlers instance with all dependencies.
func NewHandlers(store *turso.Store, timerService *application.TimerService, hub *websocket.Hub) *Handlers {
	h := &Handlers{
		store:        store,
		timerService: timerService,
		hub:          hub,
	}

	// Register AutoKill listener to broadcast WS messages seamlessly
	h.timerService.OnAutoStop = func(session *domain.Session) {
		if h.activeSessionID == session.ID {
			h.activeSessionID = ""
		}
		log.Printf("[AutoKill] Session auto-stopped on 0 CR: id=%s duration=%ds", session.ID, session.Duration)

		h.hub.Broadcast <- &domain.WSEvent{
			Type: domain.EventTimerStopped,
			Payload: map[string]interface{}{
				"sessionID":     session.ID,
				"duration":      session.Duration,
				"creditsEarned": session.CreditsEarned,
			},
		}

		h.hub.Broadcast <- &domain.WSEvent{
			Type: domain.EventBalanceUpdated,
			Payload: map[string]interface{}{
				"balance": h.timerService.CalculateGlobalBalance(),
			},
		}
	}

	return h
}

// RegisterRoutes registers all HTTP routes on the Echo instance.
func (h *Handlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/", h.IndexHandler)
	e.GET("/health", h.HealthHandler)

	// API routes
	api := e.Group("/api")
	api.GET("/activities", h.GetActivities)
	api.POST("/activities", h.CreateActivity)
	api.POST("/activities/sync", h.SyncActivities)
	api.POST("/timer/start", h.StartTimer)
	api.POST("/timer/stop", h.StopTimer)
	api.POST("/sync", h.SyncSessions)
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

	// Look up the activity
	activity, _ := h.store.FindActivityByID(activityID)
	activityName := ""
	activityCategory := ""
	if activity != nil {
		activityName = activity.Name
		activityCategory = string(activity.Category)
	}

	// Calculate the current global CR balance (base) at session start
	baseBalance := h.timerService.CalculateGlobalBalance()

	// Pre-Flight Guard: Deny Consume activities if balance is zero or lower
	if activityCategory == string(domain.ActivityCategoryConsuming) && baseBalance <= 0 {
		c.Response().Header().Set("HX-Trigger", `{"showError": "Please top up the app first."}`)
		return c.NoContent(http.StatusNoContent)
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

	log.Printf("[StopTimer] Session stopped: id=%s duration=%ds credits=%d",
		session.ID, session.Duration, session.CreditsEarned)

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
	totalBalance := h.timerService.CalculateGlobalBalance()
	log.Printf("[StopTimer] Global balance after stop: %d CR", totalBalance)

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
	totalBalance := h.timerService.CalculateGlobalBalance()

	balanceStr := "0"
	if totalBalance > 0 {
		balanceStr = formatBalance(totalBalance)
	}
	
	isMobileOnline := h.hub.IsMobileOnline()

	// Dynamic WS_URL for deployment flexibility
	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		wsURL = "auto" // frontend will auto-detect from window.location
	}

	component := templates.Dashboard(activities, balanceStr, isMobileOnline, wsURL)
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

type SyncPayload struct {
	ActivityID    string    `json:"activityID"`
	Duration      int       `json:"duration"`
	CreditsEarned int       `json:"creditsEarned"`
	StartTime     time.Time `json:"startTime"`
	Timestamp     time.Time `json:"timestamp"`
}

// SyncSessions handles POST /api/sync
func (h *Handlers) SyncSessions(c echo.Context) error {
	var payloads []SyncPayload
	if err := c.Bind(&payloads); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	for _, payload := range payloads {
		session := &domain.Session{
			ID:                fmt.Sprintf("sess_%d", payload.Timestamp.UnixNano()),
			ActivityProfileID: payload.ActivityID,
			Status:            domain.SessionStatusCompleted,
			StartTime:         payload.StartTime,
			Duration:          payload.Duration,
			CreditsEarned:     payload.CreditsEarned,
		}
		
		now := payload.Timestamp
		session.EndTime = &now

		if err := h.store.SaveSession(session); err != nil {
			log.Printf("[SyncSessions] Failed to save session %s: %v", session.ID, err)
			continue
		}
	}

	totalBalance := h.timerService.CalculateGlobalBalance()
	log.Printf("[SyncSessions] Sync processed %d sessions. New global balance: %d CR", len(payloads), totalBalance)

	h.hub.Broadcast <- &domain.WSEvent{
		Type:    domain.EventBalanceUpdated,
		Payload: map[string]interface{}{"balance": totalBalance},
	}

	return c.NoContent(http.StatusOK)
}

// CreateActivity handles POST /api/activities
func (h *Handlers) CreateActivity(c echo.Context) error {
	var profile domain.ActivityProfile
	if err := c.Bind(&profile); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now()
	}

	if err := h.store.SaveActivity(&profile); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, profile)
}

// SyncActivities handles POST /api/activities/sync
func (h *Handlers) SyncActivities(c echo.Context) error {
	var profiles []domain.ActivityProfile
	if err := c.Bind(&profiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	for _, profile := range profiles {
		if profile.CreatedAt.IsZero() {
			profile.CreatedAt = time.Now()
		}
		if profile.UpdatedAt.IsZero() {
			profile.UpdatedAt = time.Now()
		}
		if err := h.store.SaveActivity(&profile); err != nil {
			log.Printf("[SyncActivities] Failed to save activity %s: %v", profile.ID, err)
		}
	}

	return c.NoContent(http.StatusOK)
}
