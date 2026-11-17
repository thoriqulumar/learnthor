package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// seedReview inserts an SRSReview row for (user, item) with the given state
// and due time.
func seedReview(t *testing.T, db *gorm.DB, userID, itemID uuid.UUID, state models.ReviewState, dueAt time.Time) {
	t.Helper()
	r := models.SRSReview{
		UserID:       userID,
		VocabItemID:  itemID,
		State:        state,
		NextReviewAt: dueAt,
	}
	if err := db.Create(&r).Error; err != nil {
		t.Fatalf("seed review: %v", err)
	}
}

func TestGetDueCards_ReturnsOnlyDueOrderedMostOverdueFirst(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 3)
	user := createTestUser(t, db, models.UserRoleStudent)
	now := time.Now().UTC()

	seedReview(t, db, user.ID, fx.Items[0].ID, models.ReviewStateReview, now.Add(-2*time.Hour))   // due, older
	seedReview(t, db, user.ID, fx.Items[1].ID, models.ReviewStateLearning, now.Add(-1*time.Hour)) // due, newer
	seedReview(t, db, user.ID, fx.Items[2].ID, models.ReviewStateReview, now.Add(1*time.Hour))    // future — excluded

	svc := NewReviewService(db)
	cards, err := svc.GetDueCards(user.ID)
	if err != nil {
		t.Fatalf("GetDueCards: %v", err)
	}

	if len(cards) != 2 {
		t.Fatalf("got %d cards, want 2 (future card must be excluded)", len(cards))
	}
	if cards[0].VocabItemID != fx.Items[0].ID {
		t.Fatalf("first card = %s, want most-overdue item %s",
			cards[0].VocabItemID, fx.Items[0].ID)
	}
	if cards[0].Kana != fx.Items[0].Kana || cards[0].EnglishMeaning != fx.Items[0].EnglishMeaning {
		t.Fatalf("card missing vocab join data: %+v", cards[0])
	}
}

func TestGetDueCards_IsolatesUsers(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	me := createTestUser(t, db, models.UserRoleStudent)
	other := createTestUser(t, db, models.UserRoleStudent)
	now := time.Now().UTC()

	seedReview(t, db, other.ID, fx.Items[0].ID, models.ReviewStateReview, now.Add(-time.Hour))

	svc := NewReviewService(db)
	cards, err := svc.GetDueCards(me.ID)
	if err != nil {
		t.Fatalf("GetDueCards: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("got %d cards, want 0 — must not leak other users' cards", len(cards))
	}
}

func TestSubmitReview_RejectsInvalidRating(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewReviewService(db)
	for _, bad := range []models.Rating{0, 5, 99} {
		_, err := svc.SubmitReview(SubmitReviewRequest{
			UserID:      user.ID,
			VocabItemID: fx.Items[0].ID,
			Rating:      bad,
		})
		if !errors.Is(err, ErrBadRequest) {
			t.Fatalf("rating %d: err = %v, want ErrBadRequest", bad, err)
		}
	}
}

func TestSubmitReview_CreatesAndSchedulesNewCard(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	before := time.Now().UTC()

	svc := NewReviewService(db)
	resp, err := svc.SubmitReview(SubmitReviewRequest{
		UserID:      user.ID,
		VocabItemID: fx.Items[0].ID,
		Rating:      models.RatingGood,
	})
	if err != nil {
		t.Fatalf("SubmitReview: %v", err)
	}

	if resp.State != models.ReviewStateLearning {
		t.Fatalf("state = %s, want learning after first Good", resp.State)
	}
	if !resp.NextReviewAt.After(before) {
		t.Fatalf("next_review_at = %v, want after now (%v)", resp.NextReviewAt, before)
	}

	// Row must be persisted.
	var count int64
	db.Model(&models.SRSReview{}).
		Where("user_id = ? AND vocab_item_id = ?", user.ID, fx.Items[0].ID).
		Count(&count)
	if count != 1 {
		t.Fatalf("persisted rows = %d, want 1", count)
	}
}

func TestSubmitReview_UpdatesExistingCard(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	past := time.Now().UTC().Add(-48 * time.Hour)

	seedReview(t, db, user.ID, fx.Items[0].ID, models.ReviewStateReview, past)
	db.Model(&models.SRSReview{}).
		Where("user_id = ? AND vocab_item_id = ?", user.ID, fx.Items[0].ID).
		Updates(map[string]any{"stability": 10.0, "difficulty": 5.0, "reps": 5, "last_reviewed_at": past})

	svc := NewReviewService(db)
	resp, err := svc.SubmitReview(SubmitReviewRequest{
		UserID:      user.ID,
		VocabItemID: fx.Items[0].ID,
		Rating:      models.RatingAgain,
	})
	if err != nil {
		t.Fatalf("SubmitReview: %v", err)
	}
	if resp.State != models.ReviewStateRelearning {
		t.Fatalf("state = %s, want relearning after Again", resp.State)
	}
	if resp.Streak != 0 {
		t.Fatalf("streak = %d, want 0 after Again", resp.Streak)
	}
}

func TestGetStats_CountsStatesAndDue(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 4)
	user := createTestUser(t, db, models.UserRoleStudent)
	now := time.Now().UTC()

	seedReview(t, db, user.ID, fx.Items[0].ID, models.ReviewStateNew, now.Add(-time.Hour))
	seedReview(t, db, user.ID, fx.Items[1].ID, models.ReviewStateLearning, now.Add(-time.Hour))
	seedReview(t, db, user.ID, fx.Items[2].ID, models.ReviewStateReview, now.Add(-time.Hour))
	seedReview(t, db, user.ID, fx.Items[3].ID, models.ReviewStateReview, now.Add(24*time.Hour)) // future

	svc := NewReviewService(db)
	stats, err := svc.GetStats(user.ID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if stats.New != 1 || stats.Learning != 1 || stats.Review != 2 {
		t.Fatalf("stats = %+v, want new=1 learning=1 review=2", stats)
	}
	if stats.Due != 3 {
		t.Fatalf("due = %d, want 3 (future card excluded)", stats.Due)
	}
}
