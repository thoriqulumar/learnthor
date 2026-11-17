package services

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/learnthor/backend/models"
)

func TestJLPTMaterialServiceListsLocksAndGrades(t *testing.T) {
	database := newTestDB(t)
	user := createTestUser(t, database, models.UserRoleStudent)
	content, _ := json.Marshal([]JLPTContentSection{{HeadingEnglish: "One"}, {HeadingEnglish: "Two"}})
	lesson := models.JLPTLesson{
		SeedKey: "lesson:test", Level: "N5", Category: "grammar",
		TitleEnglish: "Test", TitleIndonesian: "Tes", SummaryEnglish: "Summary",
		SummaryIndonesian: "Ringkasan", ContentJSON: string(content), SortOrder: 1,
	}
	if err := database.Create(&lesson).Error; err != nil {
		t.Fatal(err)
	}
	options, _ := json.Marshal([]string{"wrong", "right"})
	for index := 1; index <= 3; index++ {
		question := models.JLPTQuestion{
			SeedKey: "question:" + uuid.NewString(), LessonID: lesson.ID, SortOrder: index,
			PromptEnglish: "Question", PromptIndonesian: "Pertanyaan", OptionsJSON: string(options),
			CorrectIndex: 1, ExplanationEnglish: "Because", ExplanationIndonesian: "Karena",
		}
		if err := database.Create(&question).Error; err != nil {
			t.Fatal(err)
		}
	}
	premiumLesson := lesson
	premiumLesson.ID = uuid.Nil
	premiumLesson.SeedKey = "lesson:premium"
	premiumLesson.Level = "N4"
	premiumLesson.IsPremium = true
	if err := database.Create(&premiumLesson).Error; err != nil {
		t.Fatal(err)
	}

	service := NewJLPTMaterialService(database)
	items, err := service.List(user.ID, "n4")
	if err != nil || len(items) != 1 || !items[0].Locked {
		t.Fatalf("expected locked N4 lesson, items=%+v err=%v", items, err)
	}
	detail, err := service.Get(user.ID, lesson.ID)
	if err != nil || len(detail.Questions) != 3 || len(detail.Questions[0].Options) != 2 {
		t.Fatalf("unexpected lesson detail: %+v err=%v", detail, err)
	}
	answers := make([]JLPTQuizAnswer, 0, len(detail.Questions))
	for _, question := range detail.Questions {
		answers = append(answers, JLPTQuizAnswer{QuestionID: question.ID, SelectedIndex: 1})
	}
	result, err := service.Finish(user.ID, lesson.ID, answers)
	if err != nil || result.Score != 100 || result.BestScore != 100 {
		t.Fatalf("unexpected quiz result: %+v err=%v", result, err)
	}
	if _, err := service.Get(user.ID, premiumLesson.ID); err != ErrForbidden {
		t.Fatalf("expected premium lesson to be forbidden, got %v", err)
	}
}
