package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/learnthor/backend/models"
)

var validJLPTLevels = map[string]bool{"N5": true, "N4": true, "N3": true}

// JLPTExample is a bilingual example inside a lesson section.
type JLPTExample struct {
	Japanese   string `json:"japanese"`
	English    string `json:"english"`
	Indonesian string `json:"indonesian"`
}

// JLPTContentSection is one teaching block in a JLPT lesson.
type JLPTContentSection struct {
	HeadingEnglish    string        `json:"heading_en"`
	HeadingIndonesian string        `json:"heading_id"`
	BodyEnglish       string        `json:"body_en"`
	BodyIndonesian    string        `json:"body_id"`
	JapaneseText      string        `json:"japanese_text,omitempty"`
	Examples          []JLPTExample `json:"examples,omitempty"`
}

// JLPTLessonSummary is returned on the level dashboard.
type JLPTLessonSummary struct {
	ID                uuid.UUID `json:"id"`
	Level             string    `json:"level"`
	Category          string    `json:"category"`
	TitleEnglish      string    `json:"title_en"`
	TitleIndonesian   string    `json:"title_id"`
	SummaryEnglish    string    `json:"summary_en"`
	SummaryIndonesian string    `json:"summary_id"`
	SortOrder         int       `json:"sort_order"`
	IsPremium         bool      `json:"is_premium"`
	Locked            bool      `json:"locked"`
	BestScore         int       `json:"best_score"`
	AttemptCount      int       `json:"attempt_count"`
}

// JLPTQuestionView hides grading data until the learner finishes the quiz.
type JLPTQuestionView struct {
	ID               uuid.UUID `json:"id"`
	PromptEnglish    string    `json:"prompt_en"`
	PromptIndonesian string    `json:"prompt_id"`
	Options          []string  `json:"options"`
}

// JLPTLessonDetail contains lesson material and an ungraded quiz.
type JLPTLessonDetail struct {
	JLPTLessonSummary
	Content   []JLPTContentSection `json:"content"`
	Questions []JLPTQuestionView   `json:"questions"`
}

// JLPTQuizAnswer is one submitted answer.
type JLPTQuizAnswer struct {
	QuestionID    uuid.UUID `json:"question_id" binding:"required"`
	SelectedIndex int       `json:"selected_index"`
}

// JLPTQuizResult explains one graded answer.
type JLPTQuizResult struct {
	QuestionID            uuid.UUID `json:"question_id"`
	SelectedIndex         int       `json:"selected_index"`
	CorrectIndex          int       `json:"correct_index"`
	Correct               bool      `json:"correct"`
	ExplanationEnglish    string    `json:"explanation_en"`
	ExplanationIndonesian string    `json:"explanation_id"`
}

// JLPTQuizResponse is returned after server-side grading.
type JLPTQuizResponse struct {
	Score     int              `json:"score"`
	Correct   int              `json:"correct"`
	Total     int              `json:"total"`
	BestScore int              `json:"best_score"`
	Results   []JLPTQuizResult `json:"results"`
}

// JLPTMaterialService serves bilingual JLPT lessons and grades their quizzes.
type JLPTMaterialService struct {
	db *gorm.DB
}

// NewJLPTMaterialService constructs a JLPTMaterialService.
func NewJLPTMaterialService(db *gorm.DB) *JLPTMaterialService {
	return &JLPTMaterialService{db: db}
}

// List returns all lesson summaries for one JLPT level.
func (s *JLPTMaterialService) List(userID uuid.UUID, level string) ([]JLPTLessonSummary, error) {
	level = strings.ToUpper(strings.TrimSpace(level))
	if !validJLPTLevels[level] {
		return nil, ErrBadRequest
	}
	var user models.User
	if err := s.db.Select("subscription_tier", "role", "progression_bypass").First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}
	canUsePremium := user.SubscriptionTier == models.SubscriptionPremium || user.Role == models.UserRoleAdmin || user.Role == models.UserRoleEditor

	var lessons []models.JLPTLesson
	if err := s.db.Where("level = ?", level).Order("category asc, sort_order asc").Find(&lessons).Error; err != nil {
		return nil, fmt.Errorf("list JLPT lessons: %w", err)
	}
	var progress []models.JLPTLessonProgress
	if err := s.db.Where("user_id = ?", userID).Find(&progress).Error; err != nil {
		return nil, fmt.Errorf("list JLPT progress: %w", err)
	}
	progressByLesson := make(map[uuid.UUID]models.JLPTLessonProgress, len(progress))
	for _, item := range progress {
		progressByLesson[item.LessonID] = item
	}

	result := make([]JLPTLessonSummary, 0, len(lessons))
	for _, lesson := range lessons {
		item := progressByLesson[lesson.ID]
		result = append(result, lessonSummary(lesson, lesson.IsPremium && !canUsePremium, item))
	}
	return result, nil
}

