package handlers

import (
	"log"
	"net/http"
	"time"

	"leavemaster/database"

	"github.com/gin-gonic/gin"
)

type DashboardStats struct {
	TotalEmployees      int     `json:"total_employees"`
	PendingRequests     int     `json:"pending_requests"`
	ApprovedThisMonth   int     `json:"approved_this_month"`
	RejectedThisMonth   int     `json:"rejected_this_month"`
	TotalLeavesThisYear int     `json:"total_leaves_this_year"`
	LeaveUtilization    float64 `json:"leave_utilization"`
	AvgProcessingTime   float64 `json:"avg_processing_time"`
}

type DepartmentStats struct {
	Department      string  `json:"department"`
	TotalEmployees  int     `json:"total_employees"`
	TotalLeaves     int     `json:"total_leaves"`
	AvgLeaveDays    float64 `json:"avg_leave_days"`
	UtilizationRate float64 `json:"utilization_rate"`
	PendingCount    int     `json:"pending_count"`
}

type MonthlyTrend struct {
	Month    string `json:"month"`
	Leaves   int    `json:"leaves"`
	Approved int    `json:"approved"`
	Pending  int    `json:"pending"`
	Rejected int    `json:"rejected"`
}

type LeaveTypeDistribution struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Color string `json:"color"`
}

// Get Enhanced Dashboard Stats
// Get Enhanced Dashboard Stats - WITH ROLE-BASED DATA
func GetDashboardStats(c *gin.Context) {
	// Get user info dari context
	userRole, _ := c.Get("role_name")
	userDeptID, hasDept := c.Get("department_id")

	log.Printf("🔍 GetDashboardStats - User: Role=%s, DeptID=%v", userRole, userDeptID)

	var stats DashboardStats
	var whereClause string
	var args []interface{}

	// Filter data berdasarkan role
	if userRole == "manager" && hasDept {
		whereClause = " AND e.department_id = ?"
		args = []interface{}{userDeptID}
		log.Printf("👤 Manager - Filtering dashboard data by department_id: %v", userDeptID)
	}
	// Untuk admin/super_admin, tidak ada filter (lihat semua data)

	// Total employees dengan filter
	totalEmployeesQuery := "SELECT COUNT(*) FROM employees e WHERE 1=1" + whereClause
	database.DB.QueryRow(totalEmployeesQuery, args...).Scan(&stats.TotalEmployees)

	// Pending requests dengan filter
	pendingQuery := `
        SELECT COUNT(*) FROM leave_requests lr 
        JOIN employees e ON lr.employee_id = e.id 
        WHERE lr.status = 'pending'` + whereClause
	database.DB.QueryRow(pendingQuery, args...).Scan(&stats.PendingRequests)

	// Approved this month dengan filter
	currentMonth := time.Now().Format("2006-01")
	approvedMonthQuery := `
        SELECT COUNT(*) FROM leave_requests lr 
        JOIN employees e ON lr.employee_id = e.id 
        WHERE lr.status = 'approved' AND DATE_FORMAT(lr.created_at, '%Y-%m') = ?` + whereClause
	database.DB.QueryRow(approvedMonthQuery, append([]interface{}{currentMonth}, args...)...).Scan(&stats.ApprovedThisMonth)

	// Rejected this month dengan filter
	rejectedMonthQuery := `
        SELECT COUNT(*) FROM leave_requests lr 
        JOIN employees e ON lr.employee_id = e.id 
        WHERE lr.status = 'rejected' AND DATE_FORMAT(lr.created_at, '%Y-%m') = ?` + whereClause
	database.DB.QueryRow(rejectedMonthQuery, append([]interface{}{currentMonth}, args...)...).Scan(&stats.RejectedThisMonth)

	// Total leaves this year dengan filter
	currentYear := time.Now().Format("2006")
	totalLeavesQuery := `
        SELECT COUNT(*) FROM leave_requests lr 
        JOIN employees e ON lr.employee_id = e.id 
        WHERE DATE_FORMAT(lr.created_at, '%Y') = ?` + whereClause
	database.DB.QueryRow(totalLeavesQuery, append([]interface{}{currentYear}, args...)...).Scan(&stats.TotalLeavesThisYear)

	// Leave utilization rate dengan filter
	var totalLeaves, totalPossibleLeaves int
	utilizationQuery := `
        SELECT COALESCE(SUM(lr.total_days), 0) 
        FROM leave_requests lr 
        JOIN employees e ON lr.employee_id = e.id 
        WHERE lr.status = 'approved'` + whereClause
	database.DB.QueryRow(utilizationQuery, args...).Scan(&totalLeaves)

	possibleLeavesQuery := "SELECT COALESCE(SUM(total_leave_days), 0) FROM employees e WHERE 1=1" + whereClause
	database.DB.QueryRow(possibleLeavesQuery, args...).Scan(&totalPossibleLeaves)

	if totalPossibleLeaves > 0 {
		stats.LeaveUtilization = (float64(totalLeaves) / float64(totalPossibleLeaves)) * 100
	}

	// Average processing time dengan filter
	avgProcessingQuery := `
        SELECT COALESCE(AVG(TIMESTAMPDIFF(HOUR, lr.created_at, lr.approved_at)), 0) 
        FROM leave_requests lr 
        JOIN employees e ON lr.employee_id = e.id 
        WHERE lr.status IN ('approved', 'rejected') AND lr.approved_at IS NOT NULL` + whereClause
	database.DB.QueryRow(avgProcessingQuery, args...).Scan(&stats.AvgProcessingTime)

	log.Printf("✅ %s retrieved dashboard stats for their scope", userRole)
	c.JSON(http.StatusOK, stats)
}

