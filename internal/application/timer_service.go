package application

import (
	"balance-web/internal/domain"
	"fmt"
	"time"
)

// TimerService encapsulates the business logic for starting, stopping,
// and calculating time/credit sessions.
type TimerService struct {
	sessionRepo  domain.SessionRepository
	activityRepo domain.ActivityRepository
}

// NewTimerService creates a new TimerService with the given repositories.
func NewTimerService(sr domain.SessionRepository, ar domain.ActivityRepository) *TimerService {
	return &TimerService{
		sessionRepo:  sr,
		activityRepo: ar,
	}
}

// StartSession begins a new time-tracking session for the given activity.
func (s *TimerService) StartSession(activityID string) (*domain.Session, error) {
	activity, err := s.activityRepo.FindByID(activityID)
	if err != nil {
		return nil, fmt.Errorf("activity not found: %w", err)
	}
	if activity == nil {
		return nil, fmt.Errorf("activity with id %s does not exist", activityID)
	}

	session := &domain.Session{
		ID:         fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		ActivityID: activityID,
		Status:     domain.SessionStatusActive,
		StartTime:  time.Now(),
	}

	if err := s.sessionRepo.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// StopSession ends an active session and calculates earned credits.
func (s *TimerService) StopSession(sessionID string) (*domain.Session, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session with id %s does not exist", sessionID)
	}
	if session.Status != domain.SessionStatusActive {
		return nil, fmt.Errorf("session is not active")
	}

	now := time.Now()
	session.EndTime = &now
	session.Status = domain.SessionStatusCompleted

	// Calculate credits based on activity's credit-per-hour rate
	activity, err := s.activityRepo.FindByID(session.ActivityID)
	if err == nil && activity != nil {
		duration := now.Sub(session.StartTime).Hours()
		session.CreditsEarned = duration * activity.CreditPerHour
	}

	if err := s.sessionRepo.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}
