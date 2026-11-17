package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/learnthor/backend/models"
)

// Connect opens a PostgreSQL connection using the DATABASE_URL environment
// variable and returns a configured *gorm.DB.
func Connect() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		// Disable auto-creating/updating created_at/updated_at via callbacks
		// so our Base model stays authoritative.
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// Migrate runs AutoMigrate for all application models in dependency order.
// It also ensures the pgcrypto extension is available for gen_random_uuid().
func Migrate(db *gorm.DB) error {
	// gen_random_uuid() requires pgcrypto (pre-Postgres 13) or is built-in
	// (Postgres 13+). This is a no-op on Postgres 13+.
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		return fmt.Errorf("could not enable pgcrypto: %w", err)
	}

	return db.AutoMigrate(
		// Auth
		&models.User{},
		// Curriculum — ordered parent → child so FK constraints resolve correctly
		&models.Course{},
		&models.Section{},
		&models.Module{},
		&models.VocabItem{},
		&models.KanjiEntry{},
		&models.JLPTLesson{},
		&models.JLPTQuestion{},
		// Progress / SRS
		&models.ModuleProgress{},
		&models.ModuleSkillProgress{},
		&models.SectionProgress{},
		&models.JLPTLessonProgress{},
		&models.SRSReview{},
		// Session loop (Knowt micro-progression layer)
		&models.StudySession{},
		&models.SessionItem{},
		// Payments
		&models.Payment{},
	)
}
