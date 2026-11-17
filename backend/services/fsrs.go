package services

import (
	"math"
	"time"

	"github.com/learnthor/backend/models"
)

// ─────────────────────────────────────────────────────────────────────────────
// FSRS-lite scheduler
//
// Implements a self-contained FSRS-4.5-style spaced-repetition scheduler.
// No external dependencies — the 17 default parameters below are the
// FSRS-4.5 published defaults (see open-spaced-repetition/fsrs4anki).
//
// The scheduler is pure: it mutates the supplied *models.SRSReview in place
// and returns the next interval. Callers persist the review afterwards.
// ─────────────────────────────────────────────────────────────────────────────

// fsrsParams are the FSRS-4.5 default weights.
//
//	w[0..3]   initial stability for ratings 1..4 (Again/Hard/Good/Easy)
//	w[4..7]   difficulty parameters (init D, rating penalty, damping, mean-reversion)
//	w[8..16]  stability update parameters (correct review, lapse, etc.)
var fsrsParams = [17]float64{
	0.40, // w0  — S init for Again
	0.60, // w1  — S init for Hard
	2.40, // w2  — S init for Good
	5.80, // w3  — S init for Easy
	4.93, // w4  — initial Difficulty (D0 = w4 - (rating-3)*w5)
	0.94, // w5  — D rating penalty
	0.86, // w6  — D damping factor (multiplier on rating delta)
	0.01, // w7  — D mean-reversion weight toward w4
	1.49, // w8  — correct review: exp multiplier
	0.14, // w9  — correct review: S exponent (decay)
	0.94, // w10 — correct review: retrievability factor
	2.18, // w11 — lapse: base multiplier
	0.05, // w12 — lapse: D exponent
	0.34, // w13 — lapse: (S+1) exponent
	1.26, // w14 — lapse: retrievability factor
	0.29, // w15 — hard review: extra penalty
	2.61, // w16 — easy review: extra bonus
}

// targetRetention is the desired recall probability at the next review.
// 0.9 = 90 % recall — the standard FSRS default.
const targetRetention = 0.9

// maxInterval caps the scheduled interval to avoid absurd values.
const maxInterval = 365 * 10 // 10 years

// ratingFactor scales the stability update by rating for correct reviews.
// Index by rating (1..4). Again (1) and Hard (2) get dampeners;
// Good (3) is the baseline; Easy (4) gets a bonus.
var ratingFactor = [5]float64{
	0,                    // unused (rating starts at 1)
	1.0,                  // Again — handled via lapse path
	1.0 + fsrsParams[15], // Hard — small penalty
	1.0,                  // Good — baseline
	1.0 + fsrsParams[16], // Easy — bonus
}

// ScheduleReview applies the FSRS-4.5 update to r in place and returns the
// next interval (the duration from now until NextReviewAt).
//
// The caller is responsible for having already set r.LastReviewedAt and
// r.Reps on first creation; this function updates every scheduling field.
// Pass the same `now` that the caller uses for persistence so timestamps
// stay consistent.
func ScheduleReview(r *models.SRSReview, rating models.Rating, now time.Time) time.Duration {
	if r == nil {
		return 24 * time.Hour
	}

	// ── Compute elapsed days since last review ────────────────────────────
	// For brand-new cards (Reps == 0) elapsed is 0; the first review uses
	// initial-stability formulas rather than the update formulas.
	var elapsedDays float64
	if r.Reps > 0 && r.LastReviewedAt != nil {
		elapsedDays = now.Sub(*r.LastReviewedAt).Hours() / 24.0
		if elapsedDays < 0 {
			elapsedDays = 0
		}
	}

	// Retrievability at review time (power forgetting curve).
	R := retrievability(r.Stability, elapsedDays)

	// ── Difficulty update ─────────────────────────────────────────────────
	// D' = D - w6*(rating-3), then mean-revert toward w4:
	// D'' = w7*D' + (1-w7)*w4. Clamp to [1, 10].
	if r.Reps == 0 {
		r.Difficulty = clampD(fsrsParams[4] - float64(rating-3)*fsrsParams[5])
	} else {
		newD := r.Difficulty - fsrsParams[6]*float64(rating-3)
		newD = fsrsParams[7]*newD + (1-fsrsParams[7])*fsrsParams[4]
		r.Difficulty = clampD(newD)
	}

	// ── Stability update + state transition ───────────────────────────────
	switch {
	case rating == models.RatingAgain:
		// Lapse: card is forgotten.
		r.Lapses++
		r.Stability = lapseStability(r.Stability, r.Difficulty, R)
		r.State = models.ReviewStateRelearning
	case r.Reps == 0 || r.State == models.ReviewStateNew:
		// First-ever review: initial stability from rating.
		r.Stability = initialStability(rating)
		r.State = models.ReviewStateLearning
	case r.State == models.ReviewStateLearning || r.State == models.ReviewStateRelearning:
		// Graduating step: use initial stability scaled by rating factor
		// so Hard doesn't graduate as fast as Easy.
		r.Stability = initialStability(rating) * ratingFactor[rating]
		if rating >= models.RatingGood {
			r.State = models.ReviewStateReview
		}
	default:
		// Mature review: full FSRS stability update.
		r.Stability = reviewStability(r.Stability, r.Difficulty, R, rating)
		if rating == models.RatingHard {
			// Hard keeps the card in review but with dampened growth (already
			// applied via ratingFactor inside reviewStability).
		}
	}

	// ── Streak (UI metric, not core FSRS) ─────────────────────────────────
	if rating == models.RatingAgain || rating == models.RatingHard {
		r.Streak = 0
	} else {
		r.Streak++
	}

	// ── Reps + timestamps ─────────────────────────────────────────────────
	r.Reps++
	r.Retrievability = R
	r.LastReviewedAt = &now

	// ── Next interval ─────────────────────────────────────────────────────
	// Relearning and fresh-learning states use a short fixed step (Anki-style
	// learning steps) rather than the stability-derived interval, so the card
	// is re-queued within the session/hour for reinforcement. Mature reviews
	// use the full FSRS interval from stability + target retention.
	var intervalDays float64
	switch r.State {
	case models.ReviewStateRelearning:
		intervalDays = 10.0 / (24.0 * 60.0) // 10 minutes
	case models.ReviewStateLearning:
		intervalDays = 10.0 / 24.0 // 10 minutes (graduating step)
	default:
		intervalDays = nextInterval(r.Stability, targetRetention)
		if intervalDays < 1.0/24.0 {
			intervalDays = 1.0 / (24.0 * 60.0) // minimum 1 minute
		}
		if intervalDays > float64(maxInterval) {
			intervalDays = float64(maxInterval)
		}
	}
	r.NextReviewAt = now.Add(time.Duration(intervalDays * 24 * float64(time.Hour)))

	return r.NextReviewAt.Sub(now)
}

