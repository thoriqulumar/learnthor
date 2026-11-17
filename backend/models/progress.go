package models

import (
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// ModuleProgress
// ─────────────────────────────────────────────────────────────────────────────

// ModuleSkillProgress tracks a user's best score for a specific quiz skill type
// within a module (e.g. kanji_to_kana, kana_to_kanji, fill_blank).
// Each (user, module, skill_type) triple is unique.
// The aggregate of all skill scores drives the module-level HighScore in ModuleProgress.
type ModuleSkillProgress struct {
	Base

	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_skill_prog_key"  json:"user_id"`
	ModuleID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_skill_prog_key"  json:"module_id"`
	SkillType string    `gorm:"not null;uniqueIndex:idx_skill_prog_key"            json:"skill_type"` // kanji_to_kana | kana_to_kanji | fill_blank
	HighScore int       `gorm:"not null;default:0"                                 json:"high_score"` // 0–100
}

// ─────────────────────────────────────────────────────────────────────────────
// ModuleProgress
// ─────────────────────────────────────────────────────────────────────────────

// ModuleProgress records a user's aggregate best score on a Module (average of
// all skill-type high scores) and whether the next module has been unlocked.
//
// Business rule: IsUnlocked is set to true when HighScore >= Module.PassThreshold.
//
// Indexes:
//   - (user_id, module_id): composite unique — one record per user/module pair.
//   - user_id:             for fetching all progress records for a dashboard.
type ModuleProgress struct {
	Base

	UserID   uuid.UUID `gorm:"type:uuid;not null;index:idx_module_progress_user_module,unique" json:"user_id"`
	ModuleID uuid.UUID `gorm:"type:uuid;not null;index:idx_module_progress_user_module,unique" json:"module_id"`

	// HighScore is the user's best recorded score (0–100) for this module.
	HighScore int `gorm:"not null;default:0" json:"high_score"`
	// IsUnlocked indicates whether the threshold has been met and the
	// subsequent module is accessible.
	IsUnlocked bool `gorm:"not null;default:false" json:"is_unlocked"`
	// AttemptCount is incremented on every quiz submission.
	AttemptCount int `gorm:"not null;default:0" json:"attempt_count"`
	// LastAttemptAt records when the user last attempted this module's quiz.
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`

	User   *User   `gorm:"foreignKey:UserID"   json:"user,omitempty"`
	Module *Module `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// SRSReview  (FSRS-based spaced-repetition state)
// ─────────────────────────────────────────────────────────────────────────────

// ReviewState mirrors the FSRS card states.
type ReviewState string

const (
	ReviewStateNew        ReviewState = "new"        // never reviewed
	ReviewStateLearning   ReviewState = "learning"   // in early learning phase
	ReviewStateReview     ReviewState = "review"     // mature card
	ReviewStateRelearning ReviewState = "relearning" // lapsed card
)

// Rating mirrors the FSRS answer ratings (1–4).
type Rating int

const (
	RatingAgain Rating = 1 // complete blackout
	RatingHard  Rating = 2 // incorrect but recalled with difficulty
	RatingGood  Rating = 3 // correct with some effort
	RatingEasy  Rating = 4 // perfect recall
)

// SRSReview stores the per-user, per-vocab FSRS scheduling state.
//
// FSRS parameters used here:
//   - Stability (S):   number of days until retention drops to 90 %.
//   - Difficulty (D):  intrinsic difficulty of the card, range [1, 10].
//   - Streak:          consecutive "Good" or "Easy" answers (custom metric
//     used for streak-based UI achievements, not a core FSRS parameter).
//
// Indexes:
//   - (user_id, vocab_item_id): composite unique — one SRS card per user/vocab.
//   - (user_id, next_review_at): for efficiently fetching a user's due cards.
//   - next_review_at:            for background jobs that send review reminders.
type SRSReview struct {
	Base

	UserID      uuid.UUID `gorm:"type:uuid;not null;index:idx_srs_user_vocab,unique;index:idx_srs_due" json:"user_id"`
	VocabItemID uuid.UUID `gorm:"type:uuid;not null;index:idx_srs_user_vocab,unique"                   json:"vocab_item_id"`

	// --- Core FSRS scheduling state ---

	// State is the current learning phase of the card.
	State ReviewState `gorm:"type:varchar(20);not null;default:'new'" json:"state"`
	// Stability is the FSRS S value in days (float for sub-day precision).
	// A higher value means the card is better retained and needs fewer reviews.
	Stability float64 `gorm:"type:numeric(10,4);not null;default:0" json:"stability"`
	// Difficulty is the FSRS D value [1.0, 10.0].
	// Higher = harder; cards that are often forgotten drift towards 10.
	Difficulty float64 `gorm:"type:numeric(5,4);not null;default:5" json:"difficulty"`
	// Retrievability is the estimated recall probability [0, 1] at LastReviewedAt.
	// Stored for analytics; recomputed on each review.
	Retrievability float64 `gorm:"type:numeric(5,4);not null;default:0" json:"retrievability"`
	// Reps is the total number of times this card has been reviewed.
	Reps int `gorm:"not null;default:0" json:"reps"`
	// Lapses is the number of times the card was rated "Again" after graduating.
	Lapses int `gorm:"not null;default:0" json:"lapses"`

	// --- Streak (UI / gamification metric) ---

	// Streak counts consecutive reviews rated Good (3) or Easy (4).
	// Resets to 0 on an "Again" or "Hard" answer.
	Streak int `gorm:"not null;default:0" json:"streak"`

	// --- Scheduling timestamps ---

	// LastReviewedAt is the wall-clock time of the most recent review.
	LastReviewedAt *time.Time `gorm:"index" json:"last_reviewed_at,omitempty"`
	// NextReviewAt is the scheduled due date/time for the next review.
	// Stored in UTC. Cards with NextReviewAt <= NOW() are considered due.
	NextReviewAt time.Time `gorm:"not null;index:idx_srs_due" json:"next_review_at"`

	// Associations
	User      *User      `gorm:"foreignKey:UserID"      json:"user,omitempty"`
	VocabItem *VocabItem `gorm:"foreignKey:VocabItemID" json:"vocab_item,omitempty"`
}
