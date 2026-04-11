package domain

// Event constants.
const (
	EventTimerStarted   = "TIMER_STARTED"
	EventTimerStopped   = "TIMER_STOPPED"
	EventBalanceUpdated = "BALANCE_UPDATED"
	EventMobileStatus   = "MOBILE_STATUS"
)

// WSEvent represents a standard structure for broadcasting events over WebSocket.
type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
