package turso

import (
	"database/sql"
	"time"

	"balance-web/internal/domain"
)

// ActivityRepoAdapter implements the domain.ActivityProfileRepository using Turso libSQL.
type ActivityRepoAdapter struct {
	db *sql.DB
}

// NewActivityRepoAdapter creates a new SQL-backed activity repository.
func NewActivityRepoAdapter(store *Store) *ActivityRepoAdapter {
	return &ActivityRepoAdapter{db: store.DB}
}

func (r *ActivityRepoAdapter) FindByID(userID, id string) (*domain.ActivityProfile, error) {
	row := r.db.QueryRow("SELECT id, user_id, name, category, icon_name, credit_per_hour, created_at, updated_at FROM activity_profiles WHERE user_id = ? AND id = ?", userID, id)
	
	var a domain.ActivityProfile
	var createdAtStr, updatedAtStr string
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.IconName, &a.CreditPerHour, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Found nothing, mimic memory map behavior
		}
		return nil, err
	}

	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &a, nil
}

func (r *ActivityRepoAdapter) FindAll(userID string) ([]*domain.ActivityProfile, error) {
	rows, err := r.db.Query("SELECT id, user_id, name, category, icon_name, credit_per_hour, created_at, updated_at FROM activity_profiles WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*domain.ActivityProfile
	for rows.Next() {
		var a domain.ActivityProfile
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.IconName, &a.CreditPerHour, &createdAtStr, &updatedAtStr); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		activities = append(activities, &a)
	}
	return activities, nil
}

func (r *ActivityRepoAdapter) Save(userID string, a *domain.ActivityProfile) error {
	query := `
		INSERT INTO activity_profiles (id, user_id, name, category, icon_name, credit_per_hour, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, user_id) DO UPDATE SET 
			name=excluded.name, 
			category=excluded.category, 
			icon_name=excluded.icon_name, 
			credit_per_hour=excluded.credit_per_hour, 
			updated_at=excluded.updated_at
	`
	_, err := r.db.Exec(query, a.ID, userID, a.Name, string(a.Category), a.IconName, a.CreditPerHour, a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *ActivityRepoAdapter) Delete(userID, id string) error {
	_, err := r.db.Exec("DELETE FROM activity_profiles WHERE user_id = ? AND id = ?", userID, id)
	return err
}
