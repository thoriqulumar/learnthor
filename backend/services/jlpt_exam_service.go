package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/learnthor/backend/models"
)

// JLPTExamQuestion includes a passage only when required by the question.
type JLPTExamQuestion struct {
	ID               uuid.UUID `json:"id"`
	Category         string    `json:"category"`
	PromptEnglish    string    `json:"prompt_en"`
	PromptIndonesian string    `json:"prompt_id"`
	Options          []string  `json:"options"`
	Passage          string    `json:"passage,omitempty"`
}

// JLPTExamPaper signs the exact questions and options without exposing answers.
type JLPTExamPaper struct {
	Level     string             `json:"level"`
	ExpiresAt int64              `json:"expires_at"`
	Questions []JLPTExamQuestion `json:"questions"`
	Token     string             `json:"token,omitempty"`
}

// GenerateExam creates an ordered vocabulary, grammar, and reading practice paper.
func (s *JLPTMaterialService) GenerateExam(userID uuid.UUID, level string) (JLPTExamPaper, error) {
	level = strings.ToUpper(level)
	if _, err := s.List(userID, level); err != nil {
		return JLPTExamPaper{}, err
	}
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return JLPTExamPaper{}, err
	}
	if level != "N5" && user.SubscriptionTier != models.SubscriptionPremium && user.Role != models.UserRoleAdmin && user.Role != models.UserRoleEditor {
		return JLPTExamPaper{}, ErrForbidden
	}
	paper := JLPTExamPaper{Level: level, ExpiresAt: time.Now().Add(3 * time.Hour).Unix(), Questions: []JLPTExamQuestion{}}
	var vocab []models.VocabItem
	if err := s.db.Joins("JOIN modules ON modules.id = vocab_items.module_id").Joins("JOIN sections ON sections.id = modules.section_id").
		Where("sections.is_hidden = ? AND modules.is_hidden = ? AND sections.title LIKE ?", false, false, "%"+level+"%").Find(&vocab).Error; err != nil {
		return paper, err
	}
	rand.Shuffle(len(vocab), func(i, j int) { vocab[i], vocab[j] = vocab[j], vocab[i] })
	seen := map[string]bool{}
	for _, item := range vocab {
		if len(paper.Questions) >= 10 {
			break
		}
		if item.Kanji == nil || *item.Kanji == "" || seen[*item.Kanji] {
			continue
		}
		seen[*item.Kanji] = true
		options := []string{item.Kana}
		for _, other := range vocab {
			duplicate := false
			for _, option := range options {
				if option == other.Kana {
					duplicate = true
				}
			}
			if !duplicate && other.Kana != "" {
				options = append(options, other.Kana)
			}
			if len(options) == 4 {
				break
			}
		}
		if len(options) < 4 {
			continue
		}
		rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
		paper.Questions = append(paper.Questions, JLPTExamQuestion{ID: item.ID, Category: "vocabulary", PromptEnglish: "Choose the reading: " + *item.Kanji, PromptIndonesian: "Pilih cara baca: " + *item.Kanji, Options: options})
	}
	var lessons []models.JLPTLesson
	if err := s.db.Preload("Questions").Where("level = ?", level).Order("sort_order asc").Find(&lessons).Error; err != nil {
		return paper, err
	}
	for _, category := range []string{"particles", "grammar", "exam", "reading"} {
		pool := []JLPTExamQuestion{}
		for _, lesson := range lessons {
			if lesson.Category != category {
				continue
			}
			var blocks []JLPTContentSection
			if err := json.Unmarshal([]byte(lesson.ContentJSON), &blocks); err != nil {
				return paper, err
			}
			passage := ""
			if category == "reading" {
				for _, block := range blocks {
					if block.JapaneseText != "" {
						passage += block.JapaneseText + "\n"
					}
				}
			}
			for _, q := range lesson.Questions {
				// Particle exams use sentence completion rather than explanation recall.
				if category == "particles" && !strings.Contains(q.PromptEnglish, "＿") {
					continue
				}
				options := []string{}
				if err := json.Unmarshal([]byte(q.OptionsJSON), &options); err != nil {
					return paper, err
				}
				rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
				pool = append(pool, JLPTExamQuestion{ID: q.ID, Category: category, PromptEnglish: q.PromptEnglish, PromptIndonesian: q.PromptIndonesian, Options: options, Passage: passage})
			}
		}
		rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		limit := len(pool)
		if category == "particles" && limit > 10 {
			limit = 10
		}
		paper.Questions = append(paper.Questions, pool[:limit]...)
	}
	if len(paper.Questions) == 0 {
		return paper, ErrNotFound
	}
	raw, err := json.Marshal(paper)
	if err != nil {
		return paper, err
	}
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		return paper, ErrForbidden
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(raw)
	mac.Write([]byte(userID.String()))
	paper.Token = base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return paper, nil
}

