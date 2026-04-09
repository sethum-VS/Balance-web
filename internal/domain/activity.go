package domain

import "time"

// ActivityType represents whether an activity earns or consumes credits.
type ActivityType string

const (
	ActivityTypeTopUp  ActivityType = "top_up"
	ActivityTypeConsume ActivityType = "consume"
)

// Activity represents a configurable activity profile.
type Activity struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Type          ActivityType `json:"type"`
	CreditPerHour float64      `json:"credit_per_hour"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// ActivityRepository defines the interface for activity persistence.
type ActivityRepository interface {
	FindByID(id string) (*Activity, error)
	FindAll() ([]*Activity, error)
	Save(activity *Activity) error
	Delete(id string) error
}
