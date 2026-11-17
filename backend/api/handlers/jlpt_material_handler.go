package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// JLPTMaterialHandler serves JLPT lesson material and quizzes.
type JLPTMaterialHandler struct {
	service *services.JLPTMaterialService
}

// NewJLPTMaterialHandler constructs a JLPTMaterialHandler.
func NewJLPTMaterialHandler(service *services.JLPTMaterialService) *JLPTMaterialHandler {
	return &JLPTMaterialHandler{service: service}
}

// List returns material cards for a JLPT level.
func (h *JLPTMaterialHandler) List(c *gin.Context) {
	items, err := h.service.List(c.MustGet("user_id").(uuid.UUID), c.Query("level"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(items), "lessons": items})
}

// Get returns one bilingual lesson and its ungraded questions.
func (h *JLPTMaterialHandler) Get(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "BAD_REQUEST", Message: "invalid lesson id"})
		return
	}
	lesson, err := h.service.Get(c.MustGet("user_id").(uuid.UUID), lessonID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, lesson)
}

type finishJLPTQuizRequest struct {
	Answers []services.JLPTQuizAnswer `json:"answers" binding:"required"`
}

// Exam returns a JLPT-style practice paper with reading last.
func (h *JLPTMaterialHandler) Exam(c *gin.Context) {
	paper, err := h.service.GenerateExam(c.MustGet("user_id").(uuid.UUID), c.Query("level"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, paper)
}

// FinishExam grades the signed paper submitted by its owner.
func (h *JLPTMaterialHandler) FinishExam(c *gin.Context) {
	var request struct {
		Token   string                    `json:"token" binding:"required"`
		Answers []services.JLPTQuizAnswer `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		h.writeError(c, services.ErrBadRequest)
		return
	}
	result, err := h.service.FinishExam(c.MustGet("user_id").(uuid.UUID), request.Token, request.Answers)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// Finish grades one material quiz and stores its best score.
func (h *JLPTMaterialHandler) Finish(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "BAD_REQUEST", Message: "invalid lesson id"})
		return
	}
	var request finishJLPTQuizRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "BAD_REQUEST", Message: "invalid quiz submission"})
		return
	}
	result, err := h.service.Finish(c.MustGet("user_id").(uuid.UUID), lessonID, request.Answers)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *JLPTMaterialHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrBadRequest):
		c.JSON(http.StatusBadRequest, errorResponse{Code: "BAD_REQUEST", Message: "invalid request"})
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: "lesson not found"})
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse{Code: "PREMIUM_REQUIRED", Message: "premium access required"})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
	}
}
