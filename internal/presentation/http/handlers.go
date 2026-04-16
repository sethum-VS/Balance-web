package httphandlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"balance-web/internal/application"
	"balance-web/internal/domain"
	"balance-web/internal/infrastructure/auth"
	"balance-web/internal/infrastructure/turso"
	"balance-web/internal/infrastructure/websocket"
	"balance-web/web/templates"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// Handlers holds dependencies for HTTP request handlers.
type Handlers struct {
	store        *turso.Store
	activityRepo *turso.ActivityRepoAdapter
	sessionRepo  *turso.SessionRepoAdapter
	timerService *application.TimerService
	hub          *websocket.Hub
	firebaseAuth *auth.FirebaseAuth
	// Track the active session per user
	activeSessions map[string]string // userID -> sessionID
}

// NewHandlers creates a new Handlers instance with all dependencies.
func NewHandlers(store *turso.Store, activityRepo *turso.ActivityRepoAdapter, sessionRepo *turso.SessionRepoAdapter, timerService *application.TimerService, hub *websocket.Hub, firebaseAuth *auth.FirebaseAuth) *Handlers {
	h := &Handlers{
		store:          store,
		activityRepo:   activityRepo,
		sessionRepo:    sessionRepo,
		timerService:   timerService,
		hub:            hub,
		firebaseAuth:   firebaseAuth,
		activeSessions: make(map[string]string),
	}

	// Register AutoKill listener to broadcast user-scoped WS messages
	h.timerService.OnAutoStop = func(userID string, session *domain.Session) {
		if activeID, ok := h.activeSessions[userID]; ok && activeID == session.ID {
			delete(h.activeSessions, userID)
		}
		log.Printf("[AutoKill] Session auto-stopped on 0 CR: user=%s id=%s duration=%ds", userID, session.ID, session.Duration)

		h.hub.Broadcast <- &domain.WSEvent{
			Type: domain.EventTimerStopped,
			Payload: map[string]interface{}{
				"sessionID":     session.ID,
				"duration":      session.Duration,
				"creditsEarned": session.CreditsEarned,
			},
			UserID: userID,
		}

		h.hub.Broadcast <- &domain.WSEvent{
			Type: domain.EventBalanceUpdated,
			Payload: map[string]interface{}{
				"balance": h.timerService.CalculateGlobalBalance(userID),
			},
			UserID: userID,
		}
	}

	return h
}

// RegisterRoutes registers all HTTP routes on the Echo instance.
func (h *Handlers) RegisterRoutes(e *echo.Echo) {
	// Public routes (no auth)
	e.GET("/login", h.LoginHandler)
	e.GET("/health", h.HealthHandler)

	// Session cookie exchange (must be unprotected — exchanging token for cookie)
	e.POST("/api/auth/session", h.CreateSession)
	e.POST("/api/auth/signout", h.SignOut)

	// Page routes protected by redirect-based auth
	pageAuth := PageAuthMiddleware(h.firebaseAuth)
	e.GET("/", h.IndexHandler, pageAuth)

	// Protected API routes (returns 401 JSON on failure)
	authMiddleware := FirebaseAuthMiddleware(h.firebaseAuth)

	api := e.Group("/api", authMiddleware)
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

// GetActivities returns all ActivityProfiles for the authenticated user.
func (h *Handlers) GetActivities(c echo.Context) error {
	userID := c.Get("user_id").(string)

	activities, err := h.activityRepo.FindAll(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, activities)
}

// StartTimer handles POST /api/timer/start?activityID=xxx
func (h *Handlers) StartTimer(c echo.Context) error {
	userID := c.Get("user_id").(string)

	activityID := c.QueryParam("activityID")
	if activityID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "activityID is required")
	}

	// Look up the activity
	activity, _ := h.activityRepo.FindByID(userID, activityID)
	activityName := ""
	activityCategory := ""
	if activity != nil {
		activityName = activity.Name
		activityCategory = string(activity.Category)
	}

	// Calculate the current global CR balance (base) at session start
	baseBalance := h.timerService.CalculateGlobalBalance(userID)

	// Pre-Flight Guard: Deny Consume activities if balance is zero or lower
	if activityCategory == string(domain.ActivityCategoryConsuming) && baseBalance <= 0 {
		c.Response().Header().Set("HX-Trigger", `{"showError": "Please top up the app first."}`)
		return c.NoContent(http.StatusNoContent)
	}

	// If there's already an active session for this user, stop it first
	if activeID, ok := h.activeSessions[userID]; ok && activeID != "" {
		_, err := h.timerService.StopSession(userID, activeID)
		if err != nil {
			log.Printf("Error auto-stopping previous session: %v", err)
		}
	}

	session, err := h.timerService.StartSession(userID, activityID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	h.activeSessions[userID] = session.ID

	// Broadcast user-scoped TIMER_STARTED event with baseBalance for client-side ticking
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
		UserID: userID,
	}

	return c.NoContent(http.StatusNoContent)
}

