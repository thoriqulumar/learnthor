package models

import "github.com/google/uuid"

// ─────────────────────────────────────────────────────────────────────────────
// Course
// ─────────────────────────────────────────────────────────────────────────────

// Course is the top-level container for a learning path (e.g. "JLPT N5").
type Course struct {
	Base

	// SeedKey is the immutable import identity; nil for content created outside the importer.
	SeedKey *string `gorm:"type:varchar(200);uniqueIndex" json:"-"`

	Title       string `gorm:"type:varchar(200);not null" json:"title"`
	Description string `gorm:"type:text"                  json:"description"`
	// SortOrder controls display ordering among all courses.
	SortOrder int `gorm:"not null;default:0;index" json:"sort_order"`

	Sections []Section `gorm:"foreignKey:CourseID" json:"sections,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Section
// ─────────────────────────────────────────────────────────────────────────────

// Section groups related Modules within a Course (e.g. "Greetings & Basics").
type Section struct {
	Base

	// SeedKey is the immutable import identity; nil for content created outside the importer.
	SeedKey *string `gorm:"type:varchar(200);uniqueIndex" json:"-"`

	CourseID  uuid.UUID `gorm:"type:uuid;not null;index"   json:"course_id"`
	Title     string    `gorm:"type:varchar(200);not null" json:"title"`
	SortOrder int       `gorm:"not null;default:0"         json:"sort_order"`
	// IsPremium marks sections that require a premium subscription (N4, N3,
	// Daily Vocab). Free-tier users see these locked with an upgrade prompt.
	IsPremium bool `gorm:"not null;default:false" json:"is_premium"`
	// IsHidden keeps retired import containers out of the learning path without deleting history.
	IsHidden bool `gorm:"not null;default:false;index" json:"-"`

	Course  *Course  `gorm:"foreignKey:CourseID"  json:"course,omitempty"`
	Modules []Module `gorm:"foreignKey:SectionID" json:"modules,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Module
// ─────────────────────────────────────────────────────────────────────────────

// Module is the atomic unit of study that a user must pass to advance.
// A user must score >= 90 % (PassThreshold) on a quiz to unlock the next module.
type Module struct {
	Base

	// SeedKey is the immutable import identity; nil for content created outside the importer.
	SeedKey *string `gorm:"type:varchar(200);uniqueIndex" json:"-"`

	SectionID uuid.UUID `gorm:"type:uuid;not null;index"   json:"section_id"`
	Title     string    `gorm:"type:varchar(200);not null" json:"title"`
	// SortOrder determines linear unlock order within a Section.
	SortOrder int `gorm:"not null;default:0"       json:"sort_order"`
	// PassThreshold is the minimum score (0–100) required to unlock the
	// subsequent module. Stored as an integer percentage (e.g. 90 = 90 %).
	PassThreshold int `gorm:"not null;default:90"      json:"pass_threshold"`
	// IsHidden keeps source-only modules out of progression while preserving saved progress.
	IsHidden bool `gorm:"not null;default:false;index" json:"-"`

	Section    *Section    `gorm:"foreignKey:SectionID"  json:"section,omitempty"`
	VocabItems []VocabItem `gorm:"foreignKey:ModuleID"   json:"vocab_items,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// VocabItem
// ─────────────────────────────────────────────────────────────────────────────

// VocabItem represents a single Japanese vocabulary entry taught in a Module.
// It is also the entity tracked by the SRS engine.
//
// Indexes:
//   - module_id: for loading all vocab in a module.
//   - kanji:     for fast dictionary-style lookups (non-unique; same kanji
//     can appear in multiple modules at different difficulty levels).
type VocabItem struct {
	Base

	// SeedKey is the immutable import identity; nil for content created outside the importer.
	SeedKey *string `gorm:"type:varchar(200);uniqueIndex" json:"-"`

	ModuleID uuid.UUID `gorm:"type:uuid;not null;index"    json:"module_id"`
	// Kanji is the full written form, may include kanji characters.
	// Nullable because some words are kana-only (e.g. "ありがとう").
	Kanji *string `gorm:"type:varchar(100);index"      json:"kanji"`
	// Kana is the hiragana/katakana reading — always present.
	Kana string `gorm:"type:varchar(200);not null"   json:"kana"`
	// Romaji is the romanised reading for absolute beginners.
	Romaji string `gorm:"type:varchar(200);not null"   json:"romaji"`
	// EnglishMeaning is a comma-separated list of English definitions.
	EnglishMeaning string `gorm:"type:varchar(500);not null"   json:"english_meaning"`
	// IndonesianMeaning is the Indonesian gloss, populated only for vocab
	// batches sourced from Indonesian datasets; nil for legacy rows.
	IndonesianMeaning *string `gorm:"type:varchar(500)" json:"indonesian_meaning,omitempty"`
	// PartOfSpeech is an optional grammatical category tag (noun, verb, etc.).
	PartOfSpeech string `gorm:"type:varchar(50)"             json:"part_of_speech,omitempty"`
	// ExampleSentence is an optional native-language example sentence.
	ExampleSentence string `gorm:"type:text"                    json:"example_sentence,omitempty"`

	Module     *Module     `gorm:"foreignKey:ModuleID"  json:"module,omitempty"`
	SRSReviews []SRSReview `gorm:"foreignKey:VocabItemID" json:"-"`
}
