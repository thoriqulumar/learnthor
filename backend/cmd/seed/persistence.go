package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/learnthor/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type seedFailure struct{ err error }

func mustSeed(err error) {
	if err != nil {
		panic(seedFailure{err})
	}
}

func importCurriculum(database *gorm.DB, c catalog, reset bool) (err error) {
	if err := c.validate(); err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(seedFailure); ok {
				err = failure.err
			} else {
				panic(recovered)
			}
		}
	}()
	return database.Transaction(func(tx *gorm.DB) error {
		// Serialize seed runs so legacy adoption and inserts cannot race.
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(73492160)").Error; err != nil {
				return err
			}
		}
		if reset {
			if err := resetCurriculum(tx); err != nil {
				return err
			}
		}
		seed(tx, c)
		return nil
	})
}

func saveSeedRecord[T any](database *gorm.DB, desired T, key, legacy string, args ...any) T {
	var matches []T
	mustSeed(database.Where("seed_key = ?", key).Find(&matches).Error)
	if len(matches) == 0 {
		mustSeed(database.Where("seed_key IS NULL").Where(legacy, args...).Limit(2).Find(&matches).Error)
	}
	if len(matches) > 1 {
		mustSeed(fmt.Errorf("ambiguous legacy match for %s; resolve duplicate database rows before importing", key))
	}
	if len(matches) == 0 {
		mustSeed(database.Omit(clause.Associations).Create(&desired).Error)
		return desired
	}
	// Explicit zero values let editors clear obsolete translations/examples.
	mustSeed(database.Model(&matches[0]).Select("*").Omit("id", "created_at", "updated_at", clause.Associations).Updates(&desired).Error)
	var result T
	mustSeed(database.Where("seed_key = ?", key).First(&result).Error)
	return result
}

func builtinKey(parts ...string) *string {
	key := "builtin"
	for _, part := range parts {
		key += ":" + part
	}
	return strPtr("builtin:" + uuid.NewSHA1(uuid.NameSpaceURL, []byte(key)).String())
}

func upsertCourse(database *gorm.DB, c models.Course) models.Course {
	if c.SeedKey == nil {
		c.SeedKey = strPtr("course:learnthor")
	}
	return saveSeedRecord(database, c, *c.SeedKey, "title = ?", c.Title)
}

func upsertSection(database *gorm.DB, courseID uuid.UUID, s models.Section) models.Section {
	s.CourseID = courseID
	if s.SeedKey == nil {
		s.SeedKey = builtinKey("section", s.Title)
	}
	return saveSeedRecord(database, s, *s.SeedKey, "course_id = ? AND title = ?", courseID, s.Title)
}

func upsertModule(database *gorm.DB, sectionID uuid.UUID, m models.Module) models.Module {
	m.SectionID = sectionID
	if m.SeedKey == nil {
		m.SeedKey = builtinKey("module", sectionID.String(), m.Title)
	}
	return saveSeedRecord(database, m, *m.SeedKey, "section_id = ? AND title = ?", sectionID, m.Title)
}

type vocabSeed struct {
	Kanji           *string
	Kana            string
	Romaji          string
	EnglishMeaning  string
	PartOfSpeech    string
	ExampleSentence string
}

func upsertVocabItem(database *gorm.DB, item models.VocabItem) {
	if item.PartOfSpeech == "character" && (item.Kanji == nil || *item.Kanji == "") {
		item.Kanji = strPtr(item.Romaji)
	}
	kanji := ""
	if item.Kanji != nil {
		kanji = *item.Kanji
	}
	if item.SeedKey == nil {
		item.SeedKey = builtinKey("vocab", item.ModuleID.String(), item.Kana, kanji)
	}
	legacy := "module_id = ? AND kana = ? AND COALESCE(kanji, '') = ?"
	if item.PartOfSpeech == "character" {
		legacy = "module_id = ? AND kana = ? AND (COALESCE(kanji, '') = ? OR COALESCE(kanji, '') = '')"
	}
	saveSeedRecord(database, item, *item.SeedKey, legacy, item.ModuleID, item.Kana, kanji)
}

func upsertVocabItems(database *gorm.DB, moduleID uuid.UUID, items []vocabSeed) {
	for _, v := range items {
		upsertVocabItem(database, models.VocabItem{ModuleID: moduleID, Kanji: v.Kanji, Kana: v.Kana, Romaji: v.Romaji, EnglishMeaning: v.EnglishMeaning, PartOfSpeech: v.PartOfSpeech, ExampleSentence: v.ExampleSentence})
	}
}

func reportCatalog(c catalog) {
	modules, items := 0, 0
	for _, s := range c.Sections {
		modules += len(s.Modules)
		for _, m := range s.Modules {
			items += len(m.Items)
		}
	}
	log.Printf("validated catalog: %d sections, %d modules, %d vocabulary entries (plus built-in basics)", len(c.Sections), modules, items)
}
