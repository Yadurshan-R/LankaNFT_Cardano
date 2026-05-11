package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthRequired is a Gin middleware that protects routes
// It reads the JWT from the httpOnly cookie and validates it
// If valid, it sets the user_id in the Gin context for handlers to use
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the cookie
		tokenString, err := c.Cookie("auth_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "not authenticated — please log in",
			})
			c.Abort() // stop the request from reaching the handler
			return
		}

		// Parse and validate the JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Make sure the signing method is HMAC (what we used)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token — please log in again",
			})
			c.Abort()
			return
		}

		// Extract user_id from token claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token: missing user_id",
			})
			c.Abort()
			return
		}

		// Set user_id in context so any handler can access it with:
		// userID := c.GetString("user_id")
		c.Set("user_id", userID)

		// Continue to the next handler
		c.Next()
	}
}
