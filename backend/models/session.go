package models

import (
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// StudySession
// ─────────────────────────────────────────────────────────────────────────────

// SessionStatus mirrors the lifecycle of a study session.
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"    // in-flight
	SessionStatusComplete  SessionStatus = "complete"  // all items reached session_streak 2
	SessionStatusAbandoned SessionStatus = "abandoned" // user quit / timeout
)

// StudySession represents a single study attempt on a Module.
//
// PRD §3.2 rules encoded here:
//   - New VocabItems are introduced 3-5 at a time (BatchSize).
//   - The session is "complete" only when every SessionItem.SessionStreak >= 2.
//   - Status is set to "complete" by the service layer after each answer when
//     all active SessionItems satisfy the streak condition.
//
// Indexes:
//   - (user_id, module_id, status): for fetching the user's active session on
//     a given module without a full table scan.
//   - user_id: for listing a user's session history.
type StudySession struct {
	Base

	UserID   uuid.UUID `gorm:"type:uuid;not null;index:idx_study_session_user_module" json:"user_id"`
	ModuleID uuid.UUID `gorm:"type:uuid;not null;index:idx_study_session_user_module" json:"module_id"`

	Status SessionStatus `gorm:"type:varchar(20);not null;default:'active';index:idx_study_session_user_module" json:"status"`

	// BatchSize is the number of new items introduced at once (3–5 per PRD §3.2).
	BatchSize int `gorm:"not null;default:3" json:"batch_size"`

	// Score is the final percentage score recorded when Status = complete.
	// Null while the session is still active.
	Score *int `json:"score,omitempty"`

	StartedAt   time.Time  `gorm:"not null" json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	User         *User         `gorm:"foreignKey:UserID"   json:"user,omitempty"`
	Module       *Module       `gorm:"foreignKey:ModuleID" json:"module,omitempty"`
	SessionItems []SessionItem `gorm:"foreignKey:StudySessionID" json:"session_items,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// SessionItem
// ─────────────────────────────────────────────────────────────────────────────

// SessionItem tracks the ephemeral, per-item state within a single StudySession.
//
// This is intentionally separate from SRSReview:
//   - SRSReview holds the long-term FSRS scheduling state (persists forever).
//   - SessionItem holds the short-term in-session streak (discarded after completion).
//
// PRD §3.2:
//   - SessionStreak resets to 0 on a wrong answer; the item is re-queued.
//   - The session ends when every item's SessionStreak reaches 2.
//
// Indexes:
//   - (study_session_id, vocab_item_id): composite unique — one row per
//     item per session.
//   - study_session_id: for loading the full active queue for a session.
type SessionItem struct {
	Base

	StudySessionID uuid.UUID `gorm:"type:uuid;not null;index:idx_session_item_session_vocab,unique" json:"study_session_id"`
	VocabItemID    uuid.UUID `gorm:"type:uuid;not null;index:idx_session_item_session_vocab,unique" json:"vocab_item_id"`

	// SessionStreak is the count of consecutive correct answers in THIS session.
	// Resets to 0 on any wrong answer. Session completes when all items hit 2.
	SessionStreak int `gorm:"not null;default:0" json:"session_streak"`

	// TotalAttempts is the number of times this item was presented in the session.
	TotalAttempts int `gorm:"not null;default:0" json:"total_attempts"`
	// CorrectAttempts is the number of times it was answered correctly.
	CorrectAttempts int `gorm:"not null;default:0" json:"correct_attempts"`

	// IsMastered is true once SessionStreak >= 2 for this item.
	IsMastered bool `gorm:"not null;default:false" json:"is_mastered"`

	StudySession *StudySession `gorm:"foreignKey:StudySessionID" json:"study_session,omitempty"`
	VocabItem    *VocabItem    `gorm:"foreignKey:VocabItemID"    json:"vocab_item,omitempty"`
}
