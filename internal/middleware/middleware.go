package middleware

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	"github.com/gin-gonic/gin"
)

var (
	missingAPIKeyError = errors.New("Error: missing API key")
	invalidAPIKeyError = errors.New("Error: invalid API key")
)

func AuthMiddleware(c *gin.Context) {
	tokenStr := c.GetHeader("Authorization")

	tokenStr, found := strings.CutPrefix(tokenStr, "Bearer ")
	if !found {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Error: incorrectly provided token"})
		return
	}

	id, role, email, err := auth.ValidateJWT(tokenStr)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": auth.ParsingTokenError.Error()})
		return
	}

	c.Set("user_id", id)
	c.Set("email", email)
	c.Set("role", role)

	c.Next()
}

func RequireRole(role byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleAny, _ := c.Get("role")
		userRole := userRoleAny.(byte)

		if userRole != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Error: forbidden"})
			return
		}

		c.Next()
	}
}

func APIKeyAuthMiddleware(c *gin.Context) {
	apiKey := c.GetHeader("X-API-KEY")

	if apiKey == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": missingAPIKeyError.Error(),
		})
		return
	}

	secretKey := os.Getenv("API_KEY")

	if apiKey != secretKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": invalidAPIKeyError.Error(),
		})
		return
	}

	c.Next()
}
