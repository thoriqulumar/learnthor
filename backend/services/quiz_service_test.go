package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/learnthor/backend/models"
)

// ── FinishQuiz aggregate scoring ─────────────────────────────────────────────

// allCorrectAnswers builds one correct answer per (item, skillType).
func allCorrectAnswers(items []models.VocabItem, skillTypes ...string) []QuizAnswerItem {
	var answers []QuizAnswerItem
	for _, item := range items {
		for _, st := range skillTypes {
			answers = append(answers, QuizAnswerItem{
				VocabItemID: item.ID,
				SkillType:   st,
				IsCorrect:   true,
			})
		}
	}
	return answers
}

func TestFinishQuiz_NoExampleSentences_DoesNotRequireFillBlank(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	// Strip example sentences → fill_blank is impossible for this module,
	// so only kanji_to_kana + kana_to_kanji count as required skills.
	db.Model(&models.VocabItem{}).
		Where("module_id = ?", fx.Module.ID).
		Update("example_sentence", "")

	user := createTestUser(t, db, models.UserRoleStudent)
	svc := NewQuizService(db)

	resp, err := svc.FinishQuiz(FinishQuizRequest{
		UserID:   user.ID,
		ModuleID: fx.Module.ID,
		Answers:  allCorrectAnswers(fx.Items, "kanji_to_kana", "kana_to_kanji"),
	})
	if err != nil {
		t.Fatalf("FinishQuiz: %v", err)
	}

	// Aggregate = avg of the two required skills = 100. Under the old
	// character-check logic this module (non-character) would require all
	// three skills and cap the aggregate at 66, so a 100 here proves the
	// compatibility-based required-skill computation works.
	if resp.HighScore != 100 {
		t.Fatalf("aggregate = %d, want 100", resp.HighScore)
	}
	if !resp.Passed {
		t.Fatal("should pass at 100 with threshold 90")
	}

	var prog models.ModuleProgress
	db.Where("user_id = ? AND module_id = ?", user.ID, fx.Module.ID).First(&prog)
	if !prog.IsUnlocked {
		t.Fatal("module should be unlocked after passing")
	}
}

func TestFinishQuiz_UnplayedRequiredSkillCountsAsZero(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5) // example sentences present → 3 required skills
	user := createTestUser(t, db, models.UserRoleStudent)
	svc := NewQuizService(db)

	resp, err := svc.FinishQuiz(FinishQuizRequest{
		UserID:   user.ID,
		ModuleID: fx.Module.ID,
		Answers:  allCorrectAnswers(fx.Items, "kanji_to_kana"),
	})
	if err != nil {
		t.Fatalf("FinishQuiz: %v", err)
	}

	// 100 on one of three required skills → aggregate 33.
	if resp.HighScore != 33 {
		t.Fatalf("aggregate = %d, want 33 (100+0+0)/3", resp.HighScore)
	}
	if resp.Passed {
		t.Fatal("should not pass below threshold")
	}
}

func TestFinishQuiz_HighScoreKeepsBestAttempt(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	db.Model(&models.VocabItem{}).
		Where("module_id = ?", fx.Module.ID).
		Update("example_sentence", "")

	user := createTestUser(t, db, models.UserRoleStudent)
	svc := NewQuizService(db)

	// Attempt 1: both required skills fully correct → 100.
	if _, err := svc.FinishQuiz(FinishQuizRequest{
		UserID:   user.ID,
		ModuleID: fx.Module.ID,
		Answers:  allCorrectAnswers(fx.Items, "kanji_to_kana", "kana_to_kanji"),
	}); err != nil {
		t.Fatalf("attempt 1: %v", err)
	}

	// Attempt 2: all wrong on both skills → this attempt 0, but high stays 100.
	var wrong []QuizAnswerItem
	for _, item := range fx.Items {
		for _, st := range []string{"kanji_to_kana", "kana_to_kanji"} {
			wrong = append(wrong, QuizAnswerItem{VocabItemID: item.ID, SkillType: st, IsCorrect: false})
		}
	}
	resp, err := svc.FinishQuiz(FinishQuizRequest{
		UserID:   user.ID,
		ModuleID: fx.Module.ID,
		Answers:  wrong,
	})
	if err != nil {
		t.Fatalf("attempt 2: %v", err)
	}

	if resp.Score != 0 {
		t.Fatalf("this attempt score = %d, want 0", resp.Score)
	}
	if resp.HighScore != 100 {
		t.Fatalf("high score = %d, want 100 (best attempt preserved)", resp.HighScore)
	}
	if resp.IsNewHigh {
		t.Fatal("worse attempt must not be flagged as new high")
	}

	var prog models.ModuleProgress
	db.Where("user_id = ? AND module_id = ?", user.ID, fx.Module.ID).First(&prog)
	if prog.AttemptCount != 2 {
		t.Fatalf("attempt_count = %d, want 2", prog.AttemptCount)
	}
}

