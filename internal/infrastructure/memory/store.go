package memory

import (
	"balance-web/internal/domain"
	"fmt"
	"sync"
)

// Store provides an in-memory implementation of the domain repositories.
// Intended for development and testing purposes.
type Store struct {
	activities map[string]*domain.Activity
	sessions   map[string]*domain.Session
	mu         sync.RWMutex
}

// NewStore creates and returns a new in-memory Store.
func NewStore() *Store {
	return &Store{
		activities: make(map[string]*domain.Activity),
		sessions:   make(map[string]*domain.Session),
	}
}

// --- ActivityRepository implementation ---

func (s *Store) FindActivityByID(id string) (*domain.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	activity, ok := s.activities[id]
	if !ok {
		return nil, fmt.Errorf("activity not found: %s", id)
	}
	return activity, nil
}

func (s *Store) FindAllActivities() ([]*domain.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.Activity, 0, len(s.activities))
	for _, a := range s.activities {
		result = append(result, a)
	}
	return result, nil
}

func (s *Store) SaveActivity(activity *domain.Activity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activities[activity.ID] = activity
	return nil
}

func (s *Store) DeleteActivity(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activities, id)
	return nil
}

// --- SessionRepository implementation ---

func (s *Store) FindSessionByID(id string) (*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

func (s *Store) FindAllSessions() ([]*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		result = append(result, sess)
	}
	return result, nil
}

func (s *Store) FindSessionsByActivityID(activityID string) ([]*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.Session, 0)
	for _, sess := range s.sessions {
		if sess.ActivityID == activityID {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (s *Store) SaveSession(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}
