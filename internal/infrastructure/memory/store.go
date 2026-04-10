package memory

import (
	"balance-web/internal/domain"
	"fmt"
	"sync"
	"time"
)

// Store provides an in-memory data store for activities and sessions.
type Store struct {
	activities map[string]*domain.ActivityProfile
	sessions   map[string]*domain.Session
	mu         sync.RWMutex
}

// NewStore creates and returns a new in-memory Store pre-seeded with mock data.
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
		s.mu.Lock()
		s.activities[m.ID] = m
		s.mu.Unlock()
	}
}

// ---------- Direct Activity Methods ----------

func (s *Store) FindActivityByID(id string) (*domain.ActivityProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	activity, ok := s.activities[id]
	if !ok {
		return nil, fmt.Errorf("activity profile not found: %s", id)
	}
	return activity, nil
}

func (s *Store) FindAllActivities() ([]*domain.ActivityProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.ActivityProfile, 0, len(s.activities))
	for _, a := range s.activities {
		result = append(result, a)
	}
	return result, nil
}

func (s *Store) SaveActivity(activity *domain.ActivityProfile) error {
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

// ---------- Direct Session Methods ----------

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

func (s *Store) FindSessionsByActivityProfileID(activityProfileID string) ([]*domain.Session, error) {
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

// ---------- Adapter Types for Domain Interfaces ----------

// ActivityRepoAdapter adapts Store to satisfy domain.ActivityProfileRepository.
type ActivityRepoAdapter struct {
	store *Store
}

// NewActivityRepoAdapter wraps a Store as an ActivityProfileRepository.
func NewActivityRepoAdapter(s *Store) *ActivityRepoAdapter {
	return &ActivityRepoAdapter{store: s}
}

func (a *ActivityRepoAdapter) FindByID(id string) (*domain.ActivityProfile, error) {
	return a.store.FindActivityByID(id)
}
func (a *ActivityRepoAdapter) FindAll() ([]*domain.ActivityProfile, error) {
	return a.store.FindAllActivities()
}
func (a *ActivityRepoAdapter) Save(ap *domain.ActivityProfile) error {
	return a.store.SaveActivity(ap)
}
func (a *ActivityRepoAdapter) Delete(id string) error {
	return a.store.DeleteActivity(id)
}

// SessionRepoAdapter adapts Store to satisfy domain.SessionRepository.
type SessionRepoAdapter struct {
	store *Store
}

// NewSessionRepoAdapter wraps a Store as a SessionRepository.
func NewSessionRepoAdapter(s *Store) *SessionRepoAdapter {
	return &SessionRepoAdapter{store: s}
}

func (a *SessionRepoAdapter) FindByID(id string) (*domain.Session, error) {
	return a.store.FindSessionByID(id)
}
func (a *SessionRepoAdapter) FindAll() ([]*domain.Session, error) {
	return a.store.FindAllSessions()
}
func (a *SessionRepoAdapter) FindByActivityProfileID(id string) ([]*domain.Session, error) {
	return a.store.FindSessionsByActivityProfileID(id)
}
func (a *SessionRepoAdapter) Save(s *domain.Session) error {
	return a.store.SaveSession(s)
}
func (a *SessionRepoAdapter) Delete(id string) error {
	return a.store.DeleteSession(id)
}
