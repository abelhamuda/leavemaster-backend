package handlers

import (
	"log"
	"net/http"

	"leavemaster/database"
	"leavemaster/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetEmployees - Get all employees dengan details (FROM employee.go)
func GetEmployees(c *gin.Context) {
	query := `
		SELECT 
			e.id, e.employee_id, e.name, e.email, e.position,
			e.department_id, d.name as department_name,
			e.role_id, r.name as role_name,
			e.total_leave_days, e.remaining_leave_days,
			e.is_manager, e.is_active, e.manager_id,
			m.name as manager_name, e.created_at
		FROM employees e
		LEFT JOIN departments d ON e.department_id = d.id
		LEFT JOIN roles r ON e.role_id = r.id
		LEFT JOIN employees m ON e.manager_id = m.id
		ORDER BY e.name`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var emp models.Employee
		var deptID, roleID, managerID *int
		var deptName, roleName, managerName *string

		err := rows.Scan(
			&emp.ID, &emp.EmployeeID, &emp.Name, &emp.Email, &emp.Position,
			&deptID, &deptName, &roleID, &roleName,
			&emp.TotalLeaveDays, &emp.RemainingLeaveDays,
			&emp.IsManager, &emp.IsActive, &managerID, &managerName, &emp.CreatedAt,
		)
		if err != nil {
			continue
		}

		if deptName != nil {
			emp.DepartmentName = *deptName
		}
		if roleName != nil {
			emp.RoleName = *roleName
		}
		if managerName != nil {
			emp.ManagerName = *managerName
		}
		if deptID != nil {
			emp.DepartmentID = deptID
		}
		if roleID != nil {
			emp.RoleID = roleID
		}

		employees = append(employees, emp)
	}

	c.JSON(http.StatusOK, employees)
}

