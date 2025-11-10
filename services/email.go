package services

import (
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client *resend.Client
	from   string
}

func NewEmailService() *EmailService {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		apiKey = "re_Efit59PS_2QG4zaSWeseqg5bTuyna4qi8" // Fallback ke API key Anda
	}

	return &EmailService{
		client: resend.NewClient(apiKey),
		from:   getEnv("RESEND_FROM", "LeaveMaster <notifications@yourdomain.com>"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (es *EmailService) SendLeaveRequestNotification(managerEmail, managerName, employeeName, leaveType, startDate, endDate, reason string) error {
	subject := "🎯 New Leave Request Requires Your Approval"

	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
			.header { background: #3498db; color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
			.content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
			.details { background: white; padding: 15px; border-radius: 5px; margin: 15px 0; }
			.button { background: #3498db; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block; }
			.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>📍 LeaveMaster</h1>
				<p>Leave Request Notification</p>
			</div>
			<div class="content">
				<h2>Hello %s,</h2>
				<p>You have a new leave request waiting for your approval:</p>
				
				<div class="details">
					<h3>Request Details</h3>
					<table style="width: 100%%; border-collapse: collapse;">
						<tr><td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Employee</strong></td><td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td></tr>
						<tr><td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Leave Type</strong></td><td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td></tr>
						<tr><td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Start Date</strong></td><td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td></tr>
						<tr><td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>End Date</strong></td><td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td></tr>
						<tr><td style="padding: 8px;"><strong>Reason</strong></td><td style="padding: 8px;">%s</td></tr>
					</table>
				</div>

				<p style="text-align: center;">
					<a href="http://localhost:3000" class="button">Review Request in LeaveMaster</a>
				</p>

				<p><small>This is an automated notification. Please do not reply to this email.</small></p>
			</div>
			<div class="footer">
				<p>&copy; 2024 LeaveMaster. All rights reserved.</p>
			</div>
		</div>
	</body>
	</html>
	`, managerName, employeeName, leaveType, startDate, endDate, reason)

	// Text version untuk email clients yang sederhana
	textBody := fmt.Sprintf(`
	New Leave Request Notification
	
	Hello %s,
	
	You have a new leave request waiting for your approval:
	
	Employee: %s
	Leave Type: %s
	Start Date: %s
	End Date: %s
	Reason: %s
	
	Please log in to LeaveMaster to review this request:
	http://localhost:3000
	
	This is an automated notification. Please do not reply to this email.
	`, managerName, employeeName, leaveType, startDate, endDate, reason)

	params := &resend.SendEmailRequest{
		From:    es.from,
		To:      []string{managerEmail},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	_, err := es.client.Emails.Send(params)
	if err != nil {
		// Fallback ke console log jika Resend gagal
		fmt.Printf("=== RESEND EMAIL FAILED - FALLBACK TO CONSOLE ===\n")
		fmt.Printf("To: %s\n", managerEmail)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Body: %s\n", textBody)
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("===============================================\n")
		return err
	}

	fmt.Printf("✅ Email sent via Resend to: %s\n", managerEmail)
	return nil
}

func (es *EmailService) SendLeaveStatusNotification(employeeEmail, employeeName, status, leaveType, startDate, endDate string) error {

	statusEmoji := "✅"
	statusText := "Approved"
	if status == "rejected" {
		statusEmoji = "❌"
		statusText = "Rejected"
	}

	subject := fmt.Sprintf("%s Leave Request %s", statusEmoji, statusText)

	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
			.header { background: %s; color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0; }
			.content { background: #f9f9f9; padding: 20px; border-radius: 0 0 10px 10px; }
			.status { font-size: 24px; font-weight: bold; text-align: center; margin: 20px 0; }
			.details { background: white; padding: 15px; border-radius: 5px; margin: 15px 0; }
			.button { background: %s; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block; }
			.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header" style="background: %s;">
				<h1>📍 LeaveMaster</h1>
				<p>Leave Request Update</p>
			</div>
			<div class="content">
				<h2>Hello %s,</h2>
				
				<div class="status" style="color: %s;">
					%s Your %s leave request has been <strong>%s</strong>
				</div>

				<div class="details">
					<h3>Request Details</h3>
					<table style="width: 100%%; border-collapse: collapse;">
						<tr><td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Leave Type</strong></td><td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td></tr>
						<tr><td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Dates</strong></td><td style="padding: 8px; border-bottom: 1px solid #eee;">%s to %s</td></tr>
						<tr><td style="padding: 8px;"><strong>Status</strong></td><td style="padding: 8px;"><strong>%s</strong></td></tr>
					</table>
				</div>

				<p style="text-align: center;">
					<a href="http://localhost:3000" class="button" style="background: %s;">View Details in LeaveMaster</a>
				</p>

				<p><small>This is an automated notification. Please do not reply to this email.</small></p>
			</div>
			<div class="footer">
				<p>&copy; 2024 LeaveMaster. All rights reserved.</p>
			</div>
		</div>
	</body>
	</html>
	`, getStatusColor(status), getStatusColor(status), getStatusColor(status),
		employeeName, getStatusColor(status), statusEmoji, leaveType, statusText,
		leaveType, startDate, endDate, statusText, getStatusColor(status))

	textBody := fmt.Sprintf(`
	Leave Request Update
	
	Hello %s,
	
	Your %s leave request has been %s.
	
	Leave Type: %s
	Dates: %s to %s
	Status: %s
	
	You can check the details in your LeaveMaster dashboard:
	http://localhost:3000
	
	This is an automated notification. Please do not reply to this email.
	`, employeeName, leaveType, statusText, leaveType, startDate, endDate, statusText)

	params := &resend.SendEmailRequest{
		From:    es.from,
		To:      []string{employeeEmail},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	_, err := es.client.Emails.Send(params)
	if err != nil {
		// Fallback ke console log
		fmt.Printf("=== RESEND EMAIL FAILED - FALLBACK TO CONSOLE ===\n")
		fmt.Printf("To: %s\n", employeeEmail)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Body: %s\n", textBody)
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("===============================================\n")
		return err
	}

	fmt.Printf("✅ Status email sent via Resend to: %s\n", employeeEmail)
	return nil
}

func getStatusColor(status string) string {
	if status == "approved" {
		return "#2ecc71" // Green
	} else if status == "rejected" {
		return "#e74c3c" // Red
	}
	return "#3498db" // Blue
}
