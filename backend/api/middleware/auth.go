package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/learnthor/backend/services"
)

// RequireAuth validates the Bearer JWT in the Authorization header and stores
// the authenticated user's UUID under the "user_id" key in the Gin context.
//
// Downstream middleware (e.g. RequireModuleAccess) reads "user_id" from context.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHENTICATED",
				"message": "Authorization: Bearer <token> header is required",
			})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := services.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_TOKEN",
				"message": "token is invalid or expired",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
