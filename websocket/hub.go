package websocket

import (
	"encoding/json"
	"fmt"
	"leavemaster/database"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID        int
	Conn      *websocket.Conn
	Hub       *Hub
	Send      chan []byte
	IsManager bool
}

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	Mutex      sync.RWMutex
}

type Notification struct {
	Type       string      `json:"type"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	ForManager bool        `json:"for_manager"`
	TargetID   int         `json:"target_id,omitempty"` // Untuk kirim ke user spesifik
}

var HubInstance = NewHub()

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			h.Clients[client] = true
			h.Mutex.Unlock()
			log.Printf("✅ Client registered - ID: %d, IsManager: %t, Total Clients: %d",
				client.ID, client.IsManager, len(h.Clients))

		case client := <-h.Unregister:
			h.Mutex.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.Mutex.Unlock()
			log.Printf("❌ Client unregistered - ID: %d, Total Clients: %d", client.ID, len(h.Clients))

		case message := <-h.Broadcast:
			h.Mutex.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.Mutex.RUnlock()
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("❌ WebSocket upgrade error:", err)
		return
	}

	employeeID, _ := c.Get("employeeID")
	isManager, _ := c.Get("isManager")

	log.Printf("✅ New WebSocket connection - EmployeeID: %d, IsManager: %t", employeeID, isManager)
	log.Printf("🌐 Connection from: %s", c.Request.RemoteAddr)

	client := &Client{
		ID:        employeeID.(int),
		Conn:      conn,
		Hub:       HubInstance,
		Send:      make(chan []byte, 256),
		IsManager: isManager.(bool),
	}

	client.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

// PERBAIKAN: Function SendNotification yang lebih spesifik
func SendNotification(notification Notification) {
	message, err := json.Marshal(notification)
	if err != nil {
		log.Println("Error marshaling notification:", err)
		return
	}

	HubInstance.Mutex.RLock()
	defer HubInstance.Mutex.RUnlock()

	clientCount := len(HubInstance.Clients)
	var sentCount int

	log.Printf("Sending notification to %d clients", clientCount)
	log.Printf("Notification Type: %s, ForManager: %t, TargetID: %d",
		notification.Type, notification.ForManager, notification.TargetID)

	for client := range HubInstance.Clients {
		shouldSend := false

		if notification.ForManager && client.IsManager {
			shouldSend = true
			log.Printf("   → Sending to MANAGER ID: %d", client.ID)
		} else if !notification.ForManager && !client.IsManager {
			shouldSend = true
			log.Printf("   → Sending to EMPLOYEE ID: %d", client.ID)
		} else if notification.TargetID > 0 && client.ID == notification.TargetID {
			shouldSend = true
			log.Printf("   → Sending to TARGET ID: %d", client.ID)
		}

		if shouldSend {
			select {
			case client.Send <- message:
				sentCount++
			default:
				log.Printf("   Failed to send to client ID: %d", client.ID)
				close(client.Send)
				delete(HubInstance.Clients, client)
			}
		}
	}

	log.Printf("Notification sent to %d/%d clients", sentCount, clientCount)
}

// PERBAIKAN: Function untuk new leave request - HANYA ke manager
func SendNewLeaveRequestNotification(employeeName, leaveType, startDate, endDate, reason string, departmentID int) {
	notification := Notification{
		Type:       "new_leave_request",
		Message:    fmt.Sprintf("New leave request from %s", employeeName),
		ForManager: true, // HANYA untuk manager
		Data: map[string]interface{}{
			"employee_name": employeeName,
			"leave_type":    leaveType,
			"start_date":    startDate,
			"end_date":      endDate,
			"reason":        reason,
			"department_id": departmentID,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}

	log.Printf("🆕 Sending NEW LEAVE REQUEST notification for employee: %s (Dept: %d)", employeeName, departmentID)
	SendNotificationToDepartmentManagers(notification, departmentID)
}

// New Func to send notification to manager
func SendNotificationToDepartmentManagers(notification Notification, departmentID int) {
	message, err := json.Marshal(notification)
	if err != nil {
		log.Println("Error marshaling notification:", err)
		return
	}

	HubInstance.Mutex.RLock()
	defer HubInstance.Mutex.RUnlock()

	clientCount := len(HubInstance.Clients)
	var sentCount int

	log.Printf("📤 Sending department notification to %d clients for department %d", clientCount, departmentID)

	for client := range HubInstance.Clients {
		shouldSend := false

		// Hanya kirim ke manager yang connected
		if client.IsManager {
			// Get department manager dari database
			var managerDeptID int
			err := database.DB.QueryRow("SELECT department_id FROM employees WHERE id = ?", client.ID).Scan(&managerDeptID)
			if err != nil {
				log.Printf("❌ Error getting department for manager %d: %v", client.ID, err)
				continue
			}

			// Hanya kirim jika manager di department yang sama
			if managerDeptID == departmentID {
				shouldSend = true
				log.Printf("   → Sending to MANAGER ID: %d (Dept: %d)", client.ID, managerDeptID)
			}
		}

		if shouldSend {
			select {
			case client.Send <- message:
				sentCount++
			default:
				log.Printf("   ❌ Failed to send to manager ID: %d", client.ID)
				close(client.Send)
				delete(HubInstance.Clients, client)
			}
		}
	}

	log.Printf("✅ Department notification sent to %d/%d managers in department %d", sentCount, clientCount, departmentID)
}

// PERBAIKAN: Function untuk status update - HANYA ke employee yang bersangkutan
func SendLeaveStatusNotification(employeeName, status, leaveType string, employeeID int) {
	notificationType := "leave_approved"
	message := fmt.Sprintf("Your %s leave request has been approved", leaveType)

	if status == "rejected" {
		notificationType = "leave_rejected"
		message = fmt.Sprintf("Your %s leave request has been rejected", leaveType)
	}

	notification := Notification{
		Type:       notificationType,
		Message:    message,
		ForManager: false,      // Untuk employee
		TargetID:   employeeID, // Target spesifik
		Data: map[string]interface{}{
			"employee_name": employeeName,
			"leave_type":    leaveType,
			"status":        status,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}

	log.Printf("📋 Sending LEAVE STATUS notification for employee: %s (ID: %d)", employeeName, employeeID)
	SendNotification(notification)
}

// Function baru untuk broadcast ke semua manager
func BroadcastToManagers(messageType, message string, data map[string]interface{}) {
	notification := Notification{
		Type:       messageType,
		Message:    message,
		ForManager: true,
		Data:       data,
	}
	SendNotification(notification)
}

// Function untuk mendapatkan info connected clients (debug)
func GetConnectedClientsInfo() map[string]interface{} {
	HubInstance.Mutex.RLock()
	defer HubInstance.Mutex.RUnlock()

	var managers []map[string]interface{}
	var employees []map[string]interface{}

	for client := range HubInstance.Clients {
		clientInfo := map[string]interface{}{
			"id":         client.ID,
			"is_manager": client.IsManager,
			"connected":  true,
		}

		if client.IsManager {
			managers = append(managers, clientInfo)
		} else {
			employees = append(employees, clientInfo)
		}
	}

	return map[string]interface{}{
		"total_clients":  len(HubInstance.Clients),
		"managers":       managers,
		"employees":      employees,
		"manager_count":  len(managers),
		"employee_count": len(employees),
	}
}

// Function to get department ID from employee
func GetEmployeeDepartment(employeeID int) (int, error) {
	var departmentID int
	err := database.DB.QueryRow("SELECT department_id FROM employees WHERE id = ?", employeeID).Scan(&departmentID)
	if err != nil {
		return 0, err
	}
	return departmentID, nil
}
