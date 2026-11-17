package services

import (
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/learnthor/backend/models"
)

const quizBlank = "＿＿＿"

// QuizService generates multiple-choice questions from VocabItems.
type QuizService struct {
	db *gorm.DB
}

// NewQuizService constructs a QuizService.
func NewQuizService(db *gorm.DB) *QuizService {
	return &QuizService{db: db}
}

// GenerateQuizRequest carries the parameters for quiz generation.
type GenerateQuizRequest struct {
	ModuleID uuid.UUID
	// Type is one of "kanji_to_kana", "kana_to_kanji", "fill_blank", or "mixed".
	// Defaults to "mixed" if empty.
	Type string
	// Limit caps the number of questions returned (1–50). Defaults to 10.
	Limit int
}

// GenerateQuiz returns up to Limit quiz questions for the given module.
//
// Distractor selection strategy:
//   - For kanji_to_kana: distractors are kana readings of other vocab items;
//     same PartOfSpeech items are preferred so the options are plausible.
//   - For kana_to_kanji / fill_blank: distractors are kanji forms; items that
//     share a CJK ideograph with the target are prioritised for visual challenge,
//     then same PartOfSpeech, then anything else in the section.
func (s *QuizService) GenerateQuiz(req GenerateQuizRequest) ([]models.QuizQuestion, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	if req.Type == "" {
		req.Type = "mixed"
	}

	// Resolve the module before loading its questions.
	var module models.Module
	if err := s.db.Where("is_hidden = ?", false).First(&module, "id = ?", req.ModuleID).Error; err != nil {
		return nil, ErrNotFound
	}

	// Items that will be tested (from the requested module only).
	var moduleItems []models.VocabItem
	if err := s.db.Where("module_id = ?", req.ModuleID).Find(&moduleItems).Error; err != nil {
		return nil, err
	}
	if len(moduleItems) == 0 {
		return []models.QuizQuestion{}, nil
	}

	// Keep distractors inside the current module. A focused module such as
	// hiragana vowels must never offer answers from another character group.
	pool := moduleItems

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Shuffle so repeated calls return different orderings.
	r.Shuffle(len(moduleItems), func(i, j int) {
		moduleItems[i], moduleItems[j] = moduleItems[j], moduleItems[i]
	})

	var questions []models.QuizQuestion
	for _, item := range moduleItems {
		if len(questions) >= req.Limit {
			break
		}
		qt := resolveQuizType(item, req.Type, r)
		if qt == "" {
			continue
		}
		q := buildQuizQuestion(item, qt, pool, r)
		if q == nil {
			continue
		}
		questions = append(questions, *q)
	}

	return questions, nil
}

// resolveQuizType picks a QuizType for the item based on the request.
// Returns "" when the item is not compatible with the requested type.
func resolveQuizType(item models.VocabItem, requestedType string, r *rand.Rand) models.QuizType {
	if requestedType != "mixed" {
		qt := models.QuizType(requestedType)
		if isQuizCompatible(item, qt) {
			return qt
		}
		return ""
	}

	// mixed: collect all compatible types and pick one at random.
	var compatible []models.QuizType
	if isQuizCompatible(item, models.QuizTypeKanjiToKana) {
		compatible = append(compatible, models.QuizTypeKanjiToKana)
	}
	if isQuizCompatible(item, models.QuizTypeKanaToKanji) {
		compatible = append(compatible, models.QuizTypeKanaToKanji)
	}
	if isQuizCompatible(item, models.QuizTypeFillBlank) {
		compatible = append(compatible, models.QuizTypeFillBlank)
	}
	if len(compatible) == 0 {
		return ""
	}
	return compatible[r.Intn(len(compatible))]
}

func isQuizCompatible(item models.VocabItem, qt models.QuizType) bool {
	hasKanji := item.Kanji != nil && *item.Kanji != ""
	switch qt {
	case models.QuizTypeKanjiToKana:
		return hasKanji
	case models.QuizTypeKanaToKanji:
		return hasKanji
	case models.QuizTypeFillBlank:
		return hasKanji && item.ExampleSentence != ""
	}
	return false
}

func buildQuizQuestion(target models.VocabItem, qt models.QuizType, pool []models.VocabItem, r *rand.Rand) *models.QuizQuestion {
	switch qt {
	case models.QuizTypeKanjiToKana:
		return buildKanjiToKana(target, pool, r)
	case models.QuizTypeKanaToKanji:
		return buildKanaToKanji(target, pool, r)
	case models.QuizTypeFillBlank:
		return buildFillBlank(target, pool, r)
	}
	return nil
}

