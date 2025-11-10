package models

import "time"

type Employee struct {
	ID                 int       `json:"id"`
	EmployeeID         string    `json:"employee_id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Password           string    `json:"password,omitempty"`
	Position           string    `json:"position"`
	DepartmentID       *int      `json:"department_id"`
	DepartmentName     string    `json:"department_name,omitempty"`
	RoleID             *int      `json:"role_id"`
	RoleName           string    `json:"role_name,omitempty"`
	TotalLeaveDays     int       `json:"total_leave_days"`
	RemainingLeaveDays int       `json:"remaining_leave_days"`
	IsManager          bool      `json:"is_manager"`
	IsActive           bool      `json:"is_active"`
	ManagerID          *int      `json:"manager_id"`
	ManagerName        string    `json:"manager_name,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type Role struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions map[string]interface{} `json:"permissions"`
	CreatedAt   time.Time              `json:"created_at"`
}

type Department struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type LeaveRequest struct {
	ID           int       `json:"id"`
	EmployeeID   int       `json:"employee_id"`
	EmployeeName string    `json:"employee_name,omitempty"`
	LeaveType    string    `json:"leave_type"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	TotalDays    int       `json:"total_days"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	ApprovedBy   *int      `json:"approved_by"`
	ApprovedAt   *string   `json:"approved_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token    string    `json:"token"`
	Employee *Employee `json:"employee"`
}

type CreateEmployeeRequest struct {
	EmployeeID     string `json:"employee_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Email          string `json:"email" binding:"required"`
	Password       string `json:"password" binding:"required"`
	Position       string `json:"position" binding:"required"`
	DepartmentID   int    `json:"department_id" binding:"required"`
	RoleID         int    `json:"role_id" binding:"required"`
	ManagerID      *int   `json:"manager_id"`
	TotalLeaveDays int    `json:"total_leave_days"`
}

type UpdateEmployeeRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Position       string `json:"position"`
	DepartmentID   int    `json:"department_id"`
	RoleID         int    `json:"role_id"`
	ManagerID      *int   `json:"manager_id"`
	IsActive       *bool  `json:"is_active"`
	TotalLeaveDays int    `json:"total_leave_days"`
}