// StopTimer handles POST /api/timer/stop
func (h *Handlers) StopTimer(c echo.Context) error {
	userID := c.Get("user_id").(string)

	activeID, ok := h.activeSessions[userID]
	if !ok || activeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no active session")
	}

	session, err := h.timerService.StopSession(userID, activeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	delete(h.activeSessions, userID)

	log.Printf("[StopTimer] Session stopped: user=%s id=%s duration=%ds credits=%d",
		userID, session.ID, session.Duration, session.CreditsEarned)

	// Broadcast user-scoped TIMER_STOPPED
	h.hub.Broadcast <- &domain.WSEvent{
		Type: domain.EventTimerStopped,
		Payload: map[string]interface{}{
			"sessionID":     session.ID,
			"duration":      session.Duration,
			"creditsEarned": session.CreditsEarned,
		},
		UserID: userID,
	}

	// Immediately follow with user-scoped BALANCE_UPDATED
	totalBalance := h.timerService.CalculateGlobalBalance(userID)
	log.Printf("[StopTimer] Balance after stop: user=%s balance=%d CR", userID, totalBalance)

	h.hub.Broadcast <- &domain.WSEvent{
		Type: domain.EventBalanceUpdated,
		Payload: map[string]interface{}{
			"balance": totalBalance,
		},
		UserID: userID,
	}

	return c.NoContent(http.StatusNoContent)
}

// LoginHandler renders the login page with Firebase config.
func (h *Handlers) LoginHandler(c echo.Context) error {
	config := templates.FirebaseConfig{
		APIKey:            os.Getenv("FIREBASE_API_KEY"),
		AuthDomain:        os.Getenv("FIREBASE_AUTH_DOMAIN"),
		ProjectID:         os.Getenv("FIREBASE_PROJECT_ID"),
		StorageBucket:     os.Getenv("FIREBASE_STORAGE_BUCKET"),
		MessagingSenderID: os.Getenv("FIREBASE_MESSAGING_SENDER_ID"),
		AppID:             os.Getenv("FIREBASE_APP_ID"),
	}
	component := templates.LoginPage(config)
	return Render(c, http.StatusOK, component)
}

// CreateSession handles POST /api/auth/session — exchanges a Firebase ID token for an HttpOnly session cookie.
func (h *Handlers) CreateSession(c echo.Context) error {
	var req struct {
		IDToken string `json:"idToken"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Verify the token is valid
	_, err := h.firebaseAuth.VerifyToken(req.IDToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "invalid Firebase token",
		})
	}

	// Create HttpOnly session cookie
	cookie := new(http.Cookie)
	cookie.Name = "session_token"
	cookie.Value = req.IDToken
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.Path = "/"
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// SignOut handles POST /api/auth/signout — clears the HttpOnly session cookie.
func (h *Handlers) SignOut(c echo.Context) error {
	cookie := new(http.Cookie)
	cookie.Name = "session_token"
	cookie.Value = ""
	cookie.Path = "/"
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	cookie.HttpOnly = true
	cookie.Secure = true
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// IndexHandler renders the dashboard with user-scoped activity data.
func (h *Handlers) IndexHandler(c echo.Context) error {
	userID := c.Get("user_id").(string)

	activities, err := h.activityRepo.FindAll(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Calculate user-scoped balance
	totalBalance := h.timerService.CalculateGlobalBalance(userID)
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

	config := templates.FirebaseConfig{
		APIKey:            os.Getenv("FIREBASE_API_KEY"),
		AuthDomain:        os.Getenv("FIREBASE_AUTH_DOMAIN"),
		ProjectID:         os.Getenv("FIREBASE_PROJECT_ID"),
		StorageBucket:     os.Getenv("FIREBASE_STORAGE_BUCKET"),
		MessagingSenderID: os.Getenv("FIREBASE_MESSAGING_SENDER_ID"),
		AppID:             os.Getenv("FIREBASE_APP_ID"),
	}

	component := templates.Dashboard(activities, balanceStr, isMobileOnline, wsURL, config)
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
	userID := c.Get("user_id").(string)

	var payloads []SyncPayload
	if err := c.Bind(&payloads); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	for _, payload := range payloads {
		session := &domain.Session{
			ID:                fmt.Sprintf("sess_%d", payload.Timestamp.UnixNano()),
			UserID:            userID,
			ActivityProfileID: payload.ActivityID,
			Status:            domain.SessionStatusCompleted,
			StartTime:         payload.StartTime,
			Duration:          payload.Duration,
			CreditsEarned:     payload.CreditsEarned,
		}
		
		now := payload.Timestamp
		session.EndTime = &now

		if err := h.sessionRepo.Save(userID, session); err != nil {
			log.Printf("[SyncSessions] Failed to save session %s: %v", session.ID, err)
			continue
		}
	}

	totalBalance := h.timerService.CalculateGlobalBalance(userID)
	log.Printf("[SyncSessions] user=%s synced %d sessions. Balance: %d CR", userID, len(payloads), totalBalance)

	h.hub.Broadcast <- &domain.WSEvent{
		Type:    domain.EventBalanceUpdated,
		Payload: map[string]interface{}{"balance": totalBalance},
		UserID:  userID,
	}

	return c.NoContent(http.StatusOK)
}

// CreateActivity handles POST /api/activities
func (h *Handlers) CreateActivity(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var profile domain.ActivityProfile
	if err := c.Bind(&profile); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	profile.UserID = userID

	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now()
	}

	if err := h.activityRepo.Save(userID, &profile); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, profile)
}

// SyncActivities handles POST /api/activities/sync
func (h *Handlers) SyncActivities(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var profiles []domain.ActivityProfile
	if err := c.Bind(&profiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	for _, profile := range profiles {
		profile.UserID = userID
		if profile.CreatedAt.IsZero() {
			profile.CreatedAt = time.Now()
		}
		if profile.UpdatedAt.IsZero() {
			profile.UpdatedAt = time.Now()
		}
		if err := h.activityRepo.Save(userID, &profile); err != nil {
			log.Printf("[SyncActivities] Failed to save activity %s: %v", profile.ID, err)
		}
	}

	return c.NoContent(http.StatusOK)
}