func TestFinishQuiz_ModuleNotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, models.UserRoleStudent)
	svc := NewQuizService(db)

	_, err := svc.FinishQuiz(FinishQuizRequest{
		UserID:   user.ID,
		ModuleID: uuid.New(),
		Answers:  []QuizAnswerItem{{VocabItemID: uuid.New(), IsCorrect: true}},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestFinishQuiz_EmptyAnswers(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 1)
	user := createTestUser(t, db, models.UserRoleStudent)
	svc := NewQuizService(db)

	_, err := svc.FinishQuiz(FinishQuizRequest{
		UserID:   user.ID,
		ModuleID: fx.Module.ID,
		Answers:  nil,
	})
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

// ── Question generation ──────────────────────────────────────────────────────

func TestGenerateQuiz_QuestionsHaveCorrectAnswerInOptions(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 8)
	user := createTestUser(t, db, models.UserRoleStudent)
	_ = user

	svc := NewQuizService(db)
	questions, err := svc.GenerateQuiz(GenerateQuizRequest{
		ModuleID: fx.Module.ID,
		Type:     "mixed",
		Limit:    8,
	})
	if err != nil {
		t.Fatalf("GenerateQuiz: %v", err)
	}
	if len(questions) == 0 {
		t.Fatal("no questions generated")
	}

	for _, q := range questions {
		found := false
		seen := map[uuid.UUID]bool{}
		for _, opt := range q.Options {
			if seen[opt.ID] {
				t.Fatalf("duplicate option id %s in question %s", opt.ID, q.VocabItemID)
			}
			seen[opt.ID] = true
			if opt.ID == q.CorrectID {
				found = true
			}
		}
		if !found {
			t.Fatalf("question %s has no correct option", q.VocabItemID)
		}
		if q.Prompt == "" {
			t.Fatalf("question %s has empty prompt", q.VocabItemID)
		}
	}
}

func TestGenerateQuiz_SkipsFillBlankWithoutExampleSentence(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	db.Model(&models.VocabItem{}).
		Where("module_id = ?", fx.Module.ID).
		Update("example_sentence", "")

	svc := NewQuizService(db)
	questions, err := svc.GenerateQuiz(GenerateQuizRequest{
		ModuleID: fx.Module.ID,
		Type:     "fill_blank",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("GenerateQuiz: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("got %d fill_blank questions, want 0 (no example sentences)", len(questions))
	}
}

func TestIsQuizCompatible(t *testing.T) {
	kanji := "漢"
	withExample := models.VocabItem{Kanji: &kanji, Kana: "かん", ExampleSentence: "例文です。"}
	noExample := models.VocabItem{Kanji: &kanji, Kana: "かん"}
	kanaOnly := models.VocabItem{Kana: "かん"}

	cases := []struct {
		name string
		item models.VocabItem
		qt   models.QuizType
		want bool
	}{
		{"kanji_to_kana with kanji", withExample, models.QuizTypeKanjiToKana, true},
		{"kana_to_kanji with kanji", withExample, models.QuizTypeKanaToKanji, true},
		{"fill_blank with example", withExample, models.QuizTypeFillBlank, true},
		{"fill_blank without example", noExample, models.QuizTypeFillBlank, false},
		{"kanji_to_kana kana-only", kanaOnly, models.QuizTypeKanjiToKana, false},
		{"kana_to_kanji kana-only", kanaOnly, models.QuizTypeKanaToKanji, false},
	}
	for _, tc := range cases {
		if got := isQuizCompatible(tc.item, tc.qt); got != tc.want {
			t.Fatalf("%s: isQuizCompatible = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestBlankSentence(t *testing.T) {
	// Kanji present → replaced.
	got, ok := blankSentence("これは漢字です。", "漢字", "かんじ")
	if !ok || got != "これは＿＿＿です。" {
		t.Fatalf("blankSentence kanji = %q, %v", got, ok)
	}
	// No kanji → falls back to kana.
	got, ok = blankSentence("これはかんじです。", "", "かんじ")
	if !ok || got != "これは＿＿＿です。" {
		t.Fatalf("blankSentence kana = %q, %v", got, ok)
	}
	// Neither present → not ok.
	if _, ok = blankSentence("なにもないです。", "漢字", "かんじ"); ok {
		t.Fatal("blankSentence should fail when nothing matches")
	}
}

// ── Section quiz gating ──────────────────────────────────────────────────────

func TestGenerateSectionQuiz_ForbiddenWithoutMastery(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	user := createTestUser(t, db, models.UserRoleStudent)

	svc := NewQuizService(db)
	_, err := svc.GenerateSectionQuiz(GenerateSectionQuizRequest{
		SectionID: fx.Section.ID,
		UserID:    user.ID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden (no modules mastered)", err)
	}
}

func TestGenerateSectionQuiz_PrivilegedUserBypassesMastery(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	admin := createTestUser(t, db, models.UserRoleAdmin)

	svc := NewQuizService(db)
	questions, err := svc.GenerateSectionQuiz(GenerateSectionQuizRequest{
		SectionID: fx.Section.ID,
		UserID:    admin.ID,
	})
	if err != nil {
		t.Fatalf("GenerateSectionQuiz for admin: %v", err)
	}
	if len(questions) == 0 {
		t.Fatal("admin should get questions without mastering modules")
	}
}

func TestGenerateQuiz_DistractorsStayInsideModule(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 5)
	other := addModule(t, db, fx, "Other character group", 2)
	foreignKanji := "さ"
	foreign := models.VocabItem{ModuleID: other.ID, Kanji: &foreignKanji, Kana: "sa", Romaji: "sa", EnglishMeaning: "foreign", PartOfSpeech: "character"}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}

	questions, err := NewQuizService(db).GenerateQuiz(GenerateQuizRequest{ModuleID: fx.Module.ID, Type: "kana_to_kanji", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range questions {
		for _, option := range question.Options {
			if option.ID == foreign.ID {
				t.Fatal("quiz used a distractor from another module")
			}
		}
	}
}

func TestGenerateSectionQuiz_IncludesEveryCompatibleQuestionWithoutAnswerKeys(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 3)
	other := addModule(t, db, fx, "Other character group", 2)
	foreignKanji := "外"
	foreign := models.VocabItem{
		ModuleID:        other.ID,
		Kanji:           &foreignKanji,
		Kana:            "そと",
		Romaji:          "soto",
		EnglishMeaning:  "outside",
		PartOfSpeech:    "noun",
		ExampleSentence: "外です。",
	}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	admin := createTestUser(t, db, models.UserRoleAdmin)

	questions, err := NewQuizService(db).GenerateSectionQuiz(GenerateSectionQuizRequest{SectionID: fx.Section.ID, UserID: admin.ID})
	if err != nil {
		t.Fatal(err)
	}
	// All four items support both directions and fill-in-the-blank: 4 x 3.
	if len(questions) != 12 {
		t.Fatalf("questions = %d, want 12", len(questions))
	}
	seen := map[string]bool{}
	crossModuleOptionFound := false
	for _, question := range questions {
		key := question.VocabItemID.String() + ":" + string(question.Type)
		if seen[key] {
			t.Fatalf("duplicate section question %s", key)
		}
		seen[key] = true
		if len(question.Options) < 2 {
			t.Fatalf("question %s has too few options", key)
		}
		if question.VocabItemID != foreign.ID {
			for _, option := range question.Options {
				if option.ID == foreign.ID {
					crossModuleOptionFound = true
				}
			}
		}
	}
	if !crossModuleOptionFound {
		t.Fatal("section test did not use answer options from another module")
	}
}

func TestFinishSectionQuiz_GradesSelectionsOnServerAndCountsUnanswered(t *testing.T) {
	db := newTestDB(t)
	fx := seedCurriculum(t, db, 3)
	admin := createTestUser(t, db, models.UserRoleAdmin)
	answers := []SectionQuizAnswer{
		{VocabItemID: fx.Items[0].ID, SkillType: models.QuizTypeKanjiToKana, SelectedOptionID: fx.Items[0].ID},
		{VocabItemID: fx.Items[0].ID, SkillType: models.QuizTypeKanaToKanji, SelectedOptionID: fx.Items[1].ID},
	}
	result, err := NewQuizService(db).FinishSectionQuiz(FinishSectionQuizRequest{UserID: admin.ID, SectionID: fx.Section.ID, Answers: answers})
	if err != nil {
		t.Fatal(err)
	}
	if result.Correct != 1 || result.Total != 9 || result.Score != 11 {
		t.Fatalf("unexpected result: %+v", result)
	}

	answers = append(answers, answers[0])
	if _, err := NewQuizService(db).FinishSectionQuiz(FinishSectionQuizRequest{UserID: admin.ID, SectionID: fx.Section.ID, Answers: answers}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("duplicate answer error = %v, want ErrBadRequest", err)
	}
}
