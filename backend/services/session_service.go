package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/learnthor/backend/models"
)

// SessionMasteryThreshold is the number of consecutive correct answers within
// a single session required to consider a VocabItem "mastered" for that session.
// At threshold, next_review_at is pushed to tomorrow (long-term memory signal).
// Below it, a wrong answer resets in-session streak and schedules a 1-minute
// penalty review (short-term re-queue per PRD §3.2).
const SessionMasteryThreshold = 3

// ─────────────────────────────────────────────────────────────────────────────
// Request / Response types
// ─────────────────────────────────────────────────────────────────────────────

// SubmitAnswerRequest is the validated payload from the frontend.
type SubmitAnswerRequest struct {
	// StudySessionID identifies which active session this answer belongs to.
	StudySessionID uuid.UUID `json:"study_session_id" binding:"required"`
	// UserID must match the session owner — validated in the service layer.
	UserID      uuid.UUID `json:"-"`
	VocabItemID uuid.UUID `json:"vocab_item_id" binding:"required"`
	// IsCorrect is the legacy correctness flag. When Rating is omitted the
	// scheduler derives RatingAgain / RatingGood from this boolean.
	IsCorrect bool `json:"is_correct"`
	// Rating is the optional FSRS 1–4 answer (Again/Hard/Good/Easy). When
	// supplied it takes precedence over IsCorrect so the long-term schedule
	// reflects the learner's actual recall quality, not just pass/fail.
	Rating *models.Rating `json:"rating,omitempty"`
}

