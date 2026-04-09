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
	activityRepo domain.ActivityProfileRepository
}

// NewTimerService creates a new TimerService with the given repositories.
func NewTimerService(sr domain.SessionRepository, ar domain.ActivityProfileRepository) *TimerService {
	return &TimerService{
		sessionRepo:  sr,
		activityRepo: ar,
	}
}

// StartSession begins a new time-tracking session for the given activity profile.
func (s *TimerService) StartSession(activityProfileID string) (*domain.Session, error) {
	activity, err := s.activityRepo.FindByID(activityProfileID)
	if err != nil {
		return nil, fmt.Errorf("activity profile not found: %w", err)
	}
	if activity == nil {
		return nil, fmt.Errorf("activity profile with id %s does not exist", activityProfileID)
	}

	session := &domain.Session{
		ID:                fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		ActivityProfileID: activityProfileID,
		Status:            domain.SessionStatusActive,
		StartTime:         time.Now(),
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
	
	duration := now.Sub(session.StartTime).Seconds()
	session.Duration = int(duration)

	// Calculate credits based on activity profile's credit-per-hour rate
	activity, err := s.activityRepo.FindByID(session.ActivityProfileID)
	if err == nil && activity != nil {
		hours := duration / 3600.0
		session.CreditsEarned = int(hours * activity.CreditPerHour)
	}

	if err := s.sessionRepo.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}
