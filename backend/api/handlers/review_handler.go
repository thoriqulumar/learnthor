package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// ReviewHandler wires the daily SRS review queue to HTTP.
type ReviewHandler struct {
	svc *services.ReviewService
}

// NewReviewHandler constructs a ReviewHandler.
func NewReviewHandler(svc *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// GetDueCards godoc
//
//	@Summary     Get cards due for SRS review
//	@Description Returns every VocabItem whose NextReviewAt has passed, across
//	             all modules. No cap — the user can quit anytime. Cards are
//	             ordered most-overdue first.
//	@Tags        review
//	@Produce     json
//	@Success     200 {object} map[string]interface{}
//	@Failure     500 {object} errorResponse
//	@Router      /reviews/due [get]
func (h *ReviewHandler) GetDueCards(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	cards, err := h.svc.GetDueCards(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(cards),
		"cards": cards,
	})
}

// SubmitReview godoc
//
//	@Summary     Submit a single SRS review
//	@Description Grades one due card with a 1–4 rating (Again/Hard/Good/Easy)
//	             and persists the updated FSRS scheduling state.
//	@Tags        review
//	@Accept      json
//	@Produce     json
//	@Param       body body     services.SubmitReviewRequest true "Review payload"
//	@Success     200  {object} services.SubmitReviewResponse
//	@Failure     400  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /reviews/submit [post]
func (h *ReviewHandler) SubmitReview(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req services.SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_PAYLOAD",
			Message: err.Error(),
		})
		return
	}
	req.UserID = userID

	resp, err := h.svc.SubmitReview(req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrBadRequest):
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "INVALID_RATING",
				Message: "rating must be 1 (Again), 2 (Hard), 3 (Good) or 4 (Easy)",
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

// GetStats godoc
//
//	@Summary     Get review queue stats
//	@Description Returns counts of cards in each SRS state (new / learning /
//	             review / relearning) plus the total due count, for the
//	             homepage dashboard widget.
//	@Tags        review
//	@Produce     json
//	@Success     200 {object} services.ReviewStats
//	@Failure     500 {object} errorResponse
//	@Router      /reviews/stats [get]
func (h *ReviewHandler) GetStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	stats, err := h.svc.GetStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
