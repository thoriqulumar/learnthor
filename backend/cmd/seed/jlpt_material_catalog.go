package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/learnthor/backend/models"
	"github.com/learnthor/backend/services"
)

//go:embed data/catalog/jlpt-materials.json
var bundledJLPTMaterialCatalog []byte

type jlptMaterialCatalog struct {
	SchemaVersion int                         `json:"schema_version"`
	Lessons       []jlptMaterialCatalogLesson `json:"lessons"`
}

type jlptMaterialCatalogLesson struct {
	ID                string                        `json:"id"`
	Level             string                        `json:"level"`
	Category          string                        `json:"category"`
	TitleEnglish      string                        `json:"title_en"`
	TitleIndonesian   string                        `json:"title_id"`
	SummaryEnglish    string                        `json:"summary_en"`
	SummaryIndonesian string                        `json:"summary_id"`
	SortOrder         int                           `json:"sort_order"`
	IsPremium         bool                          `json:"is_premium"`
	Content           []services.JLPTContentSection `json:"content"`
	Questions         []jlptMaterialCatalogQuestion `json:"questions"`
}

type jlptMaterialCatalogQuestion struct {
	ID                    string   `json:"id"`
	PromptEnglish         string   `json:"prompt_en"`
	PromptIndonesian      string   `json:"prompt_id"`
	Options               []string `json:"options"`
	CorrectIndex          int      `json:"correct_index"`
	ExplanationEnglish    string   `json:"explanation_en"`
	ExplanationIndonesian string   `json:"explanation_id"`
}

func loadJLPTMaterialCatalog(path string) (jlptMaterialCatalog, error) {
	raw := bundledJLPTMaterialCatalog
	if path != "" {
		var err error
		raw, err = os.ReadFile(path)
		if err != nil {
			return jlptMaterialCatalog{}, err
		}
	}
	var catalog jlptMaterialCatalog
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return catalog, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return catalog, fmt.Errorf("unexpected content after JLPT material catalog")
	}
	if err := catalog.validate(); err != nil {
		return catalog, err
	}
	return catalog, nil
}

func (c jlptMaterialCatalog) validate() error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("unsupported JLPT material schema version %d", c.SchemaVersion)
	}
	if len(c.Lessons) == 0 {
		return fmt.Errorf("JLPT material catalog has no lessons")
	}
	levels := map[string]bool{"N5": true, "N4": true, "N3": true}
	categories := map[string]bool{"particles": true, "grammar": true, "reading": true, "exam": true}
	lessonIDs := map[string]bool{}
	questionIDs := map[string]bool{}
	orders := map[string]bool{}
	for index, lesson := range c.Lessons {
		label := fmt.Sprintf("JLPT lesson %d", index+1)
		if strings.TrimSpace(lesson.ID) == "" || lessonIDs[lesson.ID] {
			return fmt.Errorf("%s has an empty or duplicate id", label)
		}
		lessonIDs[lesson.ID] = true
		if !levels[lesson.Level] || !categories[lesson.Category] {
			return fmt.Errorf("%s has invalid level or category", label)
		}
		orderKey := fmt.Sprintf("%s:%s:%d", lesson.Level, lesson.Category, lesson.SortOrder)
		if lesson.SortOrder < 1 || orders[orderKey] {
			return fmt.Errorf("%s has invalid or duplicate order", label)
		}
		orders[orderKey] = true
		if strings.TrimSpace(lesson.TitleEnglish) == "" || strings.TrimSpace(lesson.TitleIndonesian) == "" ||
			strings.TrimSpace(lesson.SummaryEnglish) == "" || strings.TrimSpace(lesson.SummaryIndonesian) == "" || len(lesson.Content) < 2 {
			return fmt.Errorf("%s is missing bilingual teaching content", label)
		}
		if len(lesson.Questions) < 3 {
			return fmt.Errorf("%s needs at least three questions", label)
		}
		for questionIndex, question := range lesson.Questions {
			if strings.TrimSpace(question.ID) == "" || questionIDs[question.ID] {
				return fmt.Errorf("%s question %d has an empty or duplicate id", label, questionIndex+1)
			}
			questionIDs[question.ID] = true
			if len(question.Options) < 2 || question.CorrectIndex < 0 || question.CorrectIndex >= len(question.Options) {
				return fmt.Errorf("%s question %d has invalid options", label, questionIndex+1)
			}
			if strings.TrimSpace(question.PromptEnglish) == "" || strings.TrimSpace(question.PromptIndonesian) == "" ||
				strings.TrimSpace(question.ExplanationEnglish) == "" || strings.TrimSpace(question.ExplanationIndonesian) == "" {
				return fmt.Errorf("%s question %d is missing bilingual text", label, questionIndex+1)
			}
		}
	}
	return nil
}

func importJLPTMaterialCatalog(database *gorm.DB, catalog jlptMaterialCatalog) error {
	if err := catalog.validate(); err != nil {
		return err
	}
	return database.Transaction(func(tx *gorm.DB) error {
		for _, lesson := range catalog.Lessons {
			contentJSON, err := json.Marshal(lesson.Content)
			if err != nil {
				return fmt.Errorf("encode lesson %s: %w", lesson.ID, err)
			}
			row := models.JLPTLesson{
				SeedKey: "jlpt-lesson:" + lesson.ID, Level: lesson.Level, Category: lesson.Category,
				TitleEnglish: lesson.TitleEnglish, TitleIndonesian: lesson.TitleIndonesian,
				SummaryEnglish: lesson.SummaryEnglish, SummaryIndonesian: lesson.SummaryIndonesian,
				ContentJSON: string(contentJSON), SortOrder: lesson.SortOrder, IsPremium: lesson.IsPremium,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "seed_key"}},
				DoUpdates: clause.AssignmentColumns([]string{"level", "category", "title_english", "title_indonesian", "summary_english", "summary_indonesian", "content_json", "sort_order", "is_premium", "updated_at"}),
			}).Create(&row).Error; err != nil {
				return fmt.Errorf("import JLPT lesson %s: %w", lesson.ID, err)
			}
			var persisted models.JLPTLesson
			if err := tx.Where("seed_key = ?", "jlpt-lesson:"+lesson.ID).First(&persisted).Error; err != nil {
				return fmt.Errorf("reload JLPT lesson %s: %w", lesson.ID, err)
			}
			row = persisted
			for questionIndex, question := range lesson.Questions {
				optionsJSON, err := json.Marshal(question.Options)
				if err != nil {
					return fmt.Errorf("encode question %s: %w", question.ID, err)
				}
				questionRow := models.JLPTQuestion{
					SeedKey: "jlpt-question:" + question.ID, LessonID: row.ID, SortOrder: questionIndex + 1,
					PromptEnglish: question.PromptEnglish, PromptIndonesian: question.PromptIndonesian,
					OptionsJSON: string(optionsJSON), CorrectIndex: question.CorrectIndex,
					ExplanationEnglish: question.ExplanationEnglish, ExplanationIndonesian: question.ExplanationIndonesian,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "seed_key"}},
					DoUpdates: clause.AssignmentColumns([]string{"lesson_id", "sort_order", "prompt_english", "prompt_indonesian", "options_json", "correct_index", "explanation_english", "explanation_indonesian", "updated_at"}),
				}).Create(&questionRow).Error; err != nil {
					return fmt.Errorf("import JLPT question %s: %w", question.ID, err)
				}
			}
		}
		return nil
	})
}
