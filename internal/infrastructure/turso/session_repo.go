package turso

import (
	"database/sql"
	"time"

	"balance-web/internal/domain"
)

// SessionRepoAdapter implements the domain.SessionRepository using Turso libSQL.
type SessionRepoAdapter struct {
	db *sql.DB
}

// NewSessionRepoAdapter creates a new SQL-backed session repository.
func NewSessionRepoAdapter(store *Store) *SessionRepoAdapter {
	return &SessionRepoAdapter{db: store.DB}
}

func (r *SessionRepoAdapter) FindByID(id string) (*domain.Session, error) {
	row := r.db.QueryRow("SELECT id, activity_profile_id, status, start_time, end_time, duration, credits_earned FROM sessions WHERE id = ?", id)
	return scanSession(row)
}

func (r *SessionRepoAdapter) FindAll() ([]*domain.Session, error) {
	rows, err := r.db.Query("SELECT id, activity_profile_id, status, start_time, end_time, duration, credits_earned FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionRepoAdapter) FindByActivityProfileID(activityProfileID string) ([]*domain.Session, error) {
	rows, err := r.db.Query("SELECT id, activity_profile_id, status, start_time, end_time, duration, credits_earned FROM sessions WHERE activity_profile_id = ?", activityProfileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionRepoAdapter) Save(s *domain.Session) error {
	query := `
		INSERT INTO sessions (id, activity_profile_id, status, start_time, end_time, duration, credits_earned) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			status=excluded.status, 
			end_time=excluded.end_time, 
			duration=excluded.duration, 
			credits_earned=excluded.credits_earned
	`
	
	// Handle optional end_time mapping
	var endTimeStr *string
	if s.EndTime != nil {
		str := s.EndTime.Format(time.RFC3339)
		endTimeStr = &str
	}
	
	_, err := r.db.Exec(query, s.ID, s.ActivityProfileID, string(s.Status), s.StartTime.Format(time.RFC3339), endTimeStr, s.Duration, s.CreditsEarned)
	return err
}

func (r *SessionRepoAdapter) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// GetTotalBalance implements the sum query.
func (r *SessionRepoAdapter) GetTotalBalance() int {
	var total sql.NullInt64
	err := r.db.QueryRow("SELECT SUM(credits_earned) FROM sessions").Scan(&total)
	if err != nil || !total.Valid {
		return 0
	}
	return int(total.Int64)
}

// Helpers
func scanSession(row *sql.Row) (*domain.Session, error) {
	var s domain.Session
	var startTimeStr string
	var endTimeStr sql.NullString

	err := row.Scan(&s.ID, &s.ActivityProfileID, &s.Status, &startTimeStr, &endTimeStr, &s.Duration, &s.CreditsEarned)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Mimic memory map
		}
		return nil, err
	}

	s.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
	if endTimeStr.Valid && endTimeStr.String != "" {
		t, _ := time.Parse(time.RFC3339, endTimeStr.String)
		s.EndTime = &t
	}

	return &s, nil
}

func scanSessionRow(rows *sql.Rows) (*domain.Session, error) {
	var s domain.Session
	var startTimeStr string
	var endTimeStr sql.NullString

	err := rows.Scan(&s.ID, &s.ActivityProfileID, &s.Status, &startTimeStr, &endTimeStr, &s.Duration, &s.CreditsEarned)
	if err != nil {
		return nil, err
	}

	s.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
	if endTimeStr.Valid && endTimeStr.String != "" {
		t, _ := time.Parse(time.RFC3339, endTimeStr.String)
		s.EndTime = &t
	}

	return &s, nil
}