// buildKanjiToKana — prompt: kanji/katakana form, options: kana readings.
func buildKanjiToKana(target models.VocabItem, pool []models.VocabItem, r *rand.Rand) *models.QuizQuestion {
	distractors := pickKanaDistractors(target, pool, 3, r)
	options := makeQuizOptions(target, distractors, func(v models.VocabItem) string {
		return v.Kana
	})
	if len(options) < 2 {
		return nil
	}
	r.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	return &models.QuizQuestion{
		VocabItemID: target.ID,
		Type:        models.QuizTypeKanjiToKana,
		Prompt:      *target.Kanji,
		CorrectID:   target.ID,
		Options:     options,
	}
}

// buildKanaToKanji — prompt: kana reading, options: kanji forms.
func buildKanaToKanji(target models.VocabItem, pool []models.VocabItem, r *rand.Rand) *models.QuizQuestion {
	distractors := pickKanjiDistractors(target, pool, 3, r)
	options := makeQuizOptions(target, distractors, kanjiOrKana)
	if len(options) < 2 {
		return nil
	}
	r.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	return &models.QuizQuestion{
		VocabItemID: target.ID,
		Type:        models.QuizTypeKanaToKanji,
		Prompt:      target.Kana,
		CorrectID:   target.ID,
		Options:     options,
	}
}

// buildFillBlank — prompt: example sentence with target word blanked, options: kanji forms.
func buildFillBlank(target models.VocabItem, pool []models.VocabItem, r *rand.Rand) *models.QuizQuestion {
	blanked, ok := blankSentence(target.ExampleSentence, *target.Kanji, target.Kana)
	if !ok {
		return nil
	}
	distractors := pickKanjiDistractors(target, pool, 3, r)
	options := makeQuizOptions(target, distractors, kanjiOrKana)
	if len(options) < 2 {
		return nil
	}
	r.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	return &models.QuizQuestion{
		VocabItemID: target.ID,
		Type:        models.QuizTypeFillBlank,
		Prompt:      blanked,
		CorrectID:   target.ID,
		Options:     options,
	}
}

// makeQuizOptions assembles the option slice (correct answer + distractors).
func makeQuizOptions(target models.VocabItem, distractors []models.VocabItem, text func(models.VocabItem) string) []models.QuizOption {
	opts := []models.QuizOption{{ID: target.ID, Text: text(target)}}
	for _, d := range distractors {
		opts = append(opts, models.QuizOption{ID: d.ID, Text: text(d)})
	}
	return opts
}

// kanjiOrKana returns the kanji form if present, otherwise the kana.
func kanjiOrKana(v models.VocabItem) string {
	if v.Kanji != nil && *v.Kanji != "" {
		return *v.Kanji
	}
	return v.Kana
}

// pickKanaDistractors selects up to n items from pool with a kana reading
// different from the target. Same-POS items are preferred.
func pickKanaDistractors(target models.VocabItem, pool []models.VocabItem, n int, r *rand.Rand) []models.VocabItem {
	var samePOS, other []models.VocabItem
	for _, v := range pool {
		if v.ID == target.ID || v.Kana == target.Kana {
			continue
		}
		if v.PartOfSpeech == target.PartOfSpeech {
			samePOS = append(samePOS, v)
		} else {
			other = append(other, v)
		}
	}
	r.Shuffle(len(samePOS), func(i, j int) { samePOS[i], samePOS[j] = samePOS[j], samePOS[i] })
	r.Shuffle(len(other), func(i, j int) { other[i], other[j] = other[j], other[i] })
	result := append(samePOS, other...)
	if len(result) > n {
		return result[:n]
	}
	return result
}

// pickKanjiDistractors selects up to n items from pool that have a different
// kanji form. Priority order:
//  1. Same POS AND shares a CJK character with target (visually similar, same grammar)
//  2. Shares a CJK character with target (visually challenging)
//  3. Same POS only
//  4. Anything else in the pool
func pickKanjiDistractors(target models.VocabItem, pool []models.VocabItem, n int, r *rand.Rand) []models.VocabItem {
	targetKanji := ""
	if target.Kanji != nil {
		targetKanji = *target.Kanji
	}

	var topTier, visual, samePOS, other []models.VocabItem
	for _, v := range pool {
		if v.ID == target.ID || v.Kanji == nil || *v.Kanji == "" || *v.Kanji == targetKanji {
			continue
		}
		samePos := v.PartOfSpeech == target.PartOfSpeech
		shared := sharesKanjiChar(targetKanji, *v.Kanji)
		switch {
		case samePos && shared:
			topTier = append(topTier, v)
		case shared:
			visual = append(visual, v)
		case samePos:
			samePOS = append(samePOS, v)
		default:
			other = append(other, v)
		}
	}

	for _, bucket := range []*[]models.VocabItem{&topTier, &visual, &samePOS, &other} {
		r.Shuffle(len(*bucket), func(i, j int) { (*bucket)[i], (*bucket)[j] = (*bucket)[j], (*bucket)[i] })
	}

	result := append(append(append(topTier, visual...), samePOS...), other...)
	if len(result) > n {
		return result[:n]
	}
	return result
}

