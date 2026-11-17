package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/learnthor/backend/models"
	"gorm.io/gorm"
)

//go:embed data/catalog/curriculum.json
var bundledCatalog []byte

type catalog struct {
	SchemaVersion   int              `json:"schema_version"`
	RetiredSections []string         `json:"retired_sections,omitempty"`
	RetiredModules  []string         `json:"retired_modules,omitempty"`
	Sections        []catalogSection `json:"sections"`
}

type catalogSection struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	SortOrder int             `json:"sort_order"`
	IsPremium bool            `json:"is_premium"`
	Modules   []catalogModule `json:"modules"`
}

type catalogModule struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	SortOrder     int           `json:"sort_order"`
	PassThreshold int           `json:"pass_threshold"`
	Items         []catalogItem `json:"items"`
}

type catalogItem struct {
	ID                string `json:"id"`
	Kanji             string `json:"kanji"`
	Kana              string `json:"kana"`
	Romaji            string `json:"romaji"`
	EnglishMeaning    string `json:"english_meaning"`
	IndonesianMeaning string `json:"indonesian_meaning"`
	PartOfSpeech      string `json:"part_of_speech"`
	ExampleSentence   string `json:"example_sentence"`
	Source            string `json:"source"`
}

func loadCatalog(path string) (catalog, error) {
	raw := bundledCatalog
	if path != "" {
		var err error
		raw, err = os.ReadFile(path)
		if err != nil {
			return catalog{}, err
		}
	}
	return decodeCatalog(raw)
}

func decodeCatalog(raw []byte) (catalog, error) {
	var c catalog
	if !utf8.Valid(raw) {
		return c, fmt.Errorf("catalog is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return c, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return c, fmt.Errorf("unexpected content after catalog")
	}
	return c, c.validate()
}

var contentID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (c catalog) validate() error {
	if c.SchemaVersion != 1 || len(c.Sections) == 0 {
		return fmt.Errorf("expected schema_version 1 and nonempty sections")
	}
	ids := map[string]bool{}
	checkID := func(kind, id string) error {
		key := kind + ":" + id
		if !contentID.MatchString(id) || len(key) > 200 || ids[key] {
			return fmt.Errorf("invalid or duplicate %s ID %q", kind, id)
		}
		ids[key] = true
		return nil
	}
	sectionOrders := map[int]bool{}
	retiredSectionIDs := map[string]bool{}
	for _, id := range c.RetiredSections {
		if !contentID.MatchString(id) || retiredSectionIDs[id] {
			return fmt.Errorf("invalid retired section ID %q", id)
		}
		retiredSectionIDs[id] = true
	}
	retiredModuleIDs := map[string]bool{}
	for _, id := range c.RetiredModules {
		if !contentID.MatchString(id) || retiredModuleIDs[id] {
			return fmt.Errorf("invalid retired module ID %q", id)
		}
		retiredModuleIDs[id] = true
	}
	for _, s := range c.Sections {
		if retiredSectionIDs[s.ID] {
			return fmt.Errorf("section %s cannot be active and retired", s.ID)
		}
		if err := checkID("section", s.ID); err != nil {
			return err
		}
		if err := validText(s.Title, 200, true); err != nil {
			return fmt.Errorf("section %s title: %w", s.ID, err)
		}
		if s.SortOrder < 7 || sectionOrders[s.SortOrder] || len(s.Modules) == 0 {
			return fmt.Errorf("section %s: duplicate/reserved order or empty modules", s.ID)
		}
		sectionOrders[s.SortOrder] = true
		orders := map[int]bool{}
		for _, m := range s.Modules {
			if retiredModuleIDs[m.ID] {
				return fmt.Errorf("module %s cannot be active and retired", m.ID)
			}
			if err := checkID("module", m.ID); err != nil {
				return err
			}
			if err := validText(m.Title, 200, true); err != nil {
				return fmt.Errorf("module %s title: %w", m.ID, err)
			}
			if m.SortOrder < 1 || orders[m.SortOrder] || m.PassThreshold < 1 || m.PassThreshold > 100 || len(m.Items) == 0 {
				return fmt.Errorf("module %s: invalid order, threshold, or empty items", m.ID)
			}
			orders[m.SortOrder] = true
			for _, v := range m.Items {
				if err := checkID("vocab", v.ID); err != nil {
					return err
				}
				if v.EnglishMeaning == "" && v.IndonesianMeaning == "" {
					return fmt.Errorf("vocab %s: at least one meaning is required", v.ID)
				}
				fields := []struct {
					name, value string
					max         int
					required    bool
				}{
					{"kanji", v.Kanji, 100, false}, {"kana", v.Kana, 200, true}, {"romaji", v.Romaji, 200, false},
					{"english_meaning", v.EnglishMeaning, 500, false}, {"indonesian_meaning", v.IndonesianMeaning, 500, false},
					{"part_of_speech", v.PartOfSpeech, 50, false}, {"example_sentence", v.ExampleSentence, 0, false}, {"source", v.Source, 0, true},
				}
				for _, f := range fields {
					if err := validText(f.value, f.max, f.required); err != nil {
						return fmt.Errorf("vocab %s %s: %w", v.ID, f.name, err)
					}
				}
			}
		}
	}
	return nil
}

func validText(value string, max int, required bool) error {
	if (required && value == "") || strings.TrimSpace(value) != value || strings.ContainsRune(value, '\uFFFD') || (max > 0 && utf8.RuneCountInString(value) > max) {
		return fmt.Errorf("empty, untrimmed, corrupted, or exceeds %d characters", max)
	}
	return nil
}

func seedCatalog(database *gorm.DB, course models.Course, c catalog) {
	if len(c.RetiredSections) > 0 {
		keys := make([]string, len(c.RetiredSections))
		for i, id := range c.RetiredSections {
			keys[i] = "section:" + id
		}
		mustSeed(database.Model(&models.Section{}).Where("seed_key IN ?", keys).Update("is_hidden", true).Error)
	}
	if len(c.RetiredModules) > 0 {
		keys := make([]string, len(c.RetiredModules))
		for i, id := range c.RetiredModules {
			keys[i] = "module:" + id
		}
		mustSeed(database.Model(&models.Module{}).Where("seed_key IN ?", keys).Update("is_hidden", true).Error)
	}
	for _, s := range c.Sections {
		sec := upsertSection(database, course.ID, models.Section{SeedKey: strPtr("section:" + s.ID), Title: s.Title, SortOrder: s.SortOrder, IsPremium: s.IsPremium, IsHidden: false})
		for _, m := range s.Modules {
			mod := upsertModule(database, sec.ID, models.Module{SeedKey: strPtr("module:" + m.ID), Title: m.Title, SortOrder: m.SortOrder, PassThreshold: m.PassThreshold, IsHidden: false})
			for _, v := range m.Items {
				romaji := v.Romaji
				if romaji == "" {
					romaji = KanaToRomaji(v.Kana)
				}
				upsertVocabItem(database, models.VocabItem{SeedKey: strPtr("vocab:" + v.ID), ModuleID: mod.ID, Kanji: optionalText(v.Kanji), Kana: v.Kana, Romaji: romaji, EnglishMeaning: v.EnglishMeaning, IndonesianMeaning: optionalText(v.IndonesianMeaning), PartOfSpeech: v.PartOfSpeech, ExampleSentence: v.ExampleSentence})
			}
		}
	}
}

func optionalText(value string) *string {
	if value == "" {
		return nil
	}
	return strPtr(value)
}
