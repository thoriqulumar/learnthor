package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/learnthor/backend/models"
)

// ── SubmitAnswer ─────────────────────────────────────────────────────────────

func TestSubmitAnswer_FirstCorrectCreatesRows(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	sess := createSession(t, db, user.ID, fx.Module.ID)

	svc := NewSessionService(db)
	resp, err := svc.SubmitAnswer(SubmitAnswerRequest{
		StudySessionID: sess.ID,
		UserID:         user.ID,
		VocabItemID:    fx.Items[0].ID,
		IsCorrect:      true,
	})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}

	if resp.SessionItem.SessionStreak != 1 {
		t.Fatalf("streak = %d, want 1", resp.SessionItem.SessionStreak)
	}
	if resp.SessionItem.TotalAttempts != 1 || resp.SessionItem.CorrectAttempts != 1 {
		t.Fatalf("attempts = %d/%d, want 1/1",
			resp.SessionItem.CorrectAttempts, resp.SessionItem.TotalAttempts)
	}
	if resp.SessionItem.IsMastered {
		t.Fatal("should not be mastered after 1 correct")
	}
	if resp.SessionStatus != models.SessionStatusActive {
		t.Fatalf("status = %s, want active", resp.SessionStatus)
	}
	if resp.SRSReview.Reps != 1 {
		t.Fatalf("srs reps = %d, want 1", resp.SRSReview.Reps)
	}
}

func TestSubmitAnswer_WrongAnswerResetsStreakAndLapses(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	sess := createSession(t, db, user.ID, fx.Module.ID)

	svc := NewSessionService(db)
	req := SubmitAnswerRequest{
		StudySessionID: sess.ID,
		UserID:         user.ID,
		VocabItemID:    fx.Items[0].ID,
	}

	if _, err := svc.SubmitAnswer(withCorrect(req, true)); err != nil {
		t.Fatalf("first answer: %v", err)
	}
	resp, err := svc.SubmitAnswer(withCorrect(req, false))
	if err != nil {
		t.Fatalf("second answer: %v", err)
	}

	if resp.SessionItem.SessionStreak != 0 {
		t.Fatalf("streak = %d, want 0 after wrong", resp.SessionItem.SessionStreak)
	}
	if resp.SRSReview.Lapses != 1 {
		t.Fatalf("lapses = %d, want 1", resp.SRSReview.Lapses)
	}
	if resp.SRSReview.Streak != 0 {
		t.Fatalf("srs streak = %d, want 0", resp.SRSReview.Streak)
	}
}

func TestSubmitAnswer_SessionCompletesWhenAllMastered(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	sess := createSession(t, db, user.ID, fx.Module.ID)

	svc := NewSessionService(db)
	req := SubmitAnswerRequest{
		StudySessionID: sess.ID,
		UserID:         user.ID,
		VocabItemID:    fx.Items[0].ID,
		IsCorrect:      true,
	}

	var resp *SubmitAnswerResponse
	var err error
	for i := 0; i < SessionMasteryThreshold; i++ {
		resp, err = svc.SubmitAnswer(req)
		if err != nil {
			t.Fatalf("answer %d: %v", i+1, err)
		}
	}

	if !resp.SessionItem.IsMastered {
		t.Fatal("item should be mastered after threshold correct answers")
	}
	if resp.SessionStatus != models.SessionStatusComplete {
		t.Fatalf("status = %s, want complete", resp.SessionStatus)
	}

	var session models.StudySession
	db.First(&session, "id = ?", sess.ID)
	if session.Score == nil || *session.Score != 100 {
		t.Fatalf("score = %v, want 100", session.Score)
	}
	if session.CompletedAt == nil {
		t.Fatal("completed_at should be set")
	}
}