// sharesKanjiChar reports whether strings a and b share at least one CJK ideograph.
func sharesKanjiChar(a, b string) bool {
	for _, ch := range a {
		if isCJKChar(ch) && strings.ContainsRune(b, ch) {
			return true
		}
	}
	return false
}

// isCJKChar returns true for characters in the CJK Unified Ideographs block.
func isCJKChar(r rune) bool {
	return r >= '\u4E00' && r <= '\u9FFF'
}

// blankSentence replaces the first occurrence of kanji (or kana if kanji is
// absent) in sentence with ＿＿＿. Returns ("", false) if nothing was replaced.
func blankSentence(sentence, kanji, kana string) (string, bool) {
	if kanji != "" && strings.Contains(sentence, kanji) {
		return strings.Replace(sentence, kanji, quizBlank, 1), true
	}
	if strings.Contains(sentence, kana) {
		return strings.Replace(sentence, kana, quizBlank, 1), true
	}
	return "", false
}

// ─────────────────────────────────────────────────────────────────────────────
// FinishQuiz — save quiz result to module_progress
// ─────────────────────────────────────────────────────────────────────────────

// QuizAnswerItem is a single question result submitted by the frontend.
type QuizAnswerItem struct {
	VocabItemID uuid.UUID `json:"vocab_item_id" binding:"required"`
	// SkillType is the quiz direction for this answer (kanji_to_kana, kana_to_kanji,
	// fill_blank). Required for per-skill mastery tracking.
	SkillType string `json:"skill_type"`
	IsCorrect bool   `json:"is_correct"`
}

// FinishQuizRequest is the payload for POST /modules/:id/quiz/finish.
type FinishQuizRequest struct {
	UserID   uuid.UUID        `json:"-"` // injected by handler from JWT
	ModuleID uuid.UUID        `json:"-"` // injected by handler from URL param
	Answers  []QuizAnswerItem `json:"answers" binding:"required,min=1"`
}

// FinishQuizResponse is returned after saving the result.
type FinishQuizResponse struct {
	Score         int            `json:"score"`          // aggregate score this attempt (avg of skill scores)
	HighScore     int            `json:"high_score"`     // best aggregate ever for this module
	Passed        bool           `json:"passed"`         // aggregate >= pass_threshold
	IsNewHigh     bool           `json:"is_new_high"`    // this attempt beat previous aggregate high
	PassThreshold int            `json:"pass_threshold"` // module's pass threshold
	Correct       int            `json:"correct"`
	Total         int            `json:"total"`
	SkillScores   map[string]int `json:"skill_scores"` // per-skill best scores (all historical)
}

