package handlers

import (
	"net/http"
	"time"

	"leavemaster/database"
	"leavemaster/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	var loginReq models.LoginRequest
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Updated query dengan join ke departments dan roles
	query := `
		SELECT 
			e.id, e.employee_id, e.name, e.email, e.password, e.position,
			e.department_id, d.name as department_name,
			e.role_id, r.name as role_name,
			e.total_leave_days, e.remaining_leave_days,
			e.is_manager, e.is_active, e.manager_id,
			m.name as manager_name
		FROM employees e
		LEFT JOIN departments d ON e.department_id = d.id
		LEFT JOIN roles r ON e.role_id = r.id
		LEFT JOIN employees m ON e.manager_id = m.id
		WHERE e.email = ? AND e.is_active = TRUE`

	var employee models.Employee
	var deptID, roleID, managerID *int
	var deptName, roleName, managerName *string

	err := database.DB.QueryRow(query, loginReq.Email).Scan(
		&employee.ID, &employee.EmployeeID, &employee.Name, &employee.Email,
		&employee.Password, &employee.Position,
		&deptID, &deptName, &roleID, &roleName,
		&employee.TotalLeaveDays, &employee.RemainingLeaveDays,
		&employee.IsManager, &employee.IsActive, &managerID, &managerName,
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials or account inactive"})
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(employee.Password), []byte(loginReq.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Handle nullable fields
	if deptName != nil {
		employee.DepartmentName = *deptName
	}
	if roleName != nil {
		employee.RoleName = *roleName
	}
	if managerName != nil {
		employee.ManagerName = *managerName
	}
	if deptID != nil {
		employee.DepartmentID = deptID
	}
	if roleID != nil {
		employee.RoleID = roleID
	}

	// Generate JWT token dengan claims tambahan
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["employee_id"] = employee.ID
	claims["is_manager"] = employee.IsManager
	claims["role_id"] = employee.RoleID
	claims["role_name"] = employee.RoleName
	if employee.DepartmentID != nil {
		claims["department_id"] = *employee.DepartmentID
	} else {
		claims["department_id"] = nil
	}
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	employee.Password = "" // Remove password dari response
	response := models.AuthResponse{
		Token:    tokenString,
		Employee: &employee,
	}

	c.JSON(http.StatusOK, response)
}

// ChangePassword - Handler untuk ganti password
func ChangePassword(c *gin.Context) {
	employeeID := c.GetInt("employeeID")

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current password hash
	var currentPasswordHash string
	err := database.DB.QueryRow("SELECT password FROM employees WHERE id = ?", employeeID).
		Scan(&currentPasswordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(req.CurrentPassword))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// Update password
	_, err = database.DB.Exec("UPDATE employees SET password = ? WHERE id = ?",
		string(newPasswordHash), employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// GetProfile - Handler untuk get profile user yang login
func GetProfile(c *gin.Context) {
	employeeID := c.GetInt("employeeID")

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

	var employee models.Employee
	var deptID, roleID, managerID *int
	var deptName, roleName, managerName *string

	err := database.DB.QueryRow(query, employeeID).Scan(
		&employee.ID, &employee.EmployeeID, &employee.Name, &employee.Email, &employee.Position,
		&deptID, &deptName, &roleID, &roleName,
		&employee.TotalLeaveDays, &employee.RemainingLeaveDays,
		&employee.IsManager, &employee.IsActive, &managerID, &managerName, &employee.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Handle nullable fields
	if deptName != nil {
		employee.DepartmentName = *deptName
	}
	if roleName != nil {
		employee.RoleName = *roleName
	}
	if managerName != nil {
		employee.ManagerName = *managerName
	}
	if deptID != nil {
		employee.DepartmentID = deptID
	}
	if roleID != nil {
		employee.RoleID = roleID
	}

	employee.Password = "" // Remove password
	c.JSON(http.StatusOK, employee)
}

// UpdateProfile - Handler untuk update profile
func UpdateProfile(c *gin.Context) {
	employeeID := c.GetInt("employeeID")

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Position string `json:"position"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query
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

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2] + " WHERE id = ?"
	args = append(args, employeeID)

	_, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}
