package services

import (
	"strings"

	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// KanjiResponse is the public bilingual representation of one kanji entry.
type KanjiResponse struct {
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

// KanjiService queries the shared kanji reference catalog.
type KanjiService struct {
	db *gorm.DB
}

// NewKanjiService constructs a KanjiService.
func NewKanjiService(db *gorm.DB) *KanjiService {
	return &KanjiService{db: db}
}

// Word returns all stored readings and meanings for an exact example word.
func (s *KanjiService) Word(word string) ([]models.VocabItem, error) {
	var items []models.VocabItem
	err := s.db.Where("kanji = ? OR kana = ?", word, word).Order("created_at asc").Find(&items).Error
	result := make([]models.VocabItem, 0, len(items))
	seen := make(map[string]int)
	for _, item := range items {
		key := strings.TrimSpace(item.Kana)
		if index, exists := seen[key]; exists {
			entry := &result[index]
			entry.EnglishMeaning = mergeWordMeanings(entry.EnglishMeaning, item.EnglishMeaning)
			if item.IndonesianMeaning != nil && strings.TrimSpace(*item.IndonesianMeaning) != "" {
				meaning := *item.IndonesianMeaning
				if entry.IndonesianMeaning != nil {
					meaning = mergeWordMeanings(*entry.IndonesianMeaning, meaning)
				}
				entry.IndonesianMeaning = &meaning
			}
			if entry.Romaji == "" {
				entry.Romaji = item.Romaji
			}
			if entry.PartOfSpeech == "" {
				entry.PartOfSpeech = item.PartOfSpeech
			}
			if entry.ExampleSentence == "" {
				entry.ExampleSentence = item.ExampleSentence
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, item)
	}
	return result, err
}

func mergeWordMeanings(values ...string) string {
	var meanings []string
	seen := make(map[string]bool)
	for _, value := range values {
		for _, meaning := range strings.Split(value, ";") {
			meaning = strings.TrimSpace(meaning)
			key := strings.ToLower(meaning)
			if meaning != "" && !seen[key] {
				seen[key] = true
				meanings = append(meanings, meaning)
			}
		}
	}
	return strings.Join(meanings, "; ")
}

// List returns kanji ordered from N5 through N3, optionally filtered by level or search text.
func (s *KanjiService) List(level, query string) ([]KanjiResponse, error) {
	statement := s.db.Model(&models.KanjiEntry{})
	if level != "" {
		statement = statement.Where("jlpt_level = ?", strings.ToUpper(level))
	}
	if query != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
		statement = statement.Where("character = ? OR LOWER(on_readings) LIKE ? OR LOWER(kun_readings) LIKE ? OR LOWER(english_meaning) LIKE ? OR LOWER(indonesian_meaning) LIKE ?", query, like, like, like, like)
	}
	var entries []models.KanjiEntry
	err := statement.Order("CASE jlpt_level WHEN 'N5' THEN 1 WHEN 'N4' THEN 2 ELSE 3 END").Order("sort_order asc").Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return kanjiResponses(entries), nil
}

// Get returns a single kanji entry by character.
func (s *KanjiService) Get(character string) (*KanjiResponse, error) {
	var entry models.KanjiEntry
	if err := s.db.Where("character = ?", character).First(&entry).Error; err != nil {
		return nil, err
	}
	response := kanjiResponse(entry)
	return &response, nil
}

func kanjiResponses(entries []models.KanjiEntry) []KanjiResponse {
	result := make([]KanjiResponse, 0, len(entries))
	for _, entry := range entries {
		result = append(result, kanjiResponse(entry))
	}
	return result
}

func kanjiResponse(entry models.KanjiEntry) KanjiResponse {
	return KanjiResponse{
		Character: entry.Character, JLPTLevel: entry.JLPTLevel, SortOrder: entry.SortOrder,
		StrokeCount: entry.StrokeCount, Grade: entry.Grade, Frequency: entry.Frequency,
		Radical: entry.Radical, OnReadings: splitCatalogField(entry.OnReadings),
		KunReadings: splitCatalogField(entry.KunReadings), EnglishMeaning: entry.EnglishMeaning,
		IndonesianMeaning: entry.IndonesianMeaning, ExampleWords: splitCatalogField(entry.ExampleWords),
		StrokeSVG: entry.StrokeSVG,
	}
}

func splitCatalogField(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(value, "|")
}
