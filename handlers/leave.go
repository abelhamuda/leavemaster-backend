package handlers

import (
	"fmt"
	"log"
	"net/http"

	"leavemaster/database"
	"leavemaster/models"
	"leavemaster/services"
	"leavemaster/websocket"

	"github.com/gin-gonic/gin"
)

var emailService = services.NewEmailService()

func CreateLeaveRequest(c *gin.Context) {
	var leaveReq models.LeaveRequest
	if err := c.ShouldBindJSON(&leaveReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	employeeID := c.GetInt("employee_id")
	leaveReq.EmployeeID = employeeID
	leaveReq.Status = "pending"

	// Get employee details termasuk department
	var employeeName, employeeEmail string
	var managerID *int
	var employeeDeptID int
	err := database.DB.QueryRow("SELECT name, email, manager_id, department_id FROM employees WHERE id = ?", employeeID).
		Scan(&employeeName, &employeeEmail, &managerID, &employeeDeptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Employee not found"})
		return
	}

	query := `INSERT INTO leave_requests 
        (employee_id, leave_type, start_date, end_date, total_days, reason, status) 
        VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := database.DB.Exec(query,
		leaveReq.EmployeeID, leaveReq.LeaveType, leaveReq.StartDate,
		leaveReq.EndDate, leaveReq.TotalDays, leaveReq.Reason, leaveReq.Status)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	leaveReq.ID = int(id)

	// DEBUG: Log untuk troubleshooting
	fmt.Printf("🆕 DEBUG: Employee %s (ID: %d, Dept: %d) created leave request. Manager ID: %v\n",
		employeeName, employeeID, employeeDeptID, managerID)

	// Send email notification to manager jika ada manager
	if managerID != nil {
		go func() {
			var managerEmail, managerName string
			err := database.DB.QueryRow("SELECT email, name FROM employees WHERE id = ?", *managerID).
				Scan(&managerEmail, &managerName)

			if err != nil {
				fmt.Printf("❌ DEBUG: Manager not found for ID: %d, error: %v\n", *managerID, err)
				return
			}

			fmt.Printf("📧 DEBUG: Sending email to manager: %s (%s)\n", managerName, managerEmail)

			if managerEmail != "" {
				emailService.SendLeaveRequestNotification(
					managerEmail,
					managerName,
					employeeName,
					leaveReq.LeaveType,
					leaveReq.StartDate,
					leaveReq.EndDate,
					leaveReq.Reason,
				)
			}
		}()
	} else {
		fmt.Printf("⚠️ DEBUG: No manager assigned for employee %s\n", employeeName)
	}

	// WebSocket notification untuk MANAGER DI DEPARTMENT YANG SAMA
	go func() {
		fmt.Printf("🔔 DEBUG: Sending WebSocket notification to managers in department %d\n", employeeDeptID)
		websocket.SendNewLeaveRequestNotification(
			employeeName,
			leaveReq.LeaveType,
			leaveReq.StartDate,
			leaveReq.EndDate,
			leaveReq.Reason,
			employeeDeptID, // Kirim department ID
		)

		// Debug info connected clients
		clientsInfo := websocket.GetConnectedClientsInfo()
		fmt.Printf("👥 DEBUG: Connected clients - Total: %d, Managers: %d, Employees: %d\n",
			clientsInfo["total_clients"], clientsInfo["manager_count"], clientsInfo["employee_count"])
	}()

	c.JSON(http.StatusCreated, leaveReq)
}

func GetMyLeaveRequests(c *gin.Context) {
	employeeID := c.GetInt("employee_id")

	query := `SELECT lr.*, e.name as employee_name 
		FROM leave_requests lr 
		JOIN employees e ON lr.employee_id = e.id 
		WHERE lr.employee_id = ? 
		ORDER BY lr.created_at DESC`

	rows, err := database.DB.Query(query, employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var leaveRequests []models.LeaveRequest
	for rows.Next() {
		var lr models.LeaveRequest
		err := rows.Scan(
			&lr.ID, &lr.EmployeeID, &lr.LeaveType, &lr.StartDate, &lr.EndDate,
			&lr.TotalDays, &lr.Reason, &lr.Status, &lr.ApprovedBy, &lr.ApprovedAt,
			&lr.CreatedAt, &lr.EmployeeName,
		)
		if err != nil {
			continue
		}
		leaveRequests = append(leaveRequests, lr)
	}

	c.JSON(http.StatusOK, leaveRequests)
}

func GetPendingLeaveRequests(c *gin.Context) {
	if !c.GetBool("is_manager") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	managerID := c.GetInt("employee_id")

	// Get manager's department
	var managerDeptID int
	err := database.DB.QueryRow("SELECT department_id FROM employees WHERE id = ?", managerID).
		Scan(&managerDeptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Manager department not found"})
		return
	}

	log.Printf("🔍 Manager %d loading pending requests for department %d", managerID, managerDeptID)

	// Hanya tampilkan requests dari department yang sama dengan manager
	query := `SELECT lr.*, e.name as employee_name 
		FROM leave_requests lr 
		JOIN employees e ON lr.employee_id = e.id 
		WHERE lr.status = 'pending' 
		AND e.department_id = ?
		ORDER BY lr.created_at DESC`

	rows, err := database.DB.Query(query, managerDeptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var leaveRequests []models.LeaveRequest
	for rows.Next() {
		var lr models.LeaveRequest
		err := rows.Scan(
			&lr.ID, &lr.EmployeeID, &lr.LeaveType, &lr.StartDate, &lr.EndDate,
			&lr.TotalDays, &lr.Reason, &lr.Status, &lr.ApprovedBy, &lr.ApprovedAt,
			&lr.CreatedAt, &lr.EmployeeName,
		)
		if err != nil {
			continue
		}
		leaveRequests = append(leaveRequests, lr)
	}

	log.Printf("✅ Found %d pending leave requests for department %d", len(leaveRequests), managerDeptID)
	c.JSON(http.StatusOK, leaveRequests)
}

func UpdateLeaveStatus(c *gin.Context) {
	if !c.GetBool("is_manager") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	leaveID := c.Param("id")
	var request struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	managerID := c.GetInt("employee_id")
	query := `UPDATE leave_requests SET status = ?, approved_by = ?, approved_at = NOW() WHERE id = ?`

	_, err := database.DB.Exec(query, request.Status, managerID, leaveID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update remaining leave days if approved
	if request.Status == "approved" {
		var totalDays, employeeID int
		var leaveType, startDate, endDate, employeeName string
		database.DB.QueryRow(`
            SELECT e.id, e.name, lr.total_days, lr.leave_type, lr.start_date, lr.end_date 
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            WHERE lr.id = ?`, leaveID).
			Scan(&employeeID, &employeeName, &totalDays, &leaveType, &startDate, &endDate)

		updateQuery := `UPDATE employees SET remaining_leave_days = remaining_leave_days - ? WHERE id = ?`
		database.DB.Exec(updateQuery, totalDays, employeeID)

		// Send email to employee
		go func() {
			var employeeEmail string
			database.DB.QueryRow(`SELECT email FROM employees WHERE id = ?`, employeeID).
				Scan(&employeeEmail)

			if employeeEmail != "" {
				emailService.SendLeaveStatusNotification(
					employeeEmail,
					employeeName,
					"approved",
					leaveType,
					startDate,
					endDate,
				)
			}
		}()

		// WebSocket notification untuk employee YANG BERSANGKUTAN
		go func() {
			websocket.SendLeaveStatusNotification(
				employeeName,
				"approved",
				leaveType,
				employeeID, // Kirim ke employee spesifik
			)
		}()
	} else if request.Status == "rejected" {
		// Send rejection email dan notification
		go func() {
			var employeeEmail, employeeName, leaveType, startDate, endDate string
			var employeeID int
			database.DB.QueryRow(`
                SELECT e.id, e.email, e.name, lr.leave_type, lr.start_date, lr.end_date 
                FROM employees e 
                JOIN leave_requests lr ON e.id = lr.employee_id 
                WHERE lr.id = ?`, leaveID).
				Scan(&employeeID, &employeeEmail, &employeeName, &leaveType, &startDate, &endDate)

			if employeeEmail != "" {
				// Send email
				emailService.SendLeaveStatusNotification(
					employeeEmail,
					employeeName,
					"rejected",
					leaveType,
					startDate,
					endDate,
				)

				// WebSocket notification untuk employee YANG BERSANGKUTAN
				websocket.SendLeaveStatusNotification(
					employeeName,
					"rejected",
					leaveType,
					employeeID, // Kirim ke employee spesifik
				)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave request updated successfully"})
}