// Get Enhanced Department Stats - WITH ROLE-BASED FILTERING
func GetDepartmentStats(c *gin.Context) {
	// Get user info dari context
	userRole, _ := c.Get("role_name")
	userDeptID, hasDept := c.Get("department_id")

	log.Printf("🔍 GetDepartmentStats - User: Role=%s, DeptID=%v", userRole, userDeptID)

	var query string
	var args []interface{}

	// Super Admin & Admin - lihat semua departments
	if userRole == "super_admin" || userRole == "admin" || userRole == "superadmin" {
		query = `
            SELECT 
                COALESCE(d.name, 'No Department') as department,
                COUNT(DISTINCT e.id) as total_employees,
                COUNT(CASE WHEN lr.status = 'approved' THEN lr.id END) as total_leaves,
                COALESCE(AVG(CASE WHEN lr.status = 'approved' THEN lr.total_days END), 0) as avg_leave_days,
                COALESCE((SUM(CASE WHEN lr.status = 'approved' THEN COALESCE(lr.total_days, 0) ELSE 0 END) / NULLIF(COUNT(DISTINCT e.id) * 12.0, 0)) * 100, 0) as utilization_rate,
                COUNT(CASE WHEN lr.status = 'pending' THEN lr.id END) as pending_count
            FROM employees e
            LEFT JOIN departments d ON e.department_id = d.id
            LEFT JOIN leave_requests lr ON e.id = lr.employee_id
            GROUP BY d.id, d.name
            ORDER BY total_leaves DESC`
		log.Printf("👑 Admin/SuperAdmin - Viewing ALL departments")
	} else if userRole == "manager" && hasDept {
		// Manager - hanya lihat departmentnya sendiri
		query = `
            SELECT 
                COALESCE(d.name, 'No Department') as department,
                COUNT(DISTINCT e.id) as total_employees,
                COUNT(CASE WHEN lr.status = 'approved' THEN lr.id END) as total_leaves,
                COALESCE(AVG(CASE WHEN lr.status = 'approved' THEN lr.total_days END), 0) as avg_leave_days,
                COALESCE((SUM(CASE WHEN lr.status = 'approved' THEN COALESCE(lr.total_days, 0) ELSE 0 END) / NULLIF(COUNT(DISTINCT e.id) * 12.0, 0)) * 100, 0) as utilization_rate,
                COUNT(CASE WHEN lr.status = 'pending' THEN lr.id END) as pending_count
            FROM employees e
            LEFT JOIN departments d ON e.department_id = d.id
            LEFT JOIN leave_requests lr ON e.id = lr.employee_id
            WHERE e.department_id = ?
            GROUP BY d.id, d.name
            ORDER BY total_leaves DESC`
		args = []interface{}{userDeptID}
		log.Printf("👤 Manager - Filtering by department_id: %v", userDeptID)
	} else {
		log.Printf("❌ Access denied - Role: %s, HasDept: %v", userRole, hasDept)
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"message": "You don't have permission to view department statistics",
		})
		return
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		log.Printf("❌ Query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve department statistics"})
		return
	}
	defer rows.Close()

	var stats []DepartmentStats
	totalPending := 0

	for rows.Next() {
		var s DepartmentStats
		var avgDays *float64
		err := rows.Scan(&s.Department, &s.TotalEmployees, &s.TotalLeaves, &avgDays, &s.UtilizationRate, &s.PendingCount)
		if err != nil {
			log.Printf("⚠️ Error scanning row: %v", err)
			continue
		}

		if avgDays != nil {
			s.AvgLeaveDays = *avgDays
		} else {
			s.AvgLeaveDays = 0
		}

		totalPending += s.PendingCount
		stats = append(stats, s)
	}

	if err = rows.Err(); err != nil {
		log.Printf("⚠️ Row iteration error: %v", err)
	}

	log.Printf("✅ %s retrieved %d department stats with %d total pending requests", userRole, len(stats), totalPending)
	c.JSON(http.StatusOK, stats)
}

