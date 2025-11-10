package middleware

import (
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func WebSocketAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from query parameter
		tokenString := c.Query("token")
		if tokenString == "" {
			// Try Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				tokenString = strings.Replace(authHeader, "Bearer ", "", 1)
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "WebSocket token required"})
			c.Abort()
			return
		}

		// Parse token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-secret-key"), nil // PASTIKAN SAMA DENGAN SECRET KEY LAINNYA
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid WebSocket token"})
			c.Abort()
			return
		}

		// Set user info in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("employeeID", int(claims["employee_id"].(float64)))
			c.Set("isManager", claims["is_manager"].(bool))
			c.Set("role_name", claims["role_name"].(string))

			// TAMBAHKAN DEPARTMENT_ID
			if departmentID, ok := claims["department_id"].(float64); ok {
				c.Set("department_id", int(departmentID))
			}

			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
		}
	}
}
