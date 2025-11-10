package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"leavemaster/database"

	"github.com/gin-gonic/gin"
)

type CalendarEvent struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Start        string `json:"start"`
	End          string `json:"end"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	EmployeeName string `json:"employeeName"`
	Department   string `json:"department"`
	Color        string `json:"color"`
}

func GetCalendarEvents(c *gin.Context) {
	// Get user info dari context
	employeeID, _ := c.Get("employee_id")
	isManager, _ := c.Get("is_manager")
	userRole, _ := c.Get("role_name")
	userDeptID, hasDept := c.Get("department_id")

	log.Printf("📅 GetCalendarEvents - EmployeeID: %v, IsManager: %v, Role: %s, DeptID: %v",
		employeeID, isManager, userRole, userDeptID)

	var rows *sql.Rows
	var err error
	var query string
	var args []interface{}

	if isManager == true && hasDept {
		// Manager can see team leaves in their department - CASE INSENSITIVE
		query = `
            SELECT 
                lr.id, 
                lr.start_date, 
                lr.end_date, 
                lr.leave_type, 
                lr.status, 
                e.name as employee_name,
                COALESCE(d.name, 'General') as department
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            LEFT JOIN departments d ON e.department_id = d.id
            WHERE e.department_id = ? AND (LOWER(lr.status) = 'approved' OR LOWER(lr.status) = 'pending')
            ORDER BY lr.start_date`
		args = []interface{}{userDeptID}
	} else {
		// Regular employee can see only approved leaves - CASE INSENSITIVE
		query = `
            SELECT 
                lr.id, 
                lr.start_date, 
                lr.end_date, 
                lr.leave_type, 
                lr.status, 
                e.name as employee_name,
                COALESCE(d.name, 'General') as department
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            LEFT JOIN departments d ON e.department_id = d.id
            WHERE LOWER(lr.status) = 'approved'
            ORDER BY lr.start_date`
	}

	rows, err = database.DB.Query(query, args...)
	if err != nil {
		log.Printf("❌ Calendar query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load calendar events"})
		return
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var event CalendarEvent
		var startDate, endDate string
		var department sql.NullString

		err := rows.Scan(
			&event.ID, &startDate, &endDate, &event.Type, &event.Status,
			&event.EmployeeName, &department,
		)
		if err != nil {
			log.Printf("⚠️ Error scanning calendar row: %v", err)
			continue
		}

		// Handle nullable department
		if department.Valid {
			event.Department = department.String
		} else {
			event.Department = "General"
		}

		event.Title = event.EmployeeName + " - " + event.Type
		if strings.ToLower(event.Status) == "pending" {
			event.Title += " (Pending)"
		}

		// ✅ FIX: Format tanggal yang benar (hindari double time)
		event.Start = formatCalendarDate(startDate, true) // start: 00:00:00
		event.End = formatCalendarDate(endDate, false)    // end: 23:59:59

		event.Color = getEventColor(event.Type, event.Status)

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		log.Printf("⚠️ Row iteration error: %v", err)
	}

	log.Printf("✅ Loaded %d calendar events for user %s (role: %s)", len(events), employeeID, userRole)
	c.JSON(http.StatusOK, events)
}

func GetTeamLeaveCalendar(c *gin.Context) {
	// Get user info dari context
	employeeID, _ := c.Get("employee_id")
	isManager, _ := c.Get("is_manager")
	userRole, _ := c.Get("role_name")
	userDeptID, hasDept := c.Get("department_id")

	log.Printf("📅 GetTeamLeaveCalendar - EmployeeID: %v, IsManager: %v, Role: %s, DeptID: %v",
		employeeID, isManager, userRole, userDeptID)

	// Check if user is manager or admin
	// if !(isManager == true || userRole == "admin" || userRole == "super_admin") {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - manager role required"})
	// 	return
	// }

	var query string
	var args []interface{}

	if userRole == "admin" || userRole == "super_admin" {
		// Admin can see all team leaves - CASE INSENSITIVE
		query = `
            SELECT 
                lr.id, 
                lr.start_date, 
                lr.end_date, 
                lr.leave_type, 
                lr.status, 
                e.name as employee_name,
                COALESCE(d.name, 'General') as department
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            LEFT JOIN departments d ON e.department_id = d.id
            WHERE LOWER(lr.status) IN ('approved', 'pending')
            ORDER BY lr.start_date`
	} else if isManager == true && hasDept {
		// Manager can see team leaves in their department - CASE INSENSITIVE
		query = `
            SELECT 
                lr.id, 
                lr.start_date, 
                lr.end_date, 
                lr.leave_type, 
                lr.status, 
                e.name as employee_name,
                COALESCE(d.name, 'General') as department
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            LEFT JOIN departments d ON e.department_id = d.id
            WHERE e.department_id = ? AND LOWER(lr.status) IN ('approved', 'pending')
            ORDER BY lr.start_date`
		args = []interface{}{userDeptID}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// DEBUG: Test query dulu
	testQuery := `
        SELECT COUNT(*) as total_events
        FROM leave_requests lr
        JOIN employees e ON lr.employee_id = e.id
        WHERE e.department_id = ? AND LOWER(lr.status) IN ('approved', 'pending')`

	var totalEvents int
	err := database.DB.QueryRow(testQuery, userDeptID).Scan(&totalEvents)
	if err != nil {
		log.Printf("❌ Test query failed: %v", err)
	} else {
		log.Printf("🔍 DEBUG: Found %d events in department %v", totalEvents, userDeptID)
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		log.Printf("❌ Team calendar query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load team calendar"})
		return
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var event CalendarEvent
		var startDate, endDate string
		var department sql.NullString

		err := rows.Scan(
			&event.ID, &startDate, &endDate, &event.Type, &event.Status,
			&event.EmployeeName, &department,
		)
		if err != nil {
			log.Printf("⚠️ Error scanning team calendar row: %v", err)
			continue
		}

		// Handle nullable department
		if department.Valid {
			event.Department = department.String
		} else {
			event.Department = "General"
		}

		event.Title = event.EmployeeName + " - " + event.Type
		if strings.ToLower(event.Status) == "pending" {
			event.Title += " (Pending)"
		}

		// ✅ FIX: Format tanggal yang benar (hindari double time)
		event.Start = formatCalendarDate(startDate, true) // start: 00:00:00
		event.End = formatCalendarDate(endDate, false)    // end: 23:59:59

		event.Color = getEventColor(event.Type, event.Status)

		events = append(events, event)
		log.Printf("✅ EVENT: %s - %s (%s) %s to %s", event.EmployeeName, event.Type, event.Status, event.Start, event.End)
	}

	if err = rows.Err(); err != nil {
		log.Printf("⚠️ Row iteration error: %v", err)
	}

	log.Printf("✅ FINAL: Loaded %d team calendar events for user %s (dept: %v)", len(events), employeeID, userDeptID)
	c.JSON(http.StatusOK, events)
}

// ✅ NEW: Helper function untuk format tanggal dengan benar
func formatCalendarDate(dateStr string, isStart bool) string {
	// Jika sudah ada timezone (Z), bersihkan dulu
	cleanedDate := strings.TrimSuffix(dateStr, "Z")
	cleanedDate = strings.TrimSuffix(cleanedDate, "T00:00:00")
	cleanedDate = strings.TrimSuffix(cleanedDate, "T23:59:59")

	// Ambil hanya bagian tanggal (YYYY-MM-DD)
	parts := strings.Split(cleanedDate, "T")
	if len(parts) > 0 {
		cleanedDate = parts[0]
	}

	// Format sesuai kebutuhan
	if isStart {
		return cleanedDate + "T00:00:00"
	} else {
		return cleanedDate + "T23:59:59"
	}
}

func getEventColor(leaveType, status string) string {
	// Case insensitive untuk status
	if strings.ToLower(status) == "pending" {
		return "#FFA500" // Orange
	}

	switch strings.ToLower(leaveType) {
	case "annual":
		return "#3498db" // Blue
	case "sick":
		return "#e74c3c" // Red
	case "personal":
		return "#9b59b6" // Purple
	case "other":
		return "#2ecc71" // Green
	default:
		return "#95a5a6" // Gray
	}
}
