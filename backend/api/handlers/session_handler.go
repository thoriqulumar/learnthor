package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// SessionHandler wires the HTTP layer to the SessionService.
type SessionHandler struct {
	svc *services.SessionService
}

// NewSessionHandler constructs a SessionHandler.
func NewSessionHandler(svc *services.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// SubmitAnswer godoc
//
//	@Summary     Submit a study-session answer
//	@Description Process a single correct/incorrect answer for a vocab item
//	             inside an active study session. Updates both the in-session
//	             streak (SessionItem) and the long-term SRS schedule (SRSReview).
//	@Tags        session
//	@Accept      json
//	@Produce     json
//	@Param       body body     services.SubmitAnswerRequest true "Answer payload"
//	@Success     200  {object} services.SubmitAnswerResponse
//	@Failure     400  {object} errorResponse
//	@Failure     404  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /sessions/submit [post]
func (h *SessionHandler) SubmitAnswer(c *gin.Context) {
	var req services.SubmitAnswerRequest

	// binding:"required" tags on the struct fields handle missing/malformed UUIDs.
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_PAYLOAD",
			Message: err.Error(),
		})
		return
	}

	req.UserID = c.MustGet("user_id").(uuid.UUID)
	resp, err := h.svc.SubmitAnswer(req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrForbidden):
			c.JSON(http.StatusForbidden, errorResponse{Code: "MODULE_LOCKED", Message: "Complete the required lessons first."})
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, errorResponse{
				Code:    "NOT_FOUND",
				Message: err.Error(),
			})
		case errors.Is(err, services.ErrBadRequest):
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "INVALID_STATE",
				Message: err.Error(),
			})
		default:
			// Do not leak internal error details to the client.
			c.JSON(http.StatusInternalServerError, errorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "an unexpected error occurred",
			})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// StartSession godoc
//
//	@Summary     Start or resume a module study session
//	@Description Creates a new StudySession (or resumes an active one) for the
//	             given module and returns the next batch of brand-new items
//	             per PRD §3.2 (3–5 items at a time).
//	@Tags        session
//	@Accept      json
//	@Produce     json
//	@Param       body body     services.StartSessionRequest true "Start payload"
//	@Success     200  {object} services.StartSessionResponse
//	@Failure     404  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /sessions [post]
func (h *SessionHandler) StartSession(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req services.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_PAYLOAD",
			Message: err.Error(),
		})
		return
	}
	req.UserID = userID

	resp, err := h.svc.StartSession(req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrForbidden):
			c.JSON(http.StatusForbidden, errorResponse{Code: "MODULE_LOCKED", Message: "Complete the required lessons first."})
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, errorResponse{
				Code:    "NOT_FOUND",
				Message: "module not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "an unexpected error occurred",
			})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ResumeSession godoc
//
//	@Summary     Resume an active study session
//	@Description Returns the in-progress items for an active session so the
//	             frontend can rebuild the study flow after a page reload.
//	@Tags        session
//	@Produce     json
//	@Param       session_id path string true "StudySession UUID"
//	@Success     200  {object} services.StartSessionResponse
//	@Failure     404  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /sessions/{session_id} [get]
func (h *SessionHandler) ResumeSession(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_ID",
			Message: "invalid session_id",
		})
		return
	}

	resp, err := h.svc.ResumeSession(userID, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrForbidden):
			c.JSON(http.StatusForbidden, errorResponse{Code: "MODULE_LOCKED", Message: "Complete the required lessons first."})
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, errorResponse{
				Code:    "NOT_FOUND",
				Message: "study session not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "an unexpected error occurred",
			})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}
