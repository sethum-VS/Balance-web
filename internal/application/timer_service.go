package application

import (
	"balance-web/internal/domain"
	"fmt"
	"log"
	"sync"
	"time"
)

// TimerService encapsulates the business logic for starting, stopping,
// and calculating time/credit sessions.
type TimerService struct {
	sessionRepo    domain.SessionRepository
	activityRepo   domain.ActivityProfileRepository
	autoStopTimers map[string]*time.Timer
	mu             sync.Mutex
	OnAutoStop     func(session *domain.Session)
}

// NewTimerService creates a new TimerService with the given repositories.
func NewTimerService(sr domain.SessionRepository, ar domain.ActivityProfileRepository) *TimerService {
	return &TimerService{
		sessionRepo:    sr,
		activityRepo:   ar,
		autoStopTimers: make(map[string]*time.Timer),
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

	// --- Auto-Kill Switch for Consuming Activities ---
	if activity.Category == domain.ActivityCategoryConsuming {
		currentBalance := s.CalculateGlobalBalance()
		if currentBalance > 0 {
			s.mu.Lock()
			s.autoStopTimers[session.ID] = time.AfterFunc(time.Duration(currentBalance)*time.Second, func() {
				// Stop the session automatically
				stoppedSess, err := s.StopSession(session.ID)
				if err == nil && s.OnAutoStop != nil {
					s.OnAutoStop(stoppedSess)
				}
			})
			s.mu.Unlock()
		}
	}

	return session, nil
}

// StopSession ends an active session and calculates earned credits.
// Rule: 1 Second = 1 CR. ToppingUp earns positive, Consuming earns negative.
func (s *TimerService) StopSession(sessionID string) (*domain.Session, error) {
	// Cancel any auto-stop timer associated
	s.mu.Lock()
	if timer, ok := s.autoStopTimers[sessionID]; ok {
		timer.Stop()
		delete(s.autoStopTimers, sessionID)
	}
	s.mu.Unlock()

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

	// Calculate exact duration in seconds
	durationSecs := int(time.Since(session.StartTime).Seconds())
	session.Duration = durationSecs

	// Determine credit sign based on activity category
	activity, err := s.activityRepo.FindByID(session.ActivityProfileID)
	if err == nil && activity != nil {
		if activity.Category == domain.ActivityCategoryToppingUp {
			session.CreditsEarned = durationSecs // +N CR
		} else if activity.Category == domain.ActivityCategoryConsuming {
			session.CreditsEarned = -durationSecs // -N CR
		}
	}

	log.Printf("[TimerService] StopSession: id=%s duration=%ds credits=%d category=%s",
		session.ID, session.Duration, session.CreditsEarned, activity.Category)

	// Persist the completed session with calculated credits
	if err := s.sessionRepo.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// CalculateGlobalBalance returns the exact current cumulative CR pool via SQL aggregation.
func (s *TimerService) CalculateGlobalBalance() int {
	return s.sessionRepo.GetTotalBalance()
}
