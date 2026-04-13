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

func (r *SessionRepoAdapter) FindByID(userID, id string) (*domain.Session, error) {
	row := r.db.QueryRow("SELECT id, user_id, activity_profile_id, status, start_time, end_time, duration, credits_earned FROM sessions WHERE user_id = ? AND id = ?", userID, id)
	return scanSession(row)
}

func (r *SessionRepoAdapter) FindAll(userID string) ([]*domain.Session, error) {
	rows, err := r.db.Query("SELECT id, user_id, activity_profile_id, status, start_time, end_time, duration, credits_earned FROM sessions WHERE user_id = ?", userID)
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

func (r *SessionRepoAdapter) FindByActivityProfileID(userID, activityProfileID string) ([]*domain.Session, error) {
	rows, err := r.db.Query("SELECT id, user_id, activity_profile_id, status, start_time, end_time, duration, credits_earned FROM sessions WHERE user_id = ? AND activity_profile_id = ?", userID, activityProfileID)
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

func (r *SessionRepoAdapter) Save(userID string, s *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, activity_profile_id, status, start_time, end_time, duration, credits_earned) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, user_id) DO UPDATE SET 
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
	
	_, err := r.db.Exec(query, s.ID, userID, s.ActivityProfileID, string(s.Status), s.StartTime.Format(time.RFC3339), endTimeStr, s.Duration, s.CreditsEarned)
	return err
}

func (r *SessionRepoAdapter) Delete(userID, id string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE user_id = ? AND id = ?", userID, id)
	return err
}

// GetTotalBalance implements the sum query scoped by user.
func (r *SessionRepoAdapter) GetTotalBalance(userID string) int {
	var total sql.NullInt64
	err := r.db.QueryRow("SELECT SUM(credits_earned) FROM sessions WHERE user_id = ?", userID).Scan(&total)
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

	err := row.Scan(&s.ID, &s.UserID, &s.ActivityProfileID, &s.Status, &startTimeStr, &endTimeStr, &s.Duration, &s.CreditsEarned)
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

	err := rows.Scan(&s.ID, &s.UserID, &s.ActivityProfileID, &s.Status, &startTimeStr, &endTimeStr, &s.Duration, &s.CreditsEarned)
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