// GetEmployeeByID - Get employee by ID (FROM employee.go)
func GetEmployeeByID(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT 
			e.id, e.employee_id, e.name, e.email, e.position,
			e.department_id, d.name as department_name,
			e.role_id, r.name as role_name,
			e.total_leave_days, e.remaining_leave_days,
			e.is_manager, e.is_active, e.manager_id,
			m.name as manager_name, e.created_at
		FROM employees e
		LEFT JOIN departments d ON e.department_id = d.id
		LEFT JOIN roles r ON e.role_id = r.id
		LEFT JOIN employees m ON e.manager_id = m.id
		WHERE e.id = ?`

	var emp models.Employee
	var deptID, roleID, managerID *int
	var deptName, roleName, managerName *string

	err := database.DB.QueryRow(query, id).Scan(
		&emp.ID, &emp.EmployeeID, &emp.Name, &emp.Email, &emp.Position,
		&deptID, &deptName, &roleID, &roleName,
		&emp.TotalLeaveDays, &emp.RemainingLeaveDays,
		&emp.IsManager, &emp.IsActive, &managerID, &managerName, &emp.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	if deptName != nil {
		emp.DepartmentName = *deptName
	}
	if roleName != nil {
		emp.RoleName = *roleName
	}
	if managerName != nil {
		emp.ManagerName = *managerName
	}
	if deptID != nil {
		emp.DepartmentID = deptID
	}
	if roleID != nil {
		emp.RoleID = roleID
	}

	c.JSON(http.StatusOK, emp)
}

// GetManagers - Get managers list (FROM employee.go)
func GetManagers(c *gin.Context) {
	query := `
		SELECT id, name, email, position 
		FROM employees 
		WHERE is_manager = TRUE AND is_active = TRUE
		ORDER BY name`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var managers []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Position string `json:"position"`
	}

	for rows.Next() {
		var mgr struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Email    string `json:"email"`
			Position string `json:"position"`
		}
		err := rows.Scan(&mgr.ID, &mgr.Name, &mgr.Email, &mgr.Position)
		if err != nil {
			continue
		}
		managers = append(managers, mgr)
	}

	c.JSON(http.StatusOK, managers)
}

// CreateEmployee - Create new employee
func CreateEmployee(c *gin.Context) {
	var req models.CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ ERROR binding employee data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("🆕 CREATE EMPLOYEE: %s (%s), DeptID: %d, RoleID: %d",
		req.Name, req.EmployeeID, req.DepartmentID, req.RoleID)

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Check if employee exists
	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM employees WHERE employee_id = ? OR email = ?",
		req.EmployeeID, req.Email).Scan(&exists)
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Employee ID or Email already exists"})
		return
	}

	// Get role untuk set is_manager
	var roleName string
	database.DB.QueryRow("SELECT name FROM roles WHERE id = ?", req.RoleID).Scan(&roleName)
	isManager := roleName == "manager" || roleName == "admin" || roleName == "super_admin"

	// Default values
	if req.TotalLeaveDays == 0 {
		req.TotalLeaveDays = 12
	}

	// QUERY DENGAN STRUCTUR DATABASE YANG BARU
	query := `
		INSERT INTO employees (
			employee_id, name, email, password, position, 
			department_id, role_id, is_manager, manager_id, 
			total_leave_days, remaining_leave_days, is_active, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE, NOW())`

	result, err := database.DB.Exec(query,
		req.EmployeeID, req.Name, req.Email, string(hashedPassword), req.Position,
		req.DepartmentID, req.RoleID, isManager, req.ManagerID,
		req.TotalLeaveDays, req.TotalLeaveDays,
	)

	if err != nil {
		log.Printf("❌ DATABASE ERROR: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	log.Printf("✅ EMPLOYEE CREATED: ID=%d", id)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Employee created successfully",
		"id":      id,
	})
}

// UpdateEmployee - Update employee (FROM users.go)
func UpdateEmployee(c *gin.Context) {
	employeeID := c.Param("id")
	var req models.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE employees SET "
	var args []interface{}

	if req.Name != "" {
		query += "name = ?, "
		args = append(args, req.Name)
	}
	if req.Email != "" {
		query += "email = ?, "
		args = append(args, req.Email)
	}
	if req.Position != "" {
		query += "position = ?, "
		args = append(args, req.Position)
	}
	if req.DepartmentID != 0 {
		query += "department_id = ?, "
		args = append(args, req.DepartmentID)
	}
	if req.RoleID != 0 {
		// Update is_manager based on role
		var roleName string
		database.DB.QueryRow("SELECT name FROM roles WHERE id = ?", req.RoleID).Scan(&roleName)
		isManager := roleName == "manager" || roleName == "admin" || roleName == "super_admin"

		query += "role_id = ?, is_manager = ?, "
		args = append(args, req.RoleID, isManager)
	}
	if req.ManagerID != nil {
		query += "manager_id = ?, "
		args = append(args, *req.ManagerID)
	}
	if req.IsActive != nil {
		query += "is_active = ?, "
		args = append(args, *req.IsActive)
	}
	if req.TotalLeaveDays != 0 {
		query += "total_leave_days = ?, remaining_leave_days = ?, "
		args = append(args, req.TotalLeaveDays, req.TotalLeaveDays)
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2] + " WHERE id = ?"
	args = append(args, employeeID)

	_, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee updated successfully"})
}

// DeleteEmployee - Delete employee (soft delete) (FROM users.go)
func DeleteEmployee(c *gin.Context) {
	employeeID := c.Param("id")

	_, err := database.DB.Exec("UPDATE employees SET is_active = FALSE WHERE id = ?", employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee deactivated successfully"})
}

// GetRoles - Get all roles (FROM users.go)
func GetRoles(c *gin.Context) {
	query := "SELECT id, name, description, permissions, created_at FROM roles ORDER BY name"
	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		var permissionsJSON string

		err := rows.Scan(&role.ID, &role.Name, &role.Description, &permissionsJSON, &role.CreatedAt)
		if err != nil {
			continue
		}

		role.Permissions = map[string]interface{}{"raw": permissionsJSON}
		roles = append(roles, role)
	}

	c.JSON(http.StatusOK, roles)
}

// GetDepartments - Get all departments (FROM users.go)
func GetDepartments(c *gin.Context) {
	query := "SELECT id, name, description, created_at FROM departments ORDER BY name"
	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var departments []models.Department
	for rows.Next() {
		var dept models.Department
		err := rows.Scan(&dept.ID, &dept.Name, &dept.Description, &dept.CreatedAt)
		if err != nil {
			continue
		}
		departments = append(departments, dept)
	}

	c.JSON(http.StatusOK, departments)
}
