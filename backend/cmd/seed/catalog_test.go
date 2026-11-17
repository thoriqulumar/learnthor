package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/learnthor/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func seedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := database.AutoMigrate(&models.User{}, &models.Course{}, &models.Section{}, &models.Module{}, &models.VocabItem{}, &models.KanjiEntry{}, &models.JLPTLesson{}, &models.JLPTQuestion{}, &models.JLPTLessonProgress{}, &models.ModuleProgress{}, &models.SRSReview{}); err != nil {
		t.Fatal(err)
	}
	return database
}

func smallCatalog() catalog {
	return catalog{SchemaVersion: 1, Sections: []catalogSection{{ID: "test", Title: "Test", SortOrder: 7, Modules: []catalogModule{{ID: "test-module", Title: "Words", SortOrder: 1, PassThreshold: 90, Items: []catalogItem{
		{ID: "bridge", Kanji: "橋", Kana: "はし", EnglishMeaning: "bridge", Source: "test"},
		{ID: "chopsticks", Kanji: "箸", Kana: "はし", EnglishMeaning: "chopsticks", Source: "test"},
	}}}}}}
}

func TestCatalogValidation(t *testing.T) {
	if _, err := decodeCatalog(bundledCatalog); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*catalog)
	}{
		{"schema", func(c *catalog) { c.SchemaVersion = 2 }},
		{"duplicate-id", func(c *catalog) { c.Sections[0].Modules[0].Items[1].ID = "bridge" }},
		{"missing-reading", func(c *catalog) { c.Sections[0].Modules[0].Items[0].Kana = "" }},
		{"missing-meaning", func(c *catalog) { c.Sections[0].Modules[0].Items[0].EnglishMeaning = "" }},
		{"corruption", func(c *catalog) { c.Sections[0].Modules[0].Items[0].Kanji = "�" }},
		{"long-reading", func(c *catalog) { c.Sections[0].Modules[0].Items[0].Kana = strings.Repeat("あ", 201) }},
		{"threshold", func(c *catalog) { c.Sections[0].Modules[0].PassThreshold = 101 }},
		{"duplicate-order", func(c *catalog) {
			m := c.Sections[0].Modules[0]
			m.ID = "other"
			c.Sections[0].Modules = append(c.Sections[0].Modules, m)
		}},
		{"invalid-retired-module", func(c *catalog) { c.RetiredModules = []string{"Bad ID"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := smallCatalog()
			tc.change(&c)
			if c.validate() == nil {
				t.Fatal("invalid catalog accepted")
			}
		})
	}
	raw, _ := json.Marshal(smallCatalog())
	if _, err := decodeCatalog(append(raw, []byte(`{}`)...)); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	if _, err := decodeCatalog([]byte(strings.Replace(string(raw), `"schema_version":1`, `"schema_version":1,"typo":true`, 1))); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestPublishedCatalogUsesTopicModuleNames(t *testing.T) {
	c, err := decodeCatalog(bundledCatalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range c.Sections {
		if section.ID == "daily" {
			t.Fatal("daily source section is still published")
		}
		for _, module := range section.Modules {
			if strings.HasPrefix(module.Title, "Day ") || strings.Contains(module.Title, " Vocab — Page ") {
				t.Fatalf("source-oriented module is still published: %s", module.Title)
			}
			if !strings.HasPrefix(module.Title, "Core — ") && !strings.HasPrefix(module.Title, "Extended — ") {
				t.Fatalf("module does not use the published naming convention: %s", module.Title)
			}
		}
	}
}

func TestImportHidesRetiredContainersAndPreservesProgress(t *testing.T) {
	database := seedTestDB(t)
	c := smallCatalog()
	if err := importCurriculum(database, c, false); err != nil {
		t.Fatal(err)
	}
	var course models.Course
	if err := database.First(&course, "seed_key = ?", "course:learnthor").Error; err != nil {
		t.Fatal(err)
	}
	dailyKey := "section:daily"
	daily := models.Section{CourseID: course.ID, SeedKey: &dailyKey, Title: "Daily Vocab", SortOrder: 100}
	if err := database.Create(&daily).Error; err != nil {
		t.Fatal(err)
	}
	pageKey := "module:n3-page-1"
	page := models.Module{SectionID: daily.ID, SeedKey: &pageKey, Title: "Page 1", SortOrder: 1, PassThreshold: 90}
	if err := database.Create(&page).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Email: "retired@example.test", Username: "retired-test", PasswordHash: "test"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	progress := models.ModuleProgress{UserID: user.ID, ModuleID: page.ID, HighScore: 95}
	if err := database.Create(&progress).Error; err != nil {
		t.Fatal(err)
	}
	c.RetiredSections = []string{"daily"}
	c.RetiredModules = []string{"n3-page-1"}
	if err := importCurriculum(database, c, false); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&daily, "id = ?", daily.ID).Error; err != nil || !daily.IsHidden {
		t.Fatalf("retired section was not hidden: %+v %v", daily, err)
	}
	if err := database.First(&page, "id = ?", page.ID).Error; err != nil || !page.IsHidden {
		t.Fatalf("retired module was not hidden: %+v %v", page, err)
	}
	if err := database.First(&progress, "id = ?", progress.ID).Error; err != nil || progress.HighScore != 95 {
		t.Fatalf("retirement damaged progress: %+v %v", progress, err)
	}
}

func TestImportPreservesLegacyIDsAndProgress(t *testing.T) {
	database := seedTestDB(t)
	course := models.Course{Title: "JLPT N5 Masterclass"}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	section := models.Section{CourseID: course.ID, Title: "Test", SortOrder: 7}
	if err := database.Create(&section).Error; err != nil {
		t.Fatal(err)
	}
	module := models.Module{SectionID: section.ID, Title: "Words", SortOrder: 1, PassThreshold: 90}
	if err := database.Create(&module).Error; err != nil {
		t.Fatal(err)
	}
	vocab := models.VocabItem{ModuleID: module.ID, Kanji: strPtr("橋"), Kana: "はし", Romaji: "hashi", EnglishMeaning: "old", ExampleSentence: "obsolete"}
	if err := database.Create(&vocab).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Email: "seed@example.test", Username: "seed-test", PasswordHash: "test"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	progress := models.ModuleProgress{UserID: user.ID, ModuleID: module.ID, HighScore: 95}
	if err := database.Create(&progress).Error; err != nil {
		t.Fatal(err)
	}
	review := models.SRSReview{UserID: user.ID, VocabItemID: vocab.ID, Reps: 4}
	if err := database.Create(&review).Error; err != nil {
		t.Fatal(err)
	}
	c := smallCatalog()
	if err := importCurriculum(database, c, false); err != nil {
		t.Fatal(err)
	}
	var before int64
	database.Model(&models.VocabItem{}).Count(&before)
	c.Sections[0].Title = "Renamed section"
	c.Sections[0].Modules[0].Title = "Renamed module"
	c.Sections[0].Modules[0].Items[0].Kana = "はし・corrected"
	c.Sections[0].Modules[0].Items[0].EnglishMeaning = "corrected bridge"
	if err := importCurriculum(database, c, false); err != nil {
		t.Fatal(err)
	}
	var after int64
	database.Model(&models.VocabItem{}).Count(&after)
	if before != after {
		t.Fatalf("repeat import duplicated items: %d -> %d", before, after)
	}
	var got models.VocabItem
	if err := database.First(&got, "id = ?", vocab.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.EnglishMeaning != "corrected bridge" || got.Kana != "はし・corrected" || got.ExampleSentence != "" || got.SeedKey == nil {
		t.Fatalf("update not applied: %+v", got)
	}
	var words int64
	database.Model(&models.VocabItem{}).Where("module_id = ?", module.ID).Count(&words)
	if words != 2 {
		t.Fatalf("homophones merged: got %d words", words)
	}
	var savedProgress models.ModuleProgress
	if err := database.First(&savedProgress, "id = ?", progress.ID).Error; err != nil {
		t.Fatal(err)
	}
	if savedProgress.HighScore != 95 {
		t.Fatal("progress changed")
	}
	var savedReview models.SRSReview
	if err := database.First(&savedReview, "id = ?", review.ID).Error; err != nil {
		t.Fatal(err)
	}
	if savedReview.Reps != 4 || savedReview.VocabItemID != vocab.ID {
		t.Fatal("review state changed")
	}
	var savedModule models.Module
	if err := database.First(&savedModule, "id = ?", module.ID).Error; err != nil {
		t.Fatal(err)
	}
	if savedModule.Title != "Renamed module" {
		t.Fatal("module identity lost")
	}
	// Removing an item from the source must not delete its saved review history.
	c.Sections[0].Modules[0].Items = c.Sections[0].Modules[0].Items[1:]
	if err := importCurriculum(database, c, false); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&got, "id = ?", vocab.ID).Error; err != nil {
		t.Fatal("removed source item deleted from database")
	}
}

func TestImportRollsBackOnWriteFailure(t *testing.T) {
	database := seedTestDB(t)
	if err := database.Exec(`CREATE TRIGGER reject_bridge BEFORE INSERT ON vocab_items WHEN NEW.seed_key = 'vocab:bridge' BEGIN SELECT RAISE(ABORT, 'forced failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := importCurriculum(database, smallCatalog(), false); err == nil {
		t.Fatal("write failure ignored")
	}
	var count int64
	database.Model(&models.Course{}).Count(&count)
	if count != 0 {
		t.Fatal("partial import survived rollback")
	}
}

func TestAmbiguousLegacyRowsAbort(t *testing.T) {
	database := seedTestDB(t)
	for i := 0; i < 2; i++ {
		if err := database.Create(&models.Course{Title: "JLPT N5 Masterclass"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := importCurriculum(database, smallCatalog(), false); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestFullCatalogRepeatImport(t *testing.T) {
	database := seedTestDB(t)
	c, err := loadCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := importCurriculum(database, c, false); err != nil {
			t.Fatal(err)
		}
	}
	var keyed int64
	database.Model(&models.VocabItem{}).Where("seed_key LIKE ?", "vocab:%").Count(&keyed)
	expected := 0
	for _, s := range c.Sections {
		for _, m := range s.Modules {
			expected += len(m.Items)
		}
	}
	if keyed != int64(expected) {
		t.Fatalf("catalog lost or duplicated entries: want %d got %d", expected, keyed)
	}
}
