package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/learnthor/backend/services"
)

// KanjiHandler serves the N5-N3 kanji reference catalog.
type KanjiHandler struct {
	svc *services.KanjiService
}

// NewKanjiHandler constructs a KanjiHandler.
func NewKanjiHandler(svc *services.KanjiService) *KanjiHandler {
	return &KanjiHandler{svc: svc}
}

// List returns the kanji catalog with optional level and text filters.
func (h *KanjiHandler) List(c *gin.Context) {
	level := strings.ToUpper(strings.TrimSpace(c.Query("level")))
	if level != "" && level != "N5" && level != "N4" && level != "N3" {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_LEVEL", Message: "level must be N5, N4, or N3"})
		return
	}
	query := strings.TrimSpace(c.Query("q"))
	if utf8.RuneCountInString(query) > 100 {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_QUERY", Message: "search text is too long"})
		return
	}
	entries, err := h.svc.List(level, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(entries), "items": entries})
}

// Word returns readings and meanings from the vocabulary catalog.
func (h *KanjiHandler) Word(c *gin.Context) {
	word := strings.TrimSpace(c.Query("q"))
	if word == "" || utf8.RuneCountInString(word) > 100 {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_QUERY", Message: "provide a word up to 100 characters"})
		return
	}
	items, err := h.svc.Word(word)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Get returns one kanji reference entry.
func (h *KanjiHandler) Get(c *gin.Context) {
	character := c.Param("character")
	if utf8.RuneCountInString(character) != 1 {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_KANJI", Message: "provide one kanji character"})
		return
	}
	entry, err := h.svc.Get(character)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, errorResponse{Code: "KANJI_NOT_FOUND", Message: "kanji was not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		return
	}
	c.JSON(http.StatusOK, entry)
}
