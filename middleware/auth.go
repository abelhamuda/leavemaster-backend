package middleware

import (
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var jwtSecret = []byte("your-secret-key")

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		// ✅ STANDARDIZE: Gunakan lowercase dengan underscore
		c.Set("employee_id", int(claims["employee_id"].(float64)))
		c.Set("is_manager", claims["is_manager"].(bool))
		c.Set("role_name", claims["role_name"].(string))

		// TAMBAHKAN INI untuk WebSocket department filtering
		if departmentID, ok := claims["department_id"].(float64); ok {
			c.Set("department_id", int(departmentID))
		}

		c.Next()
	}
}
