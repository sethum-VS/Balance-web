package turso

import (
	"database/sql"
	"fmt"
	"strings"

	"balance-web/internal/domain"
)

// EnsureUserProvisioned creates a user row and seeds user-scoped activity_profiles on first login.
func EnsureUserProvisioned(db *sql.DB, userID string) error {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return fmt.Errorf("user_id is required")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("INSERT OR IGNORE INTO users (id) VALUES (?)", uid); err != nil {
		return err
	}

	var existingCount int
	if err := tx.QueryRow("SELECT COUNT(1) FROM activity_profiles WHERE user_id = ?", uid).Scan(&existingCount); err != nil {
		return err
	}

	if existingCount == 0 {
		cloneDefaultsQuery := `
			INSERT INTO activity_profiles (
				id,
				user_id,
				name,
				category,
				icon_name,
				credit_per_hour,
				created_at,
				updated_at
			)
			SELECT
				'act_' || lower(hex(randomblob(16))),
				?,
				name,
				category,
				icon,
				CASE
					WHEN id = 'default_deep_work' THEN 60
					WHEN id = 'default_gym' THEN 30
					WHEN id = 'default_reading' THEN 20
					WHEN id = 'default_meditation' THEN 15
					WHEN category = ? THEN 0
					ELSE 0
				END,
				strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
				strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
			FROM default_activities
		`

		if _, err := tx.Exec(cloneDefaultsQuery, uid, string(domain.ActivityCategoryConsuming)); err != nil {
			return err
		}
	}

	return tx.Commit()
}