func TestSubmitAnswer_RejectsForeignSession(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	owner := createTestUser(t, db, models.UserRoleStudent)
	intruder := createTestUser(t, db, models.UserRoleStudent)
	sess := createSession(t, db, owner.ID, fx.Module.ID)

	svc := NewSessionService(db)
	_, err := svc.SubmitAnswer(SubmitAnswerRequest{
		StudySessionID: sess.ID,
		UserID:         intruder.ID,
		VocabItemID:    fx.Items[0].ID,
		IsCorrect:      true,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSubmitAnswer_RejectsCompletedSession(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	sess := createSession(t, db, user.ID, fx.Module.ID)

	// Mark the session complete directly.
	db.Model(&sess).Update("status", models.SessionStatusComplete)

	svc := NewSessionService(db)
	_, err := svc.SubmitAnswer(SubmitAnswerRequest{
		StudySessionID: sess.ID,
		UserID:         user.ID,
		VocabItemID:    fx.Items[0].ID,
		IsCorrect:      true,
	})
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

func TestSubmitAnswer_RatingOverridesIsCorrect(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	sess := createSession(t, db, user.ID, fx.Module.ID)

	svc := NewSessionService(db)
	easy := models.RatingEasy
	resp, err := svc.SubmitAnswer(SubmitAnswerRequest{
		StudySessionID: sess.ID,
		UserID:         user.ID,
		VocabItemID:    fx.Items[0].ID,
		IsCorrect:      false, // overridden by Rating=Easy
		Rating:         &easy,
	})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}
	if resp.SessionItem.SessionStreak != 1 {
		t.Fatalf("streak = %d, want 1 (Rating=Easy wins over IsCorrect=false)",
			resp.SessionItem.SessionStreak)
	}
	if resp.SRSReview.Lapses != 0 {
		t.Fatalf("lapses = %d, want 0", resp.SRSReview.Lapses)
	}
}

// ── StartSession ─────────────────────────────────────────────────────────────

func TestStartSession_CreatesNewSessionWithBatch(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewSessionService(db)
	resp, err := svc.StartSession(StartSessionRequest{
		UserID:   user.ID,
		ModuleID: fx.Module.ID,
	})
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	if resp.Status != models.SessionStatusActive {
		t.Fatalf("status = %s, want active", resp.Status)
	}
	if len(resp.Items) != defaultBatchSize {
		t.Fatalf("batch = %d items, want %d", len(resp.Items), defaultBatchSize)
	}
}

func TestStartSession_ResumesExistingActiveSession(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewSessionService(db)
	first, err := svc.StartSession(StartSessionRequest{UserID: user.ID, ModuleID: fx.Module.ID})
	if err != nil {
		t.Fatalf("first StartSession: %v", err)
	}
	second, err := svc.StartSession(StartSessionRequest{UserID: user.ID, ModuleID: fx.Module.ID})
	if err != nil {
		t.Fatalf("second StartSession: %v", err)
	}

	if first.SessionID != second.SessionID {
		t.Fatalf("session ids differ: %s vs %s — should resume, not create",
			first.SessionID, second.SessionID)
	}
}

func TestStartSession_ExcludesAlreadyReviewedItems(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 3)
	user := createTestUser(t, db, models.UserRoleStudent)

	// Mark the first two items as already reviewed.
	for _, item := range fx.Items[:2] {
		db.Create(&models.SRSReview{
			UserID:      user.ID,
			VocabItemID: item.ID,
			State:       models.ReviewStateReview,
		})
	}

	svc := NewSessionService(db)
	resp, err := svc.StartSession(StartSessionRequest{UserID: user.ID, ModuleID: fx.Module.ID})
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("batch = %d items, want 1 (only the unreviewed item)", len(resp.Items))
	}
	if resp.Items[0].VocabItemID != fx.Items[2].ID {
		t.Fatalf("returned wrong item: %s", resp.Items[0].VocabItemID)
	}
}

func TestStartSession_ModuleNotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewSessionService(db)
	_, err := svc.StartSession(StartSessionRequest{
		UserID:   user.ID,
		ModuleID: uuid.New(),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// ── ResumeSession ────────────────────────────────────────────────────────────

func TestResumeSession_ReturnsSessionItems(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewSessionService(db)
	started, err := svc.StartSession(StartSessionRequest{UserID: user.ID, ModuleID: fx.Module.ID})
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	resumed, err := svc.ResumeSession(user.ID, started.SessionID)
	if err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if resumed.SessionID != started.SessionID {
		t.Fatalf("session id mismatch: %s vs %s", resumed.SessionID, started.SessionID)
	}
	if len(resumed.Items) != len(started.Items) {
		t.Fatalf("resumed %d items, want %d", len(resumed.Items), len(started.Items))
	}
}

func TestResumeSession_NotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewSessionService(db)
	_, err := svc.ResumeSession(user.ID, uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// withCorrect returns a copy of req with IsCorrect set.
func withCorrect(req SubmitAnswerRequest, correct bool) SubmitAnswerRequest {
	req.IsCorrect = correct
	return req
}
