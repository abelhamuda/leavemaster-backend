package handlers

import (
	"leavemaster/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TestEmail(c *gin.Context) {
	emailService := services.NewEmailService()

	// Test leave request notification
	err := emailService.SendLeaveRequestNotification(
		"manager@company.com",
		"John Manager",
		"Alice Employee",
		"annual",
		"2024-02-15",
		"2024-02-17",
		"Family vacation",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send test email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test email sent successfully! Check your email inbox.",
	})
}