// Get returns one accessible lesson with quiz answers withheld.
func (s *JLPTMaterialService) Get(userID, lessonID uuid.UUID) (JLPTLessonDetail, error) {
	lesson, progress, err := s.loadAccessibleLesson(userID, lessonID)
	if err != nil {
		return JLPTLessonDetail{}, err
	}
	var content []JLPTContentSection
	if err := json.Unmarshal([]byte(lesson.ContentJSON), &content); err != nil {
		return JLPTLessonDetail{}, fmt.Errorf("decode lesson content: %w", err)
	}
	questions := make([]JLPTQuestionView, 0, len(lesson.Questions))
	for _, question := range lesson.Questions {
		var options []string
		if err := json.Unmarshal([]byte(question.OptionsJSON), &options); err != nil {
			return JLPTLessonDetail{}, fmt.Errorf("decode question options: %w", err)
		}
		questions = append(questions, JLPTQuestionView{
			ID: question.ID, PromptEnglish: question.PromptEnglish,
			PromptIndonesian: question.PromptIndonesian, Options: options,
		})
	}
	return JLPTLessonDetail{
		JLPTLessonSummary: lessonSummary(lesson, false, progress),
		Content:           content, Questions: questions,
	}, nil
}

// Finish grades a lesson quiz and saves the learner's best score.
func (s *JLPTMaterialService) Finish(userID, lessonID uuid.UUID, answers []JLPTQuizAnswer) (JLPTQuizResponse, error) {
	lesson, progress, err := s.loadAccessibleLesson(userID, lessonID)
	if err != nil {
		return JLPTQuizResponse{}, err
	}
	if len(lesson.Questions) == 0 {
		return JLPTQuizResponse{}, ErrBadRequest
	}
	answerByQuestion := make(map[uuid.UUID]int, len(answers))
	for _, answer := range answers {
		if _, exists := answerByQuestion[answer.QuestionID]; exists || answer.SelectedIndex < -1 {
			return JLPTQuizResponse{}, ErrBadRequest
		}
		answerByQuestion[answer.QuestionID] = answer.SelectedIndex
	}

	correct := 0
	results := make([]JLPTQuizResult, 0, len(lesson.Questions))
	for _, question := range lesson.Questions {
		selected, answered := answerByQuestion[question.ID]
		if !answered {
			selected = -1
		}
		isCorrect := selected == question.CorrectIndex
		if isCorrect {
			correct++
		}
		results = append(results, JLPTQuizResult{
			QuestionID: question.ID, SelectedIndex: selected,
			CorrectIndex: question.CorrectIndex, Correct: isCorrect,
			ExplanationEnglish:    question.ExplanationEnglish,
			ExplanationIndonesian: question.ExplanationIndonesian,
		})
	}
	score := correct * 100 / len(lesson.Questions)
	best := progress.BestScore
	if score > best {
		best = score
	}
	row := models.JLPTLessonProgress{
		UserID: userID, LessonID: lessonID, BestScore: best,
		AttemptCount: progress.AttemptCount + 1,
	}
	if err := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "lesson_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"best_score": best, "attempt_count": progress.AttemptCount + 1, "updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(&row).Error; err != nil {
		return JLPTQuizResponse{}, fmt.Errorf("save JLPT progress: %w", err)
	}
	return JLPTQuizResponse{Score: score, Correct: correct, Total: len(lesson.Questions), BestScore: best, Results: results}, nil
}

func (s *JLPTMaterialService) loadAccessibleLesson(userID, lessonID uuid.UUID) (models.JLPTLesson, models.JLPTLessonProgress, error) {
	var lesson models.JLPTLesson
	if err := s.db.Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order asc") }).First(&lesson, "id = ?", lessonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return lesson, models.JLPTLessonProgress{}, ErrNotFound
		}
		return lesson, models.JLPTLessonProgress{}, fmt.Errorf("load JLPT lesson: %w", err)
	}
	if lesson.IsPremium {
		var user models.User
		if err := s.db.Select("subscription_tier", "role").First(&user, "id = ?", userID).Error; err != nil {
			return lesson, models.JLPTLessonProgress{}, fmt.Errorf("load user: %w", err)
		}
		if user.SubscriptionTier != models.SubscriptionPremium && user.Role != models.UserRoleAdmin && user.Role != models.UserRoleEditor {
			return lesson, models.JLPTLessonProgress{}, ErrForbidden
		}
	}
	var progress models.JLPTLessonProgress
	err := s.db.Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&progress).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return lesson, progress, fmt.Errorf("load JLPT progress: %w", err)
	}
	return lesson, progress, nil
}

func lessonSummary(lesson models.JLPTLesson, locked bool, progress models.JLPTLessonProgress) JLPTLessonSummary {
	return JLPTLessonSummary{
		ID: lesson.ID, Level: lesson.Level, Category: lesson.Category,
		TitleEnglish: lesson.TitleEnglish, TitleIndonesian: lesson.TitleIndonesian,
		SummaryEnglish: lesson.SummaryEnglish, SummaryIndonesian: lesson.SummaryIndonesian,
		SortOrder: lesson.SortOrder, IsPremium: lesson.IsPremium, Locked: locked,
		BestScore: progress.BestScore, AttemptCount: progress.AttemptCount,
	}
}
