package models

import "github.com/google/uuid"

// JLPTLesson is a bilingual lesson for a JLPT skill such as grammar or reading.
type JLPTLesson struct {
	Base

	SeedKey           string `gorm:"type:varchar(200);uniqueIndex;not null" json:"-"`
	Level             string `gorm:"type:varchar(10);not null;index:idx_jlpt_lesson_order,priority:1" json:"level"`
	Category          string `gorm:"type:varchar(30);not null;index:idx_jlpt_lesson_order,priority:2" json:"category"`
	TitleEnglish      string `gorm:"type:varchar(200);not null" json:"title_en"`
	TitleIndonesian   string `gorm:"type:varchar(200);not null" json:"title_id"`
	SummaryEnglish    string `gorm:"type:text;not null" json:"summary_en"`
	SummaryIndonesian string `gorm:"type:text;not null" json:"summary_id"`
	ContentJSON       string `gorm:"type:text;not null" json:"-"`
	SortOrder         int    `gorm:"not null;index:idx_jlpt_lesson_order,priority:3" json:"sort_order"`
	IsPremium         bool   `gorm:"not null;default:false" json:"is_premium"`

	Questions []JLPTQuestion `gorm:"foreignKey:LessonID" json:"-"`
}

// JLPTQuestion stores a server-graded multiple-choice lesson question.
type JLPTQuestion struct {
	Base

	SeedKey               string    `gorm:"type:varchar(220);uniqueIndex;not null" json:"-"`
	LessonID              uuid.UUID `gorm:"type:uuid;not null;index" json:"lesson_id"`
	SortOrder             int       `gorm:"not null" json:"sort_order"`
	PromptEnglish         string    `gorm:"type:text;not null" json:"prompt_en"`
	PromptIndonesian      string    `gorm:"type:text;not null" json:"prompt_id"`
	OptionsJSON           string    `gorm:"type:text;not null" json:"-"`
	CorrectIndex          int       `gorm:"not null" json:"-"`
	ExplanationEnglish    string    `gorm:"type:text;not null" json:"-"`
	ExplanationIndonesian string    `gorm:"type:text;not null" json:"-"`

	Lesson *JLPTLesson `gorm:"foreignKey:LessonID" json:"-"`
}

// JLPTLessonProgress stores a learner's best result for a material quiz.
type JLPTLessonProgress struct {
	Base

	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_jlpt_progress_user_lesson" json:"user_id"`
	LessonID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_jlpt_progress_user_lesson" json:"lesson_id"`
	BestScore    int       `gorm:"not null;default:0" json:"best_score"`
	AttemptCount int       `gorm:"not null;default:0" json:"attempt_count"`

	User   *User       `gorm:"foreignKey:UserID" json:"-"`
	Lesson *JLPTLesson `gorm:"foreignKey:LessonID" json:"-"`
}
