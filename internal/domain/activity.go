package domain

import "time"

// ActivityCategory represents whether an activity earns or consumes credits.
type ActivityCategory string

const (
	ActivityCategoryToppingUp ActivityCategory = "toppingUp"
	ActivityCategoryConsuming ActivityCategory = "consuming"
)

// ActivityProfile represents a configurable activity profile.
type ActivityProfile struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Category      ActivityCategory `json:"category"`
	IconName      string           `json:"icon_name"`
	CreditPerHour float64          `json:"credit_per_hour,omitempty"` // Kept for reference across components
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// ActivityProfileRepository defines the interface for activity persistence.
type ActivityProfileRepository interface {
	FindByID(id string) (*ActivityProfile, error)
	FindAll() ([]*ActivityProfile, error)
	Save(activityProfile *ActivityProfile) error
	Delete(id string) error
}
