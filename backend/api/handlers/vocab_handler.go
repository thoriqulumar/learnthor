package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// VocabHandler serves vocabulary data for a module.
type VocabHandler struct {
	svc *services.VocabService
}

// NewVocabHandler constructs a VocabHandler.
func NewVocabHandler(svc *services.VocabService) *VocabHandler {
	return &VocabHandler{svc: svc}
}

// GetVocabByModule godoc
//
//	@Summary     Get vocabulary items for a module
//	@Description Returns all VocabItems for the requested module.
//	             Access is subject to level-gating: the user must have passed
//	             the predecessor module at >= its PassThreshold score.
//	@Tags        vocab
//	@Produce     json
//	@Param       module_id path     string true "Module UUID"
//	@Success     200       {array}  models.VocabItem
//	@Failure     400       {object} errorResponse
//	@Failure     401       {object} errorResponse
//	@Failure     403       {object} errorResponse
//	@Failure     404       {object} errorResponse
//	@Failure     500       {object} errorResponse
//	@Router      /modules/{module_id}/vocab [get]
func (h *VocabHandler) GetVocabByModule(c *gin.Context) {
	// The gating middleware has already validated and stored the module UUID.
	moduleID, _ := c.Get("module_id")

	items, err := h.svc.GetVocabByModule(moduleID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"module_id": moduleID,
		"count":     len(items),
		"items":     items,
	})
}
