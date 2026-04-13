package domain

import "time"

// SessionStatus represents the state of a time-tracking session.
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// Session represents a time/credit tracking session tied to an activity profile.
type Session struct {
	ID                string        `json:"id"`
	UserID            string        `json:"user_id"`
	ActivityProfileID string        `json:"activity_profile_id"`
	Status            SessionStatus `json:"status"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           *time.Time    `json:"end_time,omitempty"`
	Duration          int           `json:"duration"` // Duration in seconds
	CreditsEarned     int           `json:"credits_earned"`
}

// SessionRepository defines the interface for session persistence.
type SessionRepository interface {
	FindByID(userID, id string) (*Session, error)
	FindAll(userID string) ([]*Session, error)
	FindByActivityProfileID(userID, activityProfileID string) ([]*Session, error)
	Save(userID string, session *Session) error
	Delete(userID, id string) error
	GetTotalBalance(userID string) int
}
