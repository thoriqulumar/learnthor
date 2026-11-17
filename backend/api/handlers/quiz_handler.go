package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// QuizHandler serves generated multiple-choice quiz questions for a module.
type QuizHandler struct {
	svc *services.QuizService
}

// NewQuizHandler constructs a QuizHandler.
func NewQuizHandler(svc *services.QuizService) *QuizHandler {
	return &QuizHandler{svc: svc}
}

var validQuizTypes = map[string]bool{
	"kanji_to_kana": true,
	"kana_to_kanji": true,
	"fill_blank":    true,
	"mixed":         true,
}

// GetQuiz godoc
//
//	@Summary     Generate quiz questions for a module
//	@Description Returns multiple-choice questions derived from the module's vocabulary.
//	             Three question types are available:
//	               kanji_to_kana — see kanji/katakana, pick the correct kana reading;
//	               kana_to_kanji — see kana reading, pick the correct kanji from similar-looking options;
//	               fill_blank    — example sentence with one word blanked (＿＿＿), pick the correct kanji.
//	             "mixed" (default) randomly selects a compatible type for each item.
//	@Tags        quiz
//	@Produce     json
//	@Param       module_id path     string true  "Module UUID"
//	@Param       type      query    string false "Question type (kanji_to_kana|kana_to_kanji|fill_blank|mixed)" default(mixed)
//	@Param       limit     query    int    false "Max number of questions (1–50)"                               default(10)
//	@Success     200       {object} map[string]interface{}
//	@Failure     400       {object} errorResponse
//	@Failure     403       {object} errorResponse
//	@Failure     404       {object} errorResponse
//	@Failure     500       {object} errorResponse
//	@Router      /modules/{module_id}/quiz [get]
func (h *QuizHandler) GetQuiz(c *gin.Context) {
	moduleID, _ := c.Get("module_id")

	quizType := c.DefaultQuery("type", "mixed")
	if !validQuizTypes[quizType] {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_QUIZ_TYPE",
			Message: "type must be one of: kanji_to_kana, kana_to_kanji, fill_blank, mixed",
		})
		return
	}

	limit := 10
	if ls := c.Query("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 {
			limit = v
		}
	}

	questions, err := h.svc.GenerateQuiz(services.GenerateQuizRequest{
		ModuleID: moduleID.(uuid.UUID),
		Type:     quizType,
		Limit:    limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"module_id": moduleID,
		"count":     len(questions),
		"questions": questions,
	})
}

// FinishQuiz godoc
//
//	@Summary     Submit quiz results and save progress
//	@Description Calculates score from submitted answers and persists it in
//	             module_progress, updating the user's high score if improved.
//	@Tags        quiz
//	@Accept      json
//	@Produce     json
//	@Param       module_id path     string                         true "Module UUID"
//	@Param       body      body     services.FinishQuizRequest     true "Answer list"
//	@Success     200       {object} services.FinishQuizResponse
//	@Failure     400       {object} errorResponse
//	@Failure     403       {object} errorResponse
//	@Failure     404       {object} errorResponse
//	@Router      /modules/{module_id}/quiz/finish [post]
func (h *QuizHandler) FinishQuiz(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	moduleID := c.MustGet("module_id").(uuid.UUID)

	var body struct {
		Answers []services.QuizAnswerItem `json:"answers" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_PAYLOAD",
			Message: err.Error(),
		})
		return
	}

	resp, err := h.svc.FinishQuiz(services.FinishQuizRequest{
		UserID:   userID,
		ModuleID: moduleID,
		Answers:  body.Answers,
	})
	if err != nil {
		switch err {
		case services.ErrNotFound:
			c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: "module not found"})
		case services.ErrBadRequest:
			c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_PAYLOAD", Message: "no answers provided"})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetSectionQuiz generates mixed questions from all modules in a section.
// Only available when all modules in the section are mastered by the user.
func (h *QuizHandler) GetSectionQuiz(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	sectionID, err := uuid.Parse(c.Param("section_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_ID", Message: "invalid section_id"})
		return
	}

	questions, err := h.svc.GenerateSectionQuiz(services.GenerateSectionQuizRequest{
		SectionID: sectionID,
		UserID:    userID,
	})
	if err != nil {
		switch err {
		case services.ErrNotFound:
			c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: "section not found or has no modules"})
		case services.ErrForbidden:
			c.JSON(http.StatusForbidden, errorResponse{Code: "FORBIDDEN", Message: "master all modules in this section first"})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"section_id":    sectionID,
		"count":         len(questions),
		"timer_seconds": 30 * 60,
		"questions":     questions,
	})
}

// FinishSectionQuiz grades submitted answers for a section-level test.
func (h *QuizHandler) FinishSectionQuiz(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	sectionID, err := uuid.Parse(c.Param("section_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_ID", Message: "invalid section_id"})
		return
	}

	var body struct {
		Answers []services.SectionQuizAnswer `json:"answers"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_PAYLOAD", Message: err.Error()})
		return
	}

	resp, err := h.svc.FinishSectionQuiz(services.FinishSectionQuizRequest{
		UserID:    userID,
		SectionID: sectionID,
		Answers:   body.Answers,
	})
	if err != nil {
		switch err {
		case services.ErrBadRequest:
			c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_PAYLOAD", Message: "answers contain an invalid or duplicate question"})
		case services.ErrForbidden:
			c.JSON(http.StatusForbidden, errorResponse{Code: "FORBIDDEN", Message: "master all modules in this section first"})
		case services.ErrNotFound:
			c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: "section test has no questions"})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}
