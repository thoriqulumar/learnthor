package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// ─────────────────────────────────────────────────────────────────────────────
// ReviewService — daily spaced-repetition review queue
//
// Unlike SessionService (which drives the per-module learning flow), the
// ReviewService operates on the cross-module daily queue: every VocabItem
// whose SRSReview.NextReviewAt <= now, regardless of which module it
// belongs to. This is the feature that turns the app from a graded
// flashcard tool into a real spaced-repetition system.
// ─────────────────────────────────────────────────────────────────────────────

// ReviewService handles the daily SRS review queue.
type ReviewService struct {
	db *gorm.DB
}

// NewReviewService constructs a ReviewService.
func NewReviewService(db *gorm.DB) *ReviewService {
	return &ReviewService{db: db}
}

// ReviewCardDTO is a single card in the due queue, flattened for the
// frontend: the VocabItem fields the review UI needs plus the scheduling
// state for analytics/debugging.
type ReviewCardDTO struct {
	VocabItemID       uuid.UUID `json:"vocab_item_id"`
	ModuleID          uuid.UUID `json:"module_id"`
	Kanji             *string   `json:"kanji"`
	Kana              string    `json:"kana"`
	Romaji            string    `json:"romaji"`
	EnglishMeaning    string    `json:"english_meaning"`
	IndonesianMeaning *string   `json:"indonesian_meaning,omitempty"`
	PartOfSpeech      string    `json:"part_of_speech,omitempty"`
	ExampleSentence   string    `json:"example_sentence,omitempty"`

	// Scheduling metadata (read-only for the frontend).
	State        models.ReviewState `json:"state"`
	Reps         int                `json:"reps"`
	Lapses       int                `json:"lapses"`
	Streak       int                `json:"streak"`
	NextReviewAt time.Time          `json:"next_review_at"`
}

// GetDueCards returns all cards due for review for the given user, across
// every module. No cap — the queue is unlimited; the user can quit anytime.
// Cards are ordered by NextReviewAt ascending so the most overdue cards
// surface first.
func (s *ReviewService) GetDueCards(userID uuid.UUID) ([]ReviewCardDTO, error) {
	now := time.Now().UTC()

	var rows []ReviewCardDTO
	err := s.db.
		Table("srs_reviews AS r").
		Select("r.vocab_item_id AS vocab_item_id, v.module_id AS module_id, "+
			"v.kanji AS kanji, v.kana AS kana, v.romaji AS romaji, "+
			"v.english_meaning AS english_meaning, v.indonesian_meaning AS indonesian_meaning, "+
			"v.part_of_speech AS part_of_speech, v.example_sentence AS example_sentence, "+
			"r.state AS state, r.reps AS reps, r.lapses AS lapses, "+
			"r.streak AS streak, r.next_review_at AS next_review_at").
		Joins("JOIN vocab_items AS v ON v.id = r.vocab_item_id").
		Where("r.user_id = ? AND r.next_review_at <= ?", userID, now).
		Order("r.next_review_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("could not load due cards: %w", err)
	}
	return rows, nil
}

// SubmitReviewRequest grades a single due card with a 1–4 rating outside of
// any module study session. The card's SRSReview is updated via the FSRS
// scheduler and persisted.
type SubmitReviewRequest struct {
	UserID      uuid.UUID     `json:"-"`
	VocabItemID uuid.UUID     `json:"vocab_item_id" binding:"required"`
	Rating      models.Rating `json:"rating" binding:"required,min=1,max=4"`
}

// SubmitReviewResponse echoes the updated scheduling state.
type SubmitReviewResponse struct {
	VocabItemID  uuid.UUID          `json:"vocab_item_id"`
	State        models.ReviewState `json:"state"`
	Stability    float64            `json:"stability"`
	Difficulty   float64            `json:"difficulty"`
	Streak       int                `json:"streak"`
	NextReviewAt time.Time          `json:"next_review_at"`
}

// SubmitReview grades one due card and persists the updated FSRS state.
// Returns ErrNotFound if the user has no SRSReview row for this card
// (i.e. they are reviewing a card they have never studied).
func (s *ReviewService) SubmitReview(req SubmitReviewRequest) (*SubmitReviewResponse, error) {
	if req.Rating < models.RatingAgain || req.Rating > models.RatingEasy {
		return nil, ErrBadRequest
	}

	now := time.Now().UTC()

	var review models.SRSReview
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Attrs keeps the defaults creation-only: passing them as inline conds
		// to FirstOrCreate would merge state/difficulty/next_review_at into the
		// search WHERE clause, so an existing (already-graduated) review would
		// never match and the insert would violate the unique index.
		if err := tx.
			Where("user_id = ? AND vocab_item_id = ?", req.UserID, req.VocabItemID).
			Attrs(models.SRSReview{
				UserID:       req.UserID,
				VocabItemID:  req.VocabItemID,
				State:        models.ReviewStateNew,
				Difficulty:   5.0,
				NextReviewAt: now,
			}).
			FirstOrCreate(&review).Error; err != nil {
			return fmt.Errorf("could not load SRS review: %w", err)
		}

		ScheduleReview(&review, req.Rating, now)

		if err := tx.Model(&review).
			Select("reps", "lapses", "streak", "state", "stability",
				"difficulty", "retrievability",
				"last_reviewed_at", "next_review_at").
			Save(&review).Error; err != nil {
			return fmt.Errorf("could not save SRS review: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &SubmitReviewResponse{
		VocabItemID:  review.VocabItemID,
		State:        review.State,
		Stability:    review.Stability,
		Difficulty:   review.Difficulty,
		Streak:       review.Streak,
		NextReviewAt: review.NextReviewAt,
	}, nil
}

// ReviewStats summarises the user's review queue for the dashboard widget.
type ReviewStats struct {
	Due        int `json:"due"`        // cards with NextReviewAt <= now
	New        int `json:"new"`        // cards never reviewed (state = new)
	Learning   int `json:"learning"`   // in early learning phase
	Review     int `json:"review"`     // mature cards
	Relearning int `json:"relearning"` // lapsed cards
}

// GetStats returns counts for each SRS state bucket.
func (s *ReviewService) GetStats(userID uuid.UUID) (*ReviewStats, error) {
	now := time.Now().UTC()

	var stats ReviewStats
	type countRow struct {
		State string
		N     int
	}
	var rows []countRow
	err := s.db.Model(&models.SRSReview{}).
		Select("state, count(*) AS n").
		Where("user_id = ?", userID).
		Group("state").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("could not load review stats: %w", err)
	}
	for _, r := range rows {
		switch models.ReviewState(r.State) {
		case models.ReviewStateNew:
			stats.New += r.N
		case models.ReviewStateLearning:
			stats.Learning += r.N
		case models.ReviewStateReview:
			stats.Review += r.N
		case models.ReviewStateRelearning:
			stats.Relearning += r.N
		}
	}

	// Due = any of the above whose NextReviewAt <= now.
	var dueCount int64
	s.db.Model(&models.SRSReview{}).
		Where("user_id = ? AND next_review_at <= ?", userID, now).
		Count(&dueCount)
	stats.Due = int(dueCount)

	return &stats, nil
}
