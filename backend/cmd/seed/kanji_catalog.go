package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/learnthor/backend/models"
)

//go:embed data/catalog/kanji.json
var bundledKanjiCatalog []byte

type kanjiCatalog struct {
	SchemaVersion int                `json:"schema_version"`
	Source        string             `json:"source"`
	Entries       []kanjiCatalogItem `json:"entries"`
}

type kanjiCatalogItem struct {
	ID                string   `json:"id"`
	Character         string   `json:"character"`
	JLPTLevel         string   `json:"jlpt_level"`
	SortOrder         int      `json:"sort_order"`
	StrokeCount       int      `json:"stroke_count"`
	Grade             int      `json:"grade"`
	Frequency         int      `json:"frequency"`
	Radical           string   `json:"radical"`
	OnReadings        []string `json:"on_readings"`
	KunReadings       []string `json:"kun_readings"`
	EnglishMeaning    string   `json:"english_meaning"`
	IndonesianMeaning string   `json:"indonesian_meaning"`
	ExampleWords      []string `json:"example_words"`
	StrokeSVG         string   `json:"stroke_svg"`
}

func loadKanjiCatalog(path string) (kanjiCatalog, error) {
	raw := bundledKanjiCatalog
	if path != "" {
		var err error
		raw, err = os.ReadFile(path)
		if err != nil {
			return kanjiCatalog{}, err
		}
	}
	var catalog kanjiCatalog
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return catalog, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return catalog, fmt.Errorf("unexpected content after kanji catalog")
	}
	if err := catalog.validate(); err != nil {
		return catalog, err
	}
	return catalog, nil
}

func (c kanjiCatalog) validate() error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("unsupported kanji schema version %d", c.SchemaVersion)
	}
	if len(c.Entries) == 0 {
		return fmt.Errorf("kanji catalog has no entries")
	}
	ids := map[string]bool{}
	characters := map[string]bool{}
	orders := map[string]map[int]bool{"N5": {}, "N4": {}, "N3": {}}
	for index, entry := range c.Entries {
		label := fmt.Sprintf("kanji entry %d", index+1)
		if entry.ID == "" || ids[entry.ID] {
			return fmt.Errorf("%s has an empty or duplicate id", label)
		}
		ids[entry.ID] = true
		if utf8.RuneCountInString(entry.Character) != 1 || characters[entry.Character] {
			return fmt.Errorf("%s has an invalid or duplicate character", label)
		}
		characters[entry.Character] = true
		if _, ok := orders[entry.JLPTLevel]; !ok {
			return fmt.Errorf("%s has invalid JLPT level %q", label, entry.JLPTLevel)
		}
		if entry.SortOrder < 1 || orders[entry.JLPTLevel][entry.SortOrder] {
			return fmt.Errorf("%s has an invalid or duplicate level order", label)
		}
		orders[entry.JLPTLevel][entry.SortOrder] = true
		if entry.StrokeCount < 1 || strings.TrimSpace(entry.EnglishMeaning) == "" || strings.TrimSpace(entry.IndonesianMeaning) == "" {
			return fmt.Errorf("%s is missing strokes or bilingual meanings", label)
		}
		if !strings.HasPrefix(entry.StrokeSVG, "/kanji/") || !strings.HasSuffix(entry.StrokeSVG, ".svg") {
			return fmt.Errorf("%s has invalid stroke SVG path", label)
		}
	}
	return nil
}

func importKanjiCatalog(database *gorm.DB, catalog kanjiCatalog) error {
	if err := catalog.validate(); err != nil {
		return err
	}
	return database.Transaction(func(tx *gorm.DB) error {
		for _, entry := range catalog.Entries {
			row := models.KanjiEntry{
				SeedKey: "kanji:" + entry.ID, Character: entry.Character, JLPTLevel: entry.JLPTLevel,
				SortOrder: entry.SortOrder, StrokeCount: entry.StrokeCount, Grade: entry.Grade,
				Frequency: entry.Frequency, Radical: entry.Radical,
				OnReadings: strings.Join(entry.OnReadings, "|"), KunReadings: strings.Join(entry.KunReadings, "|"),
				EnglishMeaning: entry.EnglishMeaning, IndonesianMeaning: entry.IndonesianMeaning,
				ExampleWords: strings.Join(entry.ExampleWords, "|"), StrokeSVG: entry.StrokeSVG,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "seed_key"}},
				DoUpdates: clause.AssignmentColumns([]string{"character", "jlpt_level", "sort_order", "stroke_count", "grade", "frequency", "radical", "on_readings", "kun_readings", "english_meaning", "indonesian_meaning", "example_words", "stroke_svg", "updated_at"}),
			}).Create(&row).Error; err != nil {
				return fmt.Errorf("import %s: %w", entry.Character, err)
			}
		}
		return nil
	})
}