// retrievability is the FSRS power forgetting curve:
//
//	R(t, S) = (1 + t / (9 * S))^(-1)
//
// with S in days and t the elapsed days since last review.
// Returns 1.0 when S == 0 (new card, no forgetting yet).
func retrievability(stability, elapsedDays float64) float64 {
	if stability <= 0 {
		return 1.0
	}
	return 1.0 / (1.0 + elapsedDays/(9.0*stability))
}

// initialStability returns the starting S for a given rating on first review.
// w[rating-1] holds the four initial-stability values (Again/Hard/Good/Easy).
// Again (1) is too small for a 1-day minimum, so we floor at 0.4.
func initialStability(rating models.Rating) float64 {
	i := int(rating) - 1
	if i < 0 || i >= 4 {
		return fsrsParams[2] // default to Good
	}
	s := fsrsParams[i]
	if s < 0.4 {
		s = 0.4
	}
	return s
}

// reviewStability computes the new stability for a mature card reviewed
// correctly (rating >= Hard). Implements the FSRS-4.5 "next stability after
// correct review" formula:
//
//	S' = S * (1 + exp(w8) * (11 - D) * S^(-w9) * (exp((1-R)*w10) - 1) * factor)
//
// where factor is ratingFactor[rating] (Hard penalty / Easy bonus).
func reviewStability(stability, difficulty, retrievabilityR float64, rating models.Rating) float64 {
	factor := ratingFactor[rating]
	inner := math.Exp((1-retrievabilityR)*fsrsParams[10]) - 1
	inc := math.Exp(fsrsParams[8]) * (11.0 - difficulty) *
		math.Pow(stability, -fsrsParams[9]) * inner * factor
	newS := stability * (1 + inc)
	if newS < stability {
		// Stability should never decrease on a correct review; clamp.
		newS = stability
	}
	return newS
}

// lapseStability computes the new (smaller) stability when a card is
// forgotten (rated Again). Implements FSRS-4.5 lapse formula:
//
//	S' = w11 * D^(-w12) * ((S+1)^w13 - 1) * exp((1-R)*w14)
func lapseStability(stability, difficulty, retrievabilityR float64) float64 {
	newS := fsrsParams[11] *
		math.Pow(difficulty, -fsrsParams[12]) *
		(math.Pow(stability+1, fsrsParams[13]) - 1) *
		math.Exp((1-retrievabilityR)*fsrsParams[14])
	if newS < 0.1 {
		newS = 0.1
	}
	return newS
}

// nextInterval converts a stability value into an interval (days) using the
// target retention. Solves R = (1 + t/(9S))^(-1) for t:
//
//	t = 9 * S * (1/R - 1)
func nextInterval(stability, retention float64) float64 {
	if retention <= 0 || retention >= 1 {
		retention = targetRetention
	}
	if stability <= 0 {
		return 1.0 // tomorrow
	}
	return 9.0 * stability * (1.0/retention - 1.0)
}

// clampD keeps Difficulty within [1.0, 10.0].
func clampD(d float64) float64 {
	if d < 1.0 {
		return 1.0
	}
	if d > 10.0 {
		return 10.0
	}
	return d
}

// RatingFromBool maps a simple correct/incorrect boolean to a FSRS Rating.
// Used by callers that only track correctness (e.g. multiple-choice quizzes)
// so they remain compatible with the rating-based scheduler.
func RatingFromBool(correct bool) models.Rating {
	if correct {
		return models.RatingGood
	}
	return models.RatingAgain
}