// Get Monthly Trends with Details
func GetMonthlyTrends(c *gin.Context) {
	query := `
		SELECT 
			DATE_FORMAT(created_at, '%Y-%m') as month,
			COUNT(*) as total_leaves,
			SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) as approved,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected
		FROM leave_requests 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 12 MONTH)
		GROUP BY DATE_FORMAT(created_at, '%Y-%m')
		ORDER BY month DESC`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var trends []MonthlyTrend
	for rows.Next() {
		var t MonthlyTrend
		err := rows.Scan(&t.Month, &t.Leaves, &t.Approved, &t.Pending, &t.Rejected)
		if err != nil {
			continue
		}
		trends = append(trends, t)
	}

	// Reverse untuk chronological order
	for i, j := 0, len(trends)-1; i < j; i, j = i+1, j-1 {
		trends[i], trends[j] = trends[j], trends[i]
	}

	c.JSON(http.StatusOK, trends)
}

// Get Leave Type Distribution
func GetLeaveTypeDistribution(c *gin.Context) {
	query := `
		SELECT 
			leave_type,
			COUNT(*) as count
		FROM leave_requests 
		WHERE status = 'approved'
		GROUP BY leave_type
		ORDER BY count DESC`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var distribution []LeaveTypeDistribution
	colors := map[string]string{
		"annual":   "#3498db",
		"sick":     "#e74c3c",
		"personal": "#9b59b6",
		"other":    "#f39c12",
	}

	for rows.Next() {
		var d LeaveTypeDistribution
		err := rows.Scan(&d.Type, &d.Count)
		if err != nil {
			continue
		}
		d.Color = colors[d.Type]
		distribution = append(distribution, d)
	}

	c.JSON(http.StatusOK, distribution)
}

// Get Recent Activities
// Get Recent Activities - WITH ROLE-BASED FILTERING
func GetRecentActivities(c *gin.Context) {
	// Get user info dari context
	userRole, _ := c.Get("role_name")
	userDeptID, hasDept := c.Get("department_id")

	log.Printf("🔍 GetRecentActivities - User: Role=%s, DeptID=%v", userRole, userDeptID)

	var query string
	var args []interface{}

	if userRole == "super_admin" || userRole == "admin" || userRole == "superadmin" {
		query = `
            SELECT 
                e.name as employee_name,
                lr.leave_type,
                lr.status,
                lr.created_at,
                lr.start_date,
                lr.end_date,
                COALESCE(m.name, 'System') as action_by
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            LEFT JOIN employees m ON lr.approved_by = m.id
            ORDER BY lr.created_at DESC
            LIMIT 10`
		log.Printf("👑 Admin/SuperAdmin - Viewing ALL activities")
	} else if userRole == "manager" && hasDept {
		query = `
            SELECT 
                e.name as employee_name,
                lr.leave_type,
                lr.status,
                lr.created_at,
                lr.start_date,
                lr.end_date,
                COALESCE(m.name, 'System') as action_by
            FROM leave_requests lr
            JOIN employees e ON lr.employee_id = e.id
            LEFT JOIN employees m ON lr.approved_by = m.id
            WHERE e.department_id = ?
            ORDER BY lr.created_at DESC
            LIMIT 10`
		args = []interface{}{userDeptID}
		log.Printf("👤 Manager - Filtering activities by department_id: %v", userDeptID)
	} else {
		log.Printf("❌ Access denied - Role: %s, HasDept: %v", userRole, hasDept)
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"message": "You don't have permission to view recent activities",
		})
		return
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var activities []map[string]interface{}
	for rows.Next() {
		var employeeName, leaveType, status, createdAt, startDate, endDate, actionBy string
		err := rows.Scan(&employeeName, &leaveType, &status, &createdAt, &startDate, &endDate, &actionBy)
		if err != nil {
			continue
		}

		activity := map[string]interface{}{
			"employee_name": employeeName,
			"leave_type":    leaveType,
			"status":        status,
			"created_at":    createdAt,
			"start_date":    startDate,
			"end_date":      endDate,
			"action_by":     actionBy,
		}
		activities = append(activities, activity)
	}

	log.Printf("✅ %s retrieved %d recent activities", userRole, len(activities))
	c.JSON(http.StatusOK, activities)
}