// FinishExam grades a signed practice paper without changing curriculum unlocks.
func (s *JLPTMaterialService) FinishExam(userID uuid.UUID, token string, answers []JLPTQuizAnswer) (JLPTQuizResponse, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return JLPTQuizResponse{}, ErrBadRequest
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return JLPTQuizResponse{}, ErrBadRequest
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return JLPTQuizResponse{}, ErrBadRequest
	}
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		return JLPTQuizResponse{}, ErrForbidden
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(raw)
	mac.Write([]byte(userID.String()))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return JLPTQuizResponse{}, ErrBadRequest
	}
	var paper JLPTExamPaper
	if err := json.Unmarshal(raw, &paper); err != nil || paper.ExpiresAt < time.Now().Unix() {
		return JLPTQuizResponse{}, ErrBadRequest
	}
	items, err := s.List(userID, paper.Level)
	if err != nil {
		return JLPTQuizResponse{}, err
	}
	for _, item := range items {
		if item.Locked {
			return JLPTQuizResponse{}, ErrForbidden
		}
	}
	selected := map[uuid.UUID]int{}
	allowed := map[uuid.UUID]int{}
	for _, q := range paper.Questions {
		allowed[q.ID] = len(q.Options)
	}
	for _, answer := range answers {
		count, ok := allowed[answer.QuestionID]
		if !ok || answer.SelectedIndex < -1 || answer.SelectedIndex >= count {
			return JLPTQuizResponse{}, ErrBadRequest
		}
		if _, duplicate := selected[answer.QuestionID]; duplicate {
			return JLPTQuizResponse{}, ErrBadRequest
		}
		selected[answer.QuestionID] = answer.SelectedIndex
	}
	result := JLPTQuizResponse{Total: len(paper.Questions), Results: []JLPTQuizResult{}}
	for _, q := range paper.Questions {
		correctText := ""
		explanationEn := ""
		explanationId := ""
		if q.Category == "vocabulary" {
			var item models.VocabItem
			if err := s.db.First(&item, "id = ?", q.ID).Error; err != nil {
				return result, err
			}
			correctText = item.Kana
			explanationEn = item.EnglishMeaning
			if item.IndonesianMeaning != nil {
				explanationId = *item.IndonesianMeaning
			}
		} else {
			var item models.JLPTQuestion
			if err := s.db.First(&item, "id = ?", q.ID).Error; err != nil {
				return result, err
			}
			var options []string
			if err := json.Unmarshal([]byte(item.OptionsJSON), &options); err != nil {
				return result, err
			}
			correctText = options[item.CorrectIndex]
			explanationEn = item.ExplanationEnglish
			explanationId = item.ExplanationIndonesian
		}
		correctIndex := -1
		for i, option := range q.Options {
			if option == correctText {
				correctIndex = i
			}
		}
		choice, ok := selected[q.ID]
		if !ok {
			choice = -1
		}
		correct := correctIndex >= 0 && choice == correctIndex
		if correct {
			result.Correct++
		}
		result.Results = append(result.Results, JLPTQuizResult{QuestionID: q.ID, SelectedIndex: choice, CorrectIndex: correctIndex, Correct: correct, ExplanationEnglish: explanationEn, ExplanationIndonesian: explanationId})
	}
	if result.Total > 0 {
		result.Score = result.Correct * 100 / result.Total
	}
	return result, nil
}
