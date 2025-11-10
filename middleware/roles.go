package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RoleMiddleware - Check if user has required role
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role_name")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role information not found"})
			c.Abort()
			return
		}

		// Debug log
		log.Printf("🔐 Role check - User role: %s, Allowed: %v", userRole, allowedRoles)

		// Convert user role to lowercase for consistent comparison
		userRoleStr := strings.ToLower(strings.TrimSpace(userRole.(string)))

		// Super admin can access everything
		if userRoleStr == "super_admin" {
			c.Next()
			return
		}

		// Check if user's role is in allowed roles
		for _, role := range allowedRoles {
			// Case insensitive comparison
			if userRoleStr == strings.ToLower(role) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":     "Access denied. Required roles: " + strings.Join(allowedRoles, ", "),
			"your_role": userRole,
		})
		c.Abort()
	}
}

// PermissionMiddleware - Check specific permissions
func PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role_name")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role information not found"})
			c.Abort()
			return
		}

		// Debug log
		log.Printf("🔐 Permission check - User role: %s, Required: %s", userRole, requiredPermission)

		// Convert user role to lowercase for consistent comparison
		userRoleStr := strings.ToLower(strings.TrimSpace(userRole.(string)))

		// Super admin has all permissions
		if userRoleStr == "super_admin" {
			c.Next()
			return
		}

		// Define role permissions
		rolePermissions := map[string][]string{
			"admin":    {"users:read", "users:write", "reports:read", "leave:approve"},
			"manager":  {"users:read", "leave:approve", "reports:read"},
			"employee": {"leave:read", "leave:write"},
		}

		// Check if user's role has the required permission
		if permissions, exists := rolePermissions[userRoleStr]; exists {
			for _, permission := range permissions {
				if permission == requiredPermission {
					c.Next()
					return
				}
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":     "Permission denied. Required: " + requiredPermission,
			"your_role": userRole,
		})
		c.Abort()
	}
}
