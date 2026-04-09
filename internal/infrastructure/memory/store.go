package memory

import (
	"balance-web/internal/domain"
	"fmt"
	"sync"
	"time"
)

// Store provides an in-memory implementation of the domain repositories.
type Store struct {
	activities map[string]*domain.ActivityProfile
	sessions   map[string]*domain.Session
	mu         sync.RWMutex
}

// NewStore creates and returns a new in-memory Store.
func NewStore() *Store {
	store := &Store{
		activities: make(map[string]*domain.ActivityProfile),
		sessions:   make(map[string]*domain.Session),
	}
	store.SeedStore()
	return store
}

// SeedStore loads default mock data mirroring the mobile layout.
func (s *Store) SeedStore() {
	now := time.Now()
	mocks := []*domain.ActivityProfile{
		{
			ID:            "act_1",
			Name:          "Deep Work",
			Category:      domain.ActivityCategoryToppingUp,
			IconName:      "desktopcomputer",
			CreditPerHour: 60.0,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "act_2",
			Name:          "Gym",
			Category:      domain.ActivityCategoryToppingUp,
			IconName:      "figure.yoga",
			CreditPerHour: 30.0,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "act_3",
			Name:          "Social Media",
			Category:      domain.ActivityCategoryConsuming,
			IconName:      "iphone",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "act_4",
			Name:          "Gaming",
			Category:      domain.ActivityCategoryConsuming,
			IconName:      "gamecontroller",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	for _, m := range mocks {
		_ = s.SaveActivityProfile(m)
	}
}

// --- ActivityProfileRepository implementation ---

func (s *Store) FindByID(id string) (*domain.ActivityProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	activity, ok := s.activities[id]
	if !ok {
		return nil, fmt.Errorf("activity profile not found: %s", id)
	}
	return activity, nil
}

func (s *Store) FindAll() ([]*domain.ActivityProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.ActivityProfile, 0, len(s.activities))
	for _, a := range s.activities {
		result = append(result, a)
	}
	return result, nil
}

func (s *Store) SaveActivityProfile(activity *domain.ActivityProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activities[activity.ID] = activity
	return nil
}

func (s *Store) DeleteActivityProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activities, id)
	return nil
}

// --- SessionRepository implementation (FindAll, etc) ---

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

func (s *Store) FindByActivityProfileID(activityProfileID string) ([]*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.Session, 0)
	for _, sess := range s.sessions {
		if sess.ActivityProfileID == activityProfileID {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (s *Store) Save(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}
