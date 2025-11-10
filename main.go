package main

import (
	"leavemaster/database"
	"leavemaster/handlers"
	"leavemaster/middleware"
	"leavemaster/websocket"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// To read env on prod
func init() {
	if os.Getenv("APP_ENV") != "production" {
		godotenv.Load()
		log.Println("🧩 Loaded .env for local development")
	} else {
		log.Println("🚀 Running in production mode")
	}
}

func main() {
	// Load environment variables
	godotenv.Load()

	// Initialize database
	database.InitDB()

	// Start WebSocket hub
	go websocket.HubInstance.Run()
	log.Println("🚀 WebSocket Hub Started!")

	// Create router
	r := gin.Default()

	// CORS configuration - VERY PERMISSIVE FOR DEVELOPMENT
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // For development only
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	// WebSocket route - DENGAN AUTH YANG BENAR
	r.GET("/ws", middleware.WebSocketAuth(), websocket.HandleWebSocket)
	log.Println("🔌 WebSocket route registered at /ws (WITH WEBSOCKET AUTH)")

	// Public routes
	r.POST("/api/login", handlers.Login)

	// Protected routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware()) // Auth middleware untuk semua API routes
	{
		// 🔐 PUBLIC ROUTES - Untuk semua role yang login
		api.GET("/profile", handlers.GetProfile)
		api.PUT("/profile", handlers.UpdateProfile)
		api.PUT("/change-password", handlers.ChangePassword)

		// 📋 LEAVE ROUTES
		api.POST("/leave", middleware.PermissionMiddleware("leave:write"), handlers.CreateLeaveRequest)
		api.GET("/leave/my-requests", middleware.PermissionMiddleware("leave:read"), handlers.GetMyLeaveRequests)
		api.GET("/leave/pending", middleware.PermissionMiddleware("leave:approve"), handlers.GetPendingLeaveRequests)
		api.PUT("/leave/:id/status", middleware.PermissionMiddleware("leave:approve"), handlers.UpdateLeaveStatus)

		// 📅 CALENDAR ROUTES
		api.GET("/calendar/events", handlers.GetCalendarEvents) // Semua bisa lihat calendar
		api.GET("/calendar/team", middleware.RoleMiddleware("super_admin", "admin", "manager"), handlers.GetTeamLeaveCalendar)

		// 📊 REPORTS ROUTES - Butuh reports:read permission
		api.GET("/reports/dashboard-stats", middleware.RoleMiddleware("super_admin", "admin", "manager"), handlers.GetDashboardStats)
		api.GET("/reports/department-stats", middleware.RoleMiddleware("super_admin", "admin", "manager"), handlers.GetDepartmentStats)
		api.GET("/reports/monthly-trends", middleware.RoleMiddleware("super_admin", "admin", "manager"), handlers.GetMonthlyTrends)
		api.GET("/reports/leave-type-distribution", middleware.RoleMiddleware("super_admin", "admin", "manager"), handlers.GetLeaveTypeDistribution)
		api.GET("/reports/recent-activities", middleware.RoleMiddleware("super_admin", "admin", "manager"), handlers.GetRecentActivities)

		// 🔥 USER MANAGEMENT ROUTES - Butuh users:read & users:write permissions
		api.GET("/employees", middleware.PermissionMiddleware("users:read"), handlers.GetEmployees)
		api.GET("/employees/:id", middleware.PermissionMiddleware("users:read"), handlers.GetEmployeeByID)
		api.POST("/employees", middleware.PermissionMiddleware("users:write"), handlers.CreateEmployee)
		api.PUT("/employees/:id", middleware.PermissionMiddleware("users:write"), handlers.UpdateEmployee)
		api.DELETE("/employees/:id", middleware.PermissionMiddleware("users:write"), handlers.DeleteEmployee)

		// 🛠️ HELPER ROUTES - Butuh users:read permission
		api.GET("/managers", middleware.PermissionMiddleware("users:read"), handlers.GetManagers)
		api.GET("/roles", middleware.PermissionMiddleware("users:read"), handlers.GetRoles)
		api.GET("/departments", middleware.PermissionMiddleware("users:read"), handlers.GetDepartments)

		// 🧪 TEST ENDPOINTS - Butuh users:write permission
		api.POST("/test-ws", middleware.PermissionMiddleware("users:write"), func(c *gin.Context) {
			// Get department dari user yang login
			departmentID, exists := c.Get("department_id")
			if !exists {
				departmentID = 1 // Default fallback
			}

			websocket.SendNewLeaveRequestNotification(
				"Test Employee",
				"annual",
				"2024-02-15",
				"2024-02-17",
				"Testing WebSocket from API",
				departmentID.(int),
			)
			c.JSON(200, gin.H{"message": "Test WebSocket notification sent!"})
		})

		api.GET("/debug/websocket", middleware.PermissionMiddleware("users:read"), func(c *gin.Context) {
			websocket.HubInstance.Mutex.RLock()
			defer websocket.HubInstance.Mutex.RUnlock()

			var clients []map[string]interface{}
			for client := range websocket.HubInstance.Clients {
				// Get department info untuk setiap client
				var departmentID int
				var departmentName string
				database.DB.QueryRow(`
					SELECT e.department_id, d.name 
					FROM employees e 
					LEFT JOIN departments d ON e.department_id = d.id 
					WHERE e.id = ?`, client.ID).Scan(&departmentID, &departmentName)

				clients = append(clients, map[string]interface{}{
					"employee_id":   client.ID,
					"is_manager":    client.IsManager,
					"department_id": departmentID,
					"department":    departmentName,
					"connected":     true,
				})
			}

			c.JSON(200, gin.H{
				"total_clients": len(clients),
				"clients":       clients,
			})
		})

		//check user context
		api.GET("/debug/user-context", func(c *gin.Context) {
			employeeID, _ := c.Get("employee_id")
			roleName, _ := c.Get("role_name")
			isManager, _ := c.Get("is_manager")
			departmentID, _ := c.Get("department_id")

			c.JSON(200, gin.H{
				"employee_id":   employeeID,
				"role_name":     roleName,
				"is_manager":    isManager,
				"department_id": departmentID,
			})
		})

		// Debug endpoint untuk test notification
		api.POST("/debug/test-notification", middleware.PermissionMiddleware("users:write"), func(c *gin.Context) {
			var req struct {
				DepartmentID int    `json:"department_id"`
				EmployeeName string `json:"employee_name"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			log.Printf("🧪 TEST NOTIFICATION: Employee %s, Department %d", req.EmployeeName, req.DepartmentID)

			websocket.SendNewLeaveRequestNotification(
				req.EmployeeName,
				"annual",
				"2024-02-15",
				"2024-02-17",
				"Test notification",
				req.DepartmentID,
			)

			c.JSON(200, gin.H{"message": "Test notification sent!"})
		})
	}

	// Health check dengan WebSocket info
	r.GET("/health", func(c *gin.Context) {
		websocket.HubInstance.Mutex.RLock()
		clientCount := len(websocket.HubInstance.Clients)
		websocket.HubInstance.Mutex.RUnlock()

		c.JSON(200, gin.H{
			"status":    "OK",
			"websocket": "Running",
			"clients":   clientCount,
			"message":   "Server is healthy and WebSocket is running",
		})
	})

	// Start server
	log.Println("🚀 Server starting on :8080")
	log.Println("🔌 WebSocket available at: ws://localhost:8080/ws (WITH AUTH)")
	log.Println("🏥 Health check at: http://localhost:8080/health")
	log.Println("🔐 Role-based access control: ENABLED")
	log.Println("🔑 Permission-based access: ENABLED")
	log.Println("🌐 WebSocket authentication: ENABLED")
	r.Run(":8080")
}
