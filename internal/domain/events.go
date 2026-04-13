package domain

// Event constants.
const (
	EventTimerStarted   = "TIMER_STARTED"
	EventTimerStopped   = "TIMER_STOPPED"
	EventBalanceUpdated = "BALANCE_UPDATED"
	EventMobileStatus   = "MOBILE_STATUS"
)

// WSEvent represents a standard structure for broadcasting events over WebSocket.
// UserID scopes the event to a specific user for isolated broadcasting.
type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	UserID  string      `json:"-"` // Not serialized to clients; used for server-side routing
}