// SubmitAnswerResponse is returned to the frontend after processing an answer.
type SubmitAnswerResponse struct {
	// SessionItem reflects the updated in-session state for this item.
	SessionItem models.SessionItem `json:"session_item"`
	// SRSReview reflects the updated long-term scheduling state.
	SRSReview models.SRSReview `json:"srs_review"`
	// SessionStatus is "active" or "complete".
	SessionStatus models.SessionStatus `json:"session_status"`
	// NextReviewAt is a convenience copy of SRSReview.NextReviewAt, formatted
	// as RFC3339 UTC — ready for the frontend countdown timer.
	NextReviewAt time.Time `json:"next_review_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Service
// ─────────────────────────────────────────────────────────────────────────────

// SessionService handles the study-session submission business logic.
type SessionService struct {
	db *gorm.DB
}

// NewSessionService constructs a SessionService with the application DB.
func NewSessionService(db *gorm.DB) *SessionService {
	return &SessionService{db: db}
}

// StartSessionRequest starts a new (or resumes the existing active) study
// session for a module.
type StartSessionRequest struct {
	UserID   uuid.UUID `json:"-"`
	ModuleID uuid.UUID `json:"module_id" binding:"required"`
}

// StartSessionResponse returns the session ID plus the batch of vocab items
// the learner should study next. Per PRD §3.2 a batch is 3–5 brand-new
// items; existing in-session items are continued rather than re-introduced.
type StartSessionResponse struct {
	SessionID uuid.UUID             `json:"session_id"`
	Status    models.SessionStatus  `json:"status"`
	BatchSize int                   `json:"batch_size"`
	Items     []StartSessionItemDTO `json:"items"`
}

// StartSessionItemDTO is a flat view of a VocabItem for the session intro.
type StartSessionItemDTO struct {
	VocabItemID       uuid.UUID `json:"vocab_item_id"`
	Kanji             *string   `json:"kanji"`
	Kana              string    `json:"kana"`
	Romaji            string    `json:"romaji"`
	EnglishMeaning    string    `json:"english_meaning"`
	IndonesianMeaning *string   `json:"indonesian_meaning,omitempty"`
	PartOfSpeech      string    `json:"part_of_speech,omitempty"`
	ExampleSentence   string    `json:"example_sentence,omitempty"`
}

// StartSession creates a new study session for the module (or resumes an
// existing active one) and returns the next batch of brand-new items per
// PRD §3.2. Items that already have an SRSReview row in Learning/Review
// state are excluded — those live in the separate daily review queue.
func (s *SessionService) StartSession(req StartSessionRequest) (*StartSessionResponse, error) {
	access, accessErr := NewGatingService(s.db).CheckModuleAccess(req.UserID, req.ModuleID)
	if accessErr != nil {
		return nil, accessErr
	}
	if !access.Allowed {
		return nil, ErrForbidden
	}
	now := time.Now().UTC()

	// Verify module exists.
	var module models.Module
	if err := s.db.Select("id", "title").Where("is_hidden = ?", false).First(&module, "id = ?", req.ModuleID).Error; err != nil {
		return nil, fmt.Errorf("module %s not found: %w", req.ModuleID, ErrNotFound)
	}

	// Resume an existing active session if one exists.
	var session models.StudySession
	err := s.db.
		Where("user_id = ? AND module_id = ? AND status = ?",
			req.UserID, req.ModuleID, models.SessionStatusActive).
		First(&session).Error
	if err == gorm.ErrRecordNotFound {
		// Create a new session.
		session = models.StudySession{
			UserID:    req.UserID,
			ModuleID:  req.ModuleID,
			Status:    models.SessionStatusActive,
			BatchSize: defaultBatchSize,
			StartedAt: now,
		}
		if err := s.db.Create(&session).Error; err != nil {
			return nil, fmt.Errorf("could not create study session: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("could not query study session: %w", err)
	}

	// Pick the next batch of brand-new items for this user/module.
	// "New" = the user has no SRSReview row for this VocabItem yet.
	items, err := s.pickNewBatch(req.UserID, req.ModuleID, session.ID, defaultBatchSize)
	if err != nil {
		return nil, err
	}

	return &StartSessionResponse{
		SessionID: session.ID,
		Status:    session.Status,
		BatchSize: len(items),
		Items:     items,
	}, nil
}

// ResumeSession returns the in-progress items for an active session so the
// frontend can rebuild the study flow after a page reload.
func (s *SessionService) ResumeSession(userID, sessionID uuid.UUID) (*StartSessionResponse, error) {
	var session models.StudySession
	if err := s.db.
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("could not load study session: %w", err)
	}

	access, accessErr := NewGatingService(s.db).CheckModuleAccess(userID, session.ModuleID)
	if accessErr != nil {
		return nil, accessErr
	}
	if !access.Allowed {
		return nil, ErrForbidden
	}
	var items []StartSessionItemDTO
	err := s.db.
		Table("session_items AS si").
		Select("si.vocab_item_id AS vocab_item_id, v.kanji AS kanji, v.kana AS kana, "+
			"v.romaji AS romaji, v.english_meaning AS english_meaning, "+
			"v.indonesian_meaning AS indonesian_meaning, v.part_of_speech AS part_of_speech, "+
			"v.example_sentence AS example_sentence").
		Joins("JOIN vocab_items AS v ON v.id = si.vocab_item_id").
		Where("si.study_session_id = ?", sessionID).
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("could not load session items: %w", err)
	}

	return &StartSessionResponse{
		SessionID: session.ID,
		Status:    session.Status,
		BatchSize: len(items),
		Items:     items,
	}, nil
}

// defaultBatchSize is the number of new items introduced per session
// (PRD §3.2: 3–5 at a time). 3 is the lower bound — easy on beginners,
// keeps sessions short.
const defaultBatchSize = 3

// pickNewBatch selects up to limit VocabItems from the module that the user
// has never reviewed (no SRSReview row). Items are ordered by created_at
// so the curriculum author's intended sequence is preserved.
func (s *SessionService) pickNewBatch(userID, moduleID uuid.UUID, sessionID uuid.UUID, limit int) ([]StartSessionItemDTO, error) {
	var items []StartSessionItemDTO
	err := s.db.
		Table("vocab_items AS v").
		Select("v.id AS vocab_item_id, v.kanji AS kanji, v.kana AS kana, "+
			"v.romaji AS romaji, v.english_meaning AS english_meaning, "+
			"v.indonesian_meaning AS indonesian_meaning, v.part_of_speech AS part_of_speech, "+
			"v.example_sentence AS example_sentence").
		Where("v.module_id = ? AND v.id NOT IN (?)",
			moduleID,
			s.db.Model(&models.SRSReview{}).Select("vocab_item_id").Where("user_id = ?", userID),
		).
		Order("v.created_at ASC").
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("could not load new batch: %w", err)
	}

	// Pre-create the SessionItem rows so SubmitAnswer can FirstOrCreate them
	// without a race on first submission.
	for _, it := range items {
		si := models.SessionItem{
			StudySessionID: sessionID,
			VocabItemID:    it.VocabItemID,
		}
		s.db.Where("study_session_id = ? AND vocab_item_id = ?",
			sessionID, it.VocabItemID).
			FirstOrCreate(&si)
	}

	return items, nil
}

// SubmitAnswer processes one answer submission inside a study session.
//
// It runs the following steps inside a single serialisable transaction:
//  1. Verify the StudySession exists, belongs to req.UserID, and is still active.
//  2. Load (or create) the SessionItem for this vocab card.
//  3. Load (or create) the SRSReview for this user/vocab pair.
//  4. Apply in-session streak logic (SessionItem).
//  5. Apply long-term SRS scheduling logic (SRSReview).
//  6. Persist both records atomically.
//  7. Check if all SessionItems are mastered → complete the session if so.
func (s *SessionService) SubmitAnswer(req SubmitAnswerRequest) (*SubmitAnswerResponse, error) {
	var (
		session     models.StudySession
		sessionItem models.SessionItem
		srsReview   models.SRSReview
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {

		// ── 1. Load & validate the session ──────────────────────────────────

		if err := tx.
			Where("id = ? AND user_id = ?", req.StudySessionID, req.UserID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("study session not found or does not belong to user: %w", ErrNotFound)
			}
			return err
		}

		if session.Status != models.SessionStatusActive {
			return fmt.Errorf("study session is already %s: %w", session.Status, ErrBadRequest)
		}
		access, accessErr := NewGatingService(tx).CheckModuleAccess(req.UserID, session.ModuleID)
		if accessErr != nil {
			return accessErr
		}
		if !access.Allowed {
			return ErrForbidden
		}

		// ── 2. Load or initialise the SessionItem ────────────────────────────
		// Use FirstOrCreate so the first answer on a freshly-added card
		// creates the tracking row without a separate "start session" call.

		sessionItem = models.SessionItem{
			StudySessionID: req.StudySessionID,
			VocabItemID:    req.VocabItemID,
		}
		if err := tx.
			Where("study_session_id = ? AND vocab_item_id = ?",
				req.StudySessionID, req.VocabItemID).
			Attrs(models.SessionItem{
				StudySessionID: req.StudySessionID,
				VocabItemID:    req.VocabItemID,
			}).
			FirstOrCreate(&sessionItem).Error; err != nil {
			return fmt.Errorf("could not load session item: %w", err)
		}

		// ── 3. Load or initialise the SRSReview ──────────────────────────────
		// Cards may not have an SRSReview row yet if the user has never reviewed
		// them outside a session (e.g. brand-new enrollment).

		srsReview = models.SRSReview{
			UserID:       req.UserID,
			VocabItemID:  req.VocabItemID,
			State:        models.ReviewStateNew,
			Difficulty:   5.0, // FSRS default starting difficulty
			NextReviewAt: time.Now().UTC(),
		}
		if err := tx.
			Where("user_id = ? AND vocab_item_id = ?", req.UserID, req.VocabItemID).
			Attrs(srsReview).
			FirstOrCreate(&srsReview).Error; err != nil {
			return fmt.Errorf("could not load SRS review: %w", err)
		}

		// ── 5. Apply long-term SRS scheduling logic (FSRS) ───────────────────

		now := time.Now().UTC()

		// Resolve the rating: explicit FSRS rating wins, else derive from bool.
		rating := RatingFromBool(req.IsCorrect)
		if req.Rating != nil {
			rating = *req.Rating
		}

		// Update IsCorrect to match the resolved rating so the in-session
		// streak reflects the same judgement (Again/Hard count as wrong for
		// streak purposes; Good/Easy count as correct).
		effectiveCorrect := rating == models.RatingGood || rating == models.RatingEasy

		// ── 4. Apply in-session streak logic ─────────────────────────────────

		sessionItem.TotalAttempts++

		if effectiveCorrect {
			sessionItem.CorrectAttempts++
			sessionItem.SessionStreak++
			if sessionItem.SessionStreak >= SessionMasteryThreshold {
				sessionItem.IsMastered = true
			}
		} else {
			sessionItem.SessionStreak = 0
		}

		// FSRS mutates srsReview in place and sets NextReviewAt.
		ScheduleReview(&srsReview, rating, now)

		// If the item is session-mastered, graduate the card out of the
		// short relearning step into a proper long-term interval so the next
		// review isn't 10 minutes after the session ends.
		if sessionItem.IsMastered && srsReview.State != models.ReviewStateReview {
			srsReview.State = models.ReviewStateReview
			srsReview.NextReviewAt = now.Add(24 * time.Hour)
		}

		// ── 6. Persist both records ───────────────────────────────────────────
		// Use Select to update only the columns that changed, avoiding races
		// on unrelated fields (e.g. Stability/Difficulty set by the FSRS worker).

		if err := tx.Model(&sessionItem).
			Select("session_streak", "total_attempts", "correct_attempts", "is_mastered").
			Save(&sessionItem).Error; err != nil {
			return fmt.Errorf("could not save session item: %w", err)
		}

		if err := tx.Model(&srsReview).
			Select("reps", "lapses", "streak", "state", "stability",
				"difficulty", "retrievability",
				"last_reviewed_at", "next_review_at").
			Save(&srsReview).Error; err != nil {
			return fmt.Errorf("could not save SRS review: %w", err)
		}

		// ── 7. Check session completion ───────────────────────────────────────
		// The session is complete when every SessionItem.IsMastered = true.
		// We count unmastered items with a single COUNT query to avoid loading
		// the full item list into memory.

		var unmasteredCount int64
		if err := tx.Model(&models.SessionItem{}).
			Where("study_session_id = ? AND is_mastered = false", req.StudySessionID).
			Count(&unmasteredCount).Error; err != nil {
			return fmt.Errorf("could not count unmastered items: %w", err)
		}

		if unmasteredCount == 0 {
			// All items mastered — calculate score and seal the session.
			score, err := calculateSessionScore(tx, req.StudySessionID)
			if err != nil {
				return err
			}

			completedAt := now
			if err := tx.Model(&session).
				Clauses(clause.Returning{}).
				Updates(map[string]any{
					"status":       models.SessionStatusComplete,
					"score":        score,
					"completed_at": completedAt,
				}).Error; err != nil {
				return fmt.Errorf("could not complete session: %w", err)
			}
			session.Status = models.SessionStatusComplete
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &SubmitAnswerResponse{
		SessionItem:   sessionItem,
		SRSReview:     srsReview,
		SessionStatus: session.Status,
		NextReviewAt:  srsReview.NextReviewAt,
	}, nil
}

// calculateSessionScore computes the final score as:
//
//	(sum of correct_attempts / sum of total_attempts) * 100
//
// rounded down to an integer percentage.
func calculateSessionScore(tx *gorm.DB, sessionID uuid.UUID) (int, error) {
	type aggregate struct {
		TotalCorrect  int64
		TotalAttempts int64
	}
	var agg aggregate

	if err := tx.Model(&models.SessionItem{}).
		Select("SUM(correct_attempts) AS total_correct, SUM(total_attempts) AS total_attempts").
		Where("study_session_id = ?", sessionID).
		Scan(&agg).Error; err != nil {
		return 0, fmt.Errorf("could not aggregate session score: %w", err)
	}

	if agg.TotalAttempts == 0 {
		return 0, nil
	}

	score := int((float64(agg.TotalCorrect) / float64(agg.TotalAttempts)) * 100)
	return score, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Sentinel errors (handlers map these to HTTP status codes)
// ─────────────────────────────────────────────────────────────────────────────

var (
	// ErrNotFound is wrapped when a required record cannot be found.
	ErrNotFound = errors.New("not found")
	// ErrBadRequest is wrapped for invalid state transitions (e.g. session already complete).
	ErrBadRequest = errors.New("bad request")
	// ErrForbidden is returned when the user lacks permission for an action.
	ErrForbidden = errors.New("forbidden")
)
