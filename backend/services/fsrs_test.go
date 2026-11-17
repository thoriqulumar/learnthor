package services

import (
	"math"
	"testing"
	"time"

	"github.com/learnthor/backend/models"
)

// newReview is a tiny constructor for a fresh card used by the tests.
func newReview() *models.SRSReview {
	now := time.Now().UTC()
	return &models.SRSReview{
		State:        models.ReviewStateNew,
		Difficulty:   5.0,
		Stability:    0,
		NextReviewAt: now,
	}
}

func TestScheduleReview_FirstGoodReviewGraduatesToLearning(t *testing.T) {
	r := newReview()
	now := time.Now().UTC()
	d := ScheduleReview(r, models.RatingGood, now)

	if r.Reps != 1 {
		t.Fatalf("Reps = %d, want 1", r.Reps)
	}
	if r.State != models.ReviewStateLearning {
		t.Fatalf("State = %s, want learning", r.State)
	}
	if r.Stability <= 0 {
		t.Fatalf("Stability = %v, want > 0", r.Stability)
	}
	if d <= 0 {
		t.Fatalf("interval = %v, want > 0", d)
	}
	if r.LastReviewedAt == nil || !r.LastReviewedAt.Equal(now) {
		t.Fatalf("LastReviewedAt not set to now")
	}
	if r.Streak != 1 {
		t.Fatalf("Streak = %d, want 1", r.Streak)
	}
}

func TestScheduleReview_AgainCausesLapse(t *testing.T) {
	r := newReview()
	r.Reps = 5
	r.State = models.ReviewStateReview
	r.Stability = 10.0
	r.Lapses = 1
	r.Streak = 3
	// Set a realistic last-reviewed time in the past so retrievability is < 1.
	past := time.Now().UTC().Add(-72 * time.Hour)
	r.LastReviewedAt = &past
	now := time.Now().UTC()

	ScheduleReview(r, models.RatingAgain, now)

	if r.Lapses != 2 {
		t.Fatalf("Lapses = %d, want 2", r.Lapses)
	}
	if r.State != models.ReviewStateRelearning {
		t.Fatalf("State = %s, want relearning", r.State)
	}
	if r.Streak != 0 {
		t.Fatalf("Streak = %d, want 0", r.Streak)
	}
	// Stability should shrink after a lapse.
	if r.Stability >= 10.0 {
		t.Fatalf("Stability = %v, want < previous 10.0", r.Stability)
	}
	// Relearning uses a short fixed step (10 minutes).
	if got, want := r.NextReviewAt.Sub(now), 10*time.Minute; got != want {
		t.Fatalf("interval = %v, want %v for relearning", got, want)
	}
}

func TestScheduleReview_MatureCardStabilityGrows(t *testing.T) {
	r := newReview()
	r.Reps = 3
	r.State = models.ReviewStateReview
	r.Stability = 7.0
	r.Difficulty = 6.0
	// Reviewed 3 days ago so retrievability < 1 (forces non-zero growth).
	past := time.Now().UTC().Add(-72 * time.Hour)
	r.LastReviewedAt = &past
	now := time.Now().UTC()
	prevS := r.Stability

	ScheduleReview(r, models.RatingGood, now)

	if r.Stability <= prevS {
		t.Fatalf("Stability = %v, want > previous %v on correct review", r.Stability, prevS)
	}
	// Next interval should be at least 1 day for a mature card.
	if r.NextReviewAt.Sub(now) < 24*time.Hour {
		t.Fatalf("interval = %v, want >= 24h", r.NextReviewAt.Sub(now))
	}
}

func TestScheduleReview_EasyGrowsMoreThanGood(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-72 * time.Hour)

	rGood := newReview()
	rGood.Reps = 3
	rGood.State = models.ReviewStateReview
	rGood.Stability = 7.0
	rGood.Difficulty = 6.0
	rGood.LastReviewedAt = &past

	rEasy := newReview()
	rEasy.Reps = 3
	rEasy.State = models.ReviewStateReview
	rEasy.Stability = 7.0
	rEasy.Difficulty = 6.0
	rEasy.LastReviewedAt = &past

	ScheduleReview(rGood, models.RatingGood, now)
	ScheduleReview(rEasy, models.RatingEasy, now)

	if rEasy.Stability <= rGood.Stability {
		t.Fatalf("Easy Stability = %v should exceed Good Stability = %v",
			rEasy.Stability, rGood.Stability)
	}
}

func TestScheduleReview_DifficultyClampedToRange(t *testing.T) {
	// Repeated Again ratings should drive D toward 1 but never below.
	r := newReview()
	now := time.Now().UTC()
	for i := 0; i < 20; i++ {
		ScheduleReview(r, models.RatingAgain, now.Add(time.Duration(i)*time.Hour))
	}
	if r.Difficulty < 1.0 || r.Difficulty > 10.0 {
		t.Fatalf("Difficulty = %v, want in [1, 10]", r.Difficulty)
	}

	// Repeated Easy ratings should drive D toward 10 but never above.
	r2 := newReview()
	for i := 0; i < 20; i++ {
		ScheduleReview(r2, models.RatingEasy, now.Add(time.Duration(i)*time.Hour))
	}
	if r2.Difficulty < 1.0 || r2.Difficulty > 10.0 {
		t.Fatalf("Difficulty = %v, want in [1, 10]", r2.Difficulty)
	}
}

func TestRetrievability(t *testing.T) {
	// R(0, S) = 1 for any positive S — perfect recall at t=0.
	if got := retrievability(5.0, 0); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("R(0,5) = %v, want 1.0", got)
	}
	// R(t, 0) = 1 — new card, no forgetting.
	if got := retrievability(0, 10); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("R(10,0) = %v, want 1.0", got)
	}
	// R should decrease as elapsed time grows.
	r1 := retrievability(5.0, 1)
	r10 := retrievability(5.0, 10)
	if r10 >= r1 {
		t.Fatalf("R(10) = %v should be < R(1) = %v", r10, r1)
	}
}

func TestNextInterval(t *testing.T) {
	// Higher stability => longer interval.
	i1 := nextInterval(1.0, 0.9)
	i10 := nextInterval(10.0, 0.9)
	if i10 <= i1 {
		t.Fatalf("interval(10) = %v should exceed interval(1) = %v", i10, i1)
	}
	// Higher retention target => shorter interval (review sooner to hit 99%).
	i90 := nextInterval(5.0, 0.9)
	i99 := nextInterval(5.0, 0.99)
	if i99 >= i90 {
		t.Fatalf("interval(R=0.99) = %v should be < interval(R=0.9) = %v", i99, i90)
	}
}

func TestRatingFromBool(t *testing.T) {
	if got := RatingFromBool(true); got != models.RatingGood {
		t.Fatalf("RatingFromBool(true) = %v, want Good", got)
	}
	if got := RatingFromBool(false); got != models.RatingAgain {
		t.Fatalf("RatingFromBool(false) = %v, want Again", got)
	}
}

func TestScheduleReview_HardResetsStreak(t *testing.T) {
	r := newReview()
	r.Streak = 5
	now := time.Now().UTC()
	ScheduleReview(r, models.RatingHard, now)
	if r.Streak != 0 {
		t.Fatalf("Streak = %d, want 0 after Hard", r.Streak)
	}
}
