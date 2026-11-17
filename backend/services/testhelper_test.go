package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/learnthor/backend/models"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers — in-memory SQLite + curriculum fixtures
//
// These run the real GORM queries against a throwaway database so service
// tests exercise actual SQL (joins, upserts, transactions) instead of mocks.
// ─────────────────────────────────────────────────────────────────────────────

// newTestDB returns a fresh in-memory SQLite database with every model
// migrated. Each call gets its own uniquely-named shared-cache database so
// tests never see each other's rows (reusing a single ":memory:" DSN would
// share one global database across the whole test process). The single
// pooled connection keeps GORM on the same underlying database.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// NOTE: db.Migrate() is not reused here because it executes
	// CREATE EXTENSION pgcrypto, which SQLite does not understand.
	err = db.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Section{},
		&models.Module{},
		&models.VocabItem{},
		&models.KanjiEntry{},
		&models.JLPTLesson{},
		&models.JLPTQuestion{},
		&models.ModuleProgress{},
		&models.ModuleSkillProgress{},
		&models.SectionProgress{},
		&models.JLPTLessonProgress{},
		&models.SRSReview{},
		&models.StudySession{},
		&models.SessionItem{},
		&models.Payment{},
	)
	if err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

// createTestUser inserts a user with the given role and returns it.
func createTestUser(t *testing.T, db *gorm.DB, role models.UserRole) models.User {
	t.Helper()
	user := models.User{
		Username:     "user_" + uuid.NewString()[:8],
		Email:        uuid.NewString()[:8] + "@test.dev",
		PasswordHash: "x",
		Role:         role,
		IsActive:     true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user
}

// curriculumFixture bundles the rows created by seedCurriculum so tests can
// refer to them without re-querying.
type curriculumFixture struct {
	Course  models.Course
	Section models.Section
	Module  models.Module
	Items   []models.VocabItem
}

// seedCurriculum creates one course → section → module chain with n vocab
// items. Items get kanji 漢1…漢n, kana かな1…かなn, romaji, an English
// meaning, a noun POS tag, and an example sentence containing the kanji so
// all three quiz types are compatible by default.
func seedCurriculum(t *testing.T, db *gorm.DB, n int) curriculumFixture {
	t.Helper()

	course := models.Course{Title: "Test Course", SortOrder: 1}
	if err := db.Create(&course).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	section := models.Section{CourseID: course.ID, Title: "Test Section", SortOrder: 1}
	if err := db.Create(&section).Error; err != nil {
		t.Fatalf("create section: %v", err)
	}
	module := models.Module{SectionID: section.ID, Title: "Test Module", SortOrder: 1, PassThreshold: 90}
	if err := db.Create(&module).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	items := make([]models.VocabItem, n)
	for i := 0; i < n; i++ {
		kanji := "漢" + string(rune('1'+i))
		kana := "かな" + string(rune('1'+i))
		items[i] = models.VocabItem{
			ModuleID:        module.ID,
			Kanji:           &kanji,
			Kana:            kana,
			Romaji:          "kana" + string(rune('1'+i)),
			EnglishMeaning:  "meaning " + string(rune('1'+i)),
			PartOfSpeech:    "noun",
			ExampleSentence: "これは" + kanji + "です。",
		}
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("create vocab items: %v", err)
	}

	return curriculumFixture{Course: course, Section: section, Module: module, Items: items}
}

// createSession inserts an active study session for (user, module).
func createSession(t *testing.T, db *gorm.DB, userID, moduleID uuid.UUID) models.StudySession {
	t.Helper()
	s := models.StudySession{
		UserID:    userID,
		ModuleID:  moduleID,
		Status:    models.SessionStatusActive,
		BatchSize: defaultBatchSize,
		StartedAt: time.Now().UTC(),
	}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create study session: %v", err)
	}
	return s
}
