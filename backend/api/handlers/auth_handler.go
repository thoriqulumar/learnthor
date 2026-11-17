package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// AuthHandler handles registration and login.
type AuthHandler struct {
	svc *services.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register godoc
//
//	@Summary     Create a new account
//	@Tags        auth
//	@Accept      json
//	@Produce     json
//	@Param       body body     services.RegisterRequest true "Registration payload"
//	@Success     201  {object} services.AuthResponse
//	@Failure     400  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_PAYLOAD", Message: err.Error()})
		return
	}

	resp, err := h.svc.Register(req)
	if err != nil {
		if err == services.ErrBadRequest {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "EMAIL_OR_USERNAME_TAKEN", Message: "email or username is already in use"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login godoc
//
//	@Summary     Log in and receive a JWT
//	@Tags        auth
//	@Accept      json
//	@Produce     json
//	@Param       body body     services.LoginRequest true "Login payload"
//	@Success     200  {object} services.AuthResponse
//	@Failure     400  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_PAYLOAD", Message: err.Error()})
		return
	}

	resp, err := h.svc.Login(req)
	if err != nil {
		if err == services.ErrBadRequest {
			c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_CREDENTIALS", Message: "email or password is incorrect"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPreferences godoc
//
//	@Summary     Get user preferences
//	@Description Returns the authenticated user's preference set (e.g. the
//	             preferred_gloss field controlling which meaning is shown
//	             first on study cards).
//	@Tags        preferences
//	@Produce     json
//	@Success     200 {object} services.PreferencesResponse
//	@Failure     500 {object} errorResponse
//	@Router      /me/preferences [get]
func (h *AuthHandler) GetPreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	resp, err := h.svc.GetPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
		})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdatePreferences godoc
//
//	@Summary     Update user preferences
//	@Description Updates the supplied preference fields. Only the provided
//	             fields are touched. preferred_gloss must be one of
//	             "en", "id", "both".
//	@Tags        preferences
//	@Accept      json
//	@Produce     json
//	@Param       body body     services.UpdatePreferencesRequest true "Preferences"
//	@Success     200  {object} services.PreferencesResponse
//	@Failure     400  {object} errorResponse
//	@Failure     500  {object} errorResponse
//	@Router      /me/preferences [put]
func (h *AuthHandler) UpdatePreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req services.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_PAYLOAD",
			Message: err.Error(),
		})
		return
	}

	resp, err := h.svc.UpdatePreferences(userID, req)
	if err != nil {
		switch err {
		case services.ErrBadRequest:
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "INVALID_PREFERENCE",
				Message: "preferred_gloss must be one of: en, id, both",
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
