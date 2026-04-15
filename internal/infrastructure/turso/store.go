package turso

import (
	"database/sql"
	"log"
	"os"

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

	_, err = db.Exec(createActivityTable)
	if err != nil {
		log.Fatalf("failed to create activity_profiles table: %v", err)
	}

	_, err = db.Exec(createSessionTable)
	if err != nil {
		log.Fatalf("failed to create sessions table: %v", err)
	}

	log.Println("Turso migration complete: tables ready with user_id scoping")
	return &Store{DB: db}
}
