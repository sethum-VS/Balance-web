package domain

// Event constants.
const (
	EventTimerStarted   = "TIMER_STARTED"
	EventTimerStopped   = "TIMER_STOPPED"
	EventBalanceUpdated = "BALANCE_UPDATED"
)

// WSEvent represents a standard structure for broadcasting events over WebSocket.
type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
