package middlewares

import (
	"net/http"
	"os"
	"strings"

	logger "go-multi-chat-api/src/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// RequiresRoleMiddleware creates a middleware that checks if the user has the required role
func RequiresRoleMiddleware(requiredRole string, loggerInstance *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token not provided"})
			c.Abort()
			return
		}

		accessSecret := os.Getenv("JWT_ACCESS_SECRET_KEY")
		if accessSecret == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT_ACCESS_SECRET_KEY not configured"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return []byte(accessSecret), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Check token expiration
		if exp, ok := claims["exp"].(float64); ok {
			if int64(exp) < jwt.TimeFunc().Unix() {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Check token type
		if t, ok := claims["type"].(string); ok {
			if t != "access" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Token type mismatch"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Missing token type"})
			c.Abort()
			return
		}

		// Get user ID from token
		userID, ok := claims["id"].(float64)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		// Get user role from token claims
		userRole, ok := claims["role"].(string)
		if !ok {
			loggerInstance.Error("Role claim missing from token", zap.Float64("userID", userID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid token: missing role claim"})
			c.Abort()
			return
		}

		// Check if user has the required role
		if userRole != requiredRole && !(requiredRole == "member" && userRole == "admin") {
			// Admin can do everything a member can do, but not vice versa
			loggerInstance.Warn("User does not have required role",
				zap.String("requiredRole", requiredRole),
				zap.String("userRole", userRole),
				zap.Float64("userID", userID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		// Store user ID and role in context for later use
		c.Set("userID", int(userID))
		c.Set("userRole", userRole)
		c.Next()
	}
}
