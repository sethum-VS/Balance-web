package domain

import "time"

// SessionStatus represents the state of a time-tracking session.
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// Session represents a time/credit tracking session tied to an activity.
type Session struct {
	ID            string        `json:"id"`
	ActivityID    string        `json:"activity_id"`
	Status        SessionStatus `json:"status"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
	CreditsEarned float64       `json:"credits_earned"`
}

// SessionRepository defines the interface for session persistence.
type SessionRepository interface {
	FindByID(id string) (*Session, error)
	FindAll() ([]*Session, error)
	FindByActivityID(activityID string) ([]*Session, error)
	Save(session *Session) error
	Delete(id string) error
}
