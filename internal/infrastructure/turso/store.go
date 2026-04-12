package turso

import (
	"database/sql"
	"log"
	"os"
	"time"

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

	// Auto-Migration
	createActivityTable := `
	CREATE TABLE IF NOT EXISTS activity_profiles (
		id TEXT PRIMARY KEY,
		name TEXT,
		category TEXT,
		icon_name TEXT,
		credit_per_hour REAL,
		created_at DATETIME,
		updated_at DATETIME
	);`

	createSessionTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		activity_profile_id TEXT,
		status TEXT,
		start_time DATETIME,
		end_time DATETIME,
		duration INTEGER,
		credits_earned INTEGER,
		FOREIGN KEY (activity_profile_id) REFERENCES activity_profiles(id)
	);`

	_, err = db.Exec(createActivityTable)
	if err != nil {
		log.Fatalf("failed to create activity_profiles table: %v", err)
	}

	_, err = db.Exec(createSessionTable)
	if err != nil {
		log.Fatalf("failed to create sessions table: %v", err)
	}

	store := &Store{DB: db}
	store.SeedStore()
	return store
}

func (s *Store) SeedStore() {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM activity_profiles").Scan(&count)
	if err == nil && count == 0 {
		log.Println("Seeding Mock Activities into Turso db...")
		repo := NewActivityRepoAdapter(s)
		now := time.Now()
		
		mocks := []struct{ id, name, cat, icon string; cph float64 }{
			{"act_1", "Deep Work", "toppingUp", "desktopcomputer", 60.0},
			{"act_2", "Gym", "toppingUp", "figure.yoga", 30.0},
			{"act_3", "Social Media", "consuming", "iphone", 0},
			{"act_4", "Gaming", "consuming", "gamecontroller", 0},
		}
		
		for _, m := range mocks {
			repo.Save(&domain.ActivityProfile{
				ID: m.id, Name: m.name, Category: domain.ActivityCategory(m.cat),
				IconName: m.icon, CreditPerHour: m.cph, CreatedAt: now, UpdatedAt: now,
			})
		}
	}
}

func (s *Store) FindAllActivities() ([]*domain.ActivityProfile, error) {
	repo := NewActivityRepoAdapter(s)
	return repo.FindAll()
}

func (s *Store) FindActivityByID(id string) (*domain.ActivityProfile, error) {
	repo := NewActivityRepoAdapter(s)
	return repo.FindByID(id)
}

func (s *Store) SaveActivity(profile *domain.ActivityProfile) error {
	repo := NewActivityRepoAdapter(s)
	return repo.Save(profile)
}

func (s *Store) SaveSession(session *domain.Session) error {
	repo := NewSessionRepoAdapter(s)
	return repo.Save(session)
}
