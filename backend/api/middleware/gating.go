package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// contextKey is the key under which the resolved module UUID is stored in the
// Gin context so downstream handlers don't need to parse it a second time.
const contextKeyModuleID = "module_id"

// contextKeyUserID is the key where upstream auth middleware stores the
// authenticated user's UUID. The gating middleware reads it from here.
const contextKeyUserID = "user_id"

// RequireModuleAccess returns a Gin middleware that enforces level gating.
//
// It expects:
//   - A route parameter named "module_id" (UUID string).
//   - The string key contextKeyUserID ("user_id") to be set in the Gin context
//     by a preceding authentication middleware.
//
// On success it calls c.Next(); the resolved module UUID is stored under
// contextKeyModuleID ("module_id") for use by the downstream handler.
//
// On failure it aborts with:
//   - 400 if the module_id param is not a valid UUID.
//   - 403 if the user has not passed the predecessor module.
//   - 404 if the module does not exist.
//   - 500 for unexpected database errors.
func RequireModuleAccess(gSvc *services.GatingService) gin.HandlerFunc {
	return func(c *gin.Context) {

		// ── Parse :module_id route param ──────────────────────────────────────
		moduleID, err := uuid.Parse(c.Param("module_id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":    "INVALID_MODULE_ID",
				"message": "module_id must be a valid UUID",
			})
			return
		}

		// ── Retrieve authenticated user from context ───────────────────────────
		// Set by the auth middleware that runs before this one.
		rawUserID, exists := c.Get(contextKeyUserID)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHENTICATED",
				"message": "authentication required",
			})
			return
		}
		userID, ok := rawUserID.(uuid.UUID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "an unexpected error occurred",
			})
			return
		}

		// ── Run the gate check ────────────────────────────────────────────────
		result, err := gSvc.CheckModuleAccess(userID, moduleID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrNotFound):
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"code":    "NOT_FOUND",
					"message": err.Error(),
				})
			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "an unexpected error occurred",
				})
			}
			return
		}

		if !result.Allowed {
			// Premium-gated modules get a distinct code so the frontend shows
			// an upgrade CTA instead of a progression-lock message.
			if result.RequiresPremium {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    "UPGRADE_REQUIRED",
					"message": result.Reason,
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "MODULE_LOCKED",
				"message": result.Reason,
				"details": gin.H{
					"required_module_id":    result.RequiredModuleID,
					"required_module_title": result.RequiredModuleTitle,
					"your_high_score":       result.UserHighScore,
				},
			})
			return
		}

		// Store the parsed UUID so the handler doesn't re-parse it.
		c.Set(contextKeyModuleID, moduleID)
		c.Next()
	}
}
