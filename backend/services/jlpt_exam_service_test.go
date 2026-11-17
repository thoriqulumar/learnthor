package services

import (
	"github.com/google/uuid"
	"github.com/learnthor/backend/models"
	"testing"
)

func TestJLPTExamPaperIsBoundToOwnerAndGradesMissingAnswers(t *testing.T) {
	t.Setenv("JWT_SECRET", "exam-test-secret")
	db := newTestDB(t)
	user := createTestUser(t, db, models.UserRoleStudent)
	other := createTestUser(t, db, models.UserRoleStudent)
	fixture := seedCurriculum(t, db, 5)
	db.Model(&models.Section{}).Where("id = ?", fixture.Section.ID).Update("title", "JLPT N5 Vocabulary")
	svc := NewJLPTMaterialService(db)
	paper, err := svc.GenerateExam(user.ID, "n5")
	if err != nil {
		t.Fatal(err)
	}
	if len(paper.Questions) == 0 {
		t.Fatal("empty paper")
	}
	if _, err := svc.FinishExam(other.ID, paper.Token, nil); err != ErrBadRequest {
		t.Fatalf("accepted another owner: %v", err)
	}
	if _, err := svc.FinishExam(user.ID, paper.Token, []JLPTQuizAnswer{{QuestionID: uuid.New(), SelectedIndex: 0}}); err != ErrBadRequest {
		t.Fatalf("accepted foreign question: %v", err)
	}
	result, err := svc.FinishExam(user.ID, paper.Token, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != len(paper.Questions) || result.Score != 0 {
		t.Fatalf("missing answers lost: %+v", result)
	}
}