// FinishQuiz calculates per-skill scores from the submitted answers,
// persists them in module_skill_progress, and derives the aggregate module
// score (average of all attempted skills) which is stored in module_progress.
func (s *QuizService) FinishQuiz(req FinishQuizRequest) (*FinishQuizResponse, error) {
	if len(req.Answers) == 0 {
		return nil, ErrBadRequest
	}

	// Load module to obtain pass threshold.
	var module models.Module
	if err := s.db.Where("is_hidden = ?", false).First(&module, "id = ?", req.ModuleID).Error; err != nil {
		return nil, ErrNotFound
	}

	// ── Group answers by skill type ───────────────────────────────────────────
	type tally struct{ correct, total int }
	skillTally := map[string]*tally{}
	totalCorrect := 0
	for _, a := range req.Answers {
		st := a.SkillType
		if st == "" {
			st = string(models.QuizTypeKanjiToKana) // safe fallback
		}
		if _, ok := skillTally[st]; !ok {
			skillTally[st] = &tally{}
		}
		skillTally[st].total++
		if a.IsCorrect {
			skillTally[st].correct++
			totalCorrect++
		}
	}

	// ── Per-skill scores for this attempt ─────────────────────────────────────
	attemptSkillScores := make(map[string]int, len(skillTally))
	for st, t := range skillTally {
		attemptSkillScores[st] = (t.correct * 100) / t.total
	}

	now := time.Now().UTC()
	var aggregateScore int
	allSkillScores := map[string]int{}
	isNewHigh := false

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Upsert each skill's high score.
		for st, score := range attemptSkillScores {
			var sp models.ModuleSkillProgress
			tx.Where("user_id = ? AND module_id = ? AND skill_type = ?",
				req.UserID, req.ModuleID, st).FirstOrInit(&sp)
			sp.UserID = req.UserID
			sp.ModuleID = req.ModuleID
			sp.SkillType = st
			if score > sp.HighScore {
				sp.HighScore = score
			}
			if err := tx.Save(&sp).Error; err != nil {
				return err
			}
		}

		// 2. Reload ALL skill scores for this module to compute aggregate.
		var allSkillProgs []models.ModuleSkillProgress
		if err := tx.Where("user_id = ? AND module_id = ?",
			req.UserID, req.ModuleID).Find(&allSkillProgs).Error; err != nil {
			return err
		}
		for _, sp := range allSkillProgs {
			allSkillScores[sp.SkillType] = sp.HighScore
		}

		// Determine required skills by checking which quiz types are
		// actually compatible with the module's vocab items. Only skills
		// that can be attempted are required, so modules without example
		// sentences aren't penalised with a permanent fill_blank 0.
		var moduleItems []models.VocabItem
		if err := tx.Where("module_id = ?", req.ModuleID).
			Find(&moduleItems).Error; err != nil {
			return err
		}
		var requiredSkills []string
		seen := map[string]bool{}
		for _, item := range moduleItems {
			for _, qt := range []models.QuizType{
				models.QuizTypeKanjiToKana,
				models.QuizTypeKanaToKanji,
				models.QuizTypeFillBlank,
			} {
				st := string(qt)
				if !seen[st] && isQuizCompatible(item, qt) {
					seen[st] = true
					requiredSkills = append(requiredSkills, st)
				}
			}
		}
		if len(requiredSkills) == 0 {
			requiredSkills = []string{"kanji_to_kana", "kana_to_kanji"}
		}

		sum := 0
		for _, sk := range requiredSkills {
			sum += allSkillScores[sk] // 0 if not yet attempted
		}
		aggregateScore = sum / len(requiredSkills)

		// 3. Upsert module_progress with aggregate score.
		var prog models.ModuleProgress
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND module_id = ?", req.UserID, req.ModuleID).
			FirstOrInit(&prog).Error; err != nil {
			return err
		}
		prog.UserID = req.UserID
		prog.ModuleID = req.ModuleID
		prog.AttemptCount++
		prog.LastAttemptAt = &now
		if aggregateScore > prog.HighScore {
			prog.HighScore = aggregateScore
			isNewHigh = true
		}
		prog.IsUnlocked = prog.HighScore >= module.PassThreshold

		return tx.Save(&prog).Error
	})
	if err != nil {
		return nil, err
	}

	thisAttemptScore := 0
	if len(req.Answers) > 0 {
		thisAttemptScore = (totalCorrect * 100) / len(req.Answers)
	}

	return &FinishQuizResponse{
		Score:         thisAttemptScore,
		HighScore:     aggregateScore,
		Passed:        aggregateScore >= module.PassThreshold,
		IsNewHigh:     isNewHigh,
		PassThreshold: module.PassThreshold,
		Correct:       totalCorrect,
		Total:         len(req.Answers),
		SkillScores:   allSkillScores,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Section Quiz — test covering all modules in a section (mastered users only)
// ─────────────────────────────────────────────────────────────────────────────

// GenerateSectionQuizRequest carries parameters for a section-level quiz.
type GenerateSectionQuizRequest struct {
	SectionID uuid.UUID
	UserID    uuid.UUID
}

// SectionQuizQuestion is safe to send before an exam is submitted: it omits the answer key.
type SectionQuizQuestion struct {
	VocabItemID uuid.UUID           `json:"vocab_item_id"`
	Type        models.QuizType     `json:"type"`
	Prompt      string              `json:"prompt"`
	Options     []models.QuizOption `json:"options"`
}

// GenerateSectionQuiz returns every compatible quiz direction for every item
// in every module in the section, shuffled into one exam.
// Returns ErrNotFound when the section has no modules, and ErrForbidden when
// not all modules have been mastered by the user.
func (s *QuizService) GenerateSectionQuiz(req GenerateSectionQuizRequest) ([]SectionQuizQuestion, error) {
	access, err := NewGatingService(s.db).CheckSectionAccess(req.UserID, req.SectionID)
	if err != nil {
		return nil, err
	}
	if !access.Allowed {
		return nil, ErrForbidden
	}
	// Load all modules in the section.
	var modules []models.Module
	if err := s.db.Where("section_id = ? AND is_hidden = ?", req.SectionID, false).Find(&modules).Error; err != nil {
		return nil, ErrNotFound
	}
	if len(modules) == 0 {
		return nil, ErrNotFound
	}

	moduleIDs := make([]uuid.UUID, len(modules))
	for i, m := range modules {
		moduleIDs[i] = m.ID
	}

	// Verify all modules are mastered (is_unlocked = true) for this user.
	// Privileged users (admin / editor) skip the mastery requirement so they
	// can review any section quiz without first passing every module.
	if !CanBypassProgression(s.db, req.UserID) {
		var unlockedCount int64
		s.db.Model(&models.ModuleProgress{}).
			Where("user_id = ? AND module_id IN ? AND is_unlocked = true", req.UserID, moduleIDs).
			Count(&unlockedCount)
		if int(unlockedCount) < len(modules) {
			return nil, ErrForbidden
		}
	}

	// Load all vocabulary so every item in the section is eligible as a
	// distractor during the comprehensive section test.
	var allItems []models.VocabItem
	if err := s.db.Where("module_id IN ?", moduleIDs).Find(&allItems).Error; err != nil {
		return nil, err
	}
	if len(allItems) == 0 {
		return []SectionQuizQuestion{}, nil
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	questions := buildSectionQuestions(allItems, r)

	return questions, nil
}

func buildSectionQuestions(allItems []models.VocabItem, r *rand.Rand) []SectionQuizQuestion {
	questions := make([]SectionQuizQuestion, 0, len(allItems)*2)
	quizTypes := []models.QuizType{models.QuizTypeKanjiToKana, models.QuizTypeKanaToKanji, models.QuizTypeFillBlank}
	for _, item := range allItems {
		for _, quizType := range quizTypes {
			if !isQuizCompatible(item, quizType) {
				continue
			}
			question := buildQuizQuestion(item, quizType, allItems, r)
			if question == nil {
				continue
			}
			questions = append(questions, SectionQuizQuestion{
				VocabItemID: question.VocabItemID,
				Type:        question.Type,
				Prompt:      question.Prompt,
				Options:     question.Options,
			})
		}
	}
	r.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	return questions
}

// SectionQuizAnswer contains the selected option, never client-computed correctness.
type SectionQuizAnswer struct {
	VocabItemID      uuid.UUID       `json:"vocab_item_id" binding:"required"`
	SkillType        models.QuizType `json:"skill_type" binding:"required"`
	SelectedOptionID uuid.UUID       `json:"selected_option_id" binding:"required"`
}

// FinishSectionQuizRequest is the payload for POST /sections/:id/quiz/finish.
type FinishSectionQuizRequest struct {
	UserID    uuid.UUID           `json:"-"`
	SectionID uuid.UUID           `json:"-"`
	Answers   []SectionQuizAnswer `json:"answers"`
}

// FinishSectionQuizResponse is returned after grading a section quiz.
type FinishSectionQuizResponse struct {
	Score   int `json:"score"`
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

// FinishSectionQuiz grades the submitted answers and returns the score.
// Results are not persisted — the section test is a review exercise only.
func (s *QuizService) FinishSectionQuiz(req FinishSectionQuizRequest) (*FinishSectionQuizResponse, error) {
	access, err := NewGatingService(s.db).CheckSectionAccess(req.UserID, req.SectionID)
	if err != nil {
		return nil, err
	}
	if !access.Allowed {
		return nil, ErrForbidden
	}
	var items []models.VocabItem
	if err := s.db.Where("module_id IN (?)", s.db.Model(&models.Module{}).Select("id").Where("section_id = ? AND is_hidden = ?", req.SectionID, false)).Find(&items).Error; err != nil {
		return nil, err
	}
	itemByID := make(map[uuid.UUID]models.VocabItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	total := len(buildSectionQuestions(items, rand.New(rand.NewSource(1))))
	if total == 0 {
		return nil, ErrNotFound
	}
	correct := 0
	seen := make(map[string]bool, len(req.Answers))
	for _, answer := range req.Answers {
		item, exists := itemByID[answer.VocabItemID]
		key := answer.VocabItemID.String() + ":" + string(answer.SkillType)
		if !exists || !isQuizCompatible(item, answer.SkillType) || seen[key] {
			return nil, ErrBadRequest
		}
		seen[key] = true
		if answer.SelectedOptionID == answer.VocabItemID {
			correct++
		}
	}
	score := (correct * 100) / total

	return &FinishSectionQuizResponse{
		Score:   score,
		Correct: correct,
		Total:   total,
	}, nil
}
