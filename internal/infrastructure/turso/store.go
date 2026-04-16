package turso

import (
	"database/sql"
	"log"
	"os"

	"balance-web/internal/domain"

	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// Store encapsulates the Turso libSQL database connection.
type Store struct {
	DB *sql.DB
}

// NewStore initializes a connection to Turso and runs auto-migrations.
func NewStore() *Store {
	_ = godotenv.Load()

	url := "libsql://forbalance-smw-dev.aws-ap-south-1.turso.io?authToken=" + os.Getenv("TURSO_AUTH_TOKEN")
	db, err := sql.Open("libsql", url)
	if err != nil {
		log.Fatalf("failed to open db %s: %v", url, err)
	}

	// Ping the DB to ensure connection is valid
	if err = db.Ping(); err != nil {
		log.Fatalf("failed to connect to db %s: %v", url, err)
	}

	// Auto-migration: ensure required tables exist.
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createDefaultActivitiesTable := `
	CREATE TABLE IF NOT EXISTS default_activities (
		id TEXT PRIMARY KEY,
		name TEXT,
		category TEXT,
		icon TEXT,
		color TEXT
	);`

	createActivityTable := `
	CREATE TABLE IF NOT EXISTS activity_profiles (
		id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		name TEXT,
		category TEXT,
		icon_name TEXT,
		credit_per_hour REAL,
		created_at DATETIME,
		updated_at DATETIME,
		PRIMARY KEY (id, user_id)
	);`

	createSessionTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		activity_profile_id TEXT,
		status TEXT,
		start_time DATETIME,
		end_time DATETIME,
		duration INTEGER,
		credits_earned INTEGER,
		PRIMARY KEY (id, user_id)
	);`

	_, err = db.Exec(createUsersTable)
	if err != nil {
		log.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(createDefaultActivitiesTable)
	if err != nil {
		log.Fatalf("failed to create default_activities table: %v", err)
	}

	_, err = db.Exec(createActivityTable)
	if err != nil {
		log.Fatalf("failed to create activity_profiles table: %v", err)
	}

	_, err = db.Exec(createSessionTable)
	if err != nil {
		log.Fatalf("failed to create sessions table: %v", err)
	}

	if err := seedDefaultActivities(db); err != nil {
		log.Fatalf("failed to seed default_activities table: %v", err)
	}

	log.Println("Turso migration complete: tables ready with user_id scoping")
	return &Store{DB: db}
}

func seedDefaultActivities(db *sql.DB) error {
	defaults := []struct {
		ID       string
		Name     string
		Category string
		Icon     string
		Color    string
	}{
		{ID: "default_deep_work", Name: "Deep Work", Category: string(domain.ActivityCategoryToppingUp), Icon: "laptopcomputer", Color: "Blue"},
		{ID: "default_gym", Name: "Gym", Category: string(domain.ActivityCategoryToppingUp), Icon: "dumbbell.fill", Color: "Orange"},
		{ID: "default_reading", Name: "Reading", Category: string(domain.ActivityCategoryToppingUp), Icon: "book.closed.fill", Color: "Purple"},
		{ID: "default_meditation", Name: "Meditation", Category: string(domain.ActivityCategoryToppingUp), Icon: "leaf.fill", Color: "Green"},
		{ID: "default_social_media", Name: "Social Media", Category: string(domain.ActivityCategoryConsuming), Icon: "iphone", Color: "Pink"},
		{ID: "default_gaming", Name: "Gaming", Category: string(domain.ActivityCategoryConsuming), Icon: "gamecontroller.fill", Color: "Red"},
		{ID: "default_netflix", Name: "Netflix", Category: string(domain.ActivityCategoryConsuming), Icon: "play.tv.fill", Color: "Red"},
		{ID: "default_youtube", Name: "YouTube", Category: string(domain.ActivityCategoryConsuming), Icon: "play.rectangle.fill", Color: "Red"},
	}

	query := `
		INSERT OR IGNORE INTO default_activities (id, name, category, icon, color)
		VALUES (?, ?, ?, ?, ?)
	`

	for _, d := range defaults {
		if _, err := db.Exec(query, d.ID, d.Name, d.Category, d.Icon, d.Color); err != nil {
			return err
		}
	}

	return nil
}
