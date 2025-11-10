# 🏝️ LeaveMaster Backend

**LeaveMaster** is a web-based employee leave management system.  
This project uses **Golang (Gin Framework)** for the backend API, **MySQL** as the main database, and supports **WebSocket** for real-time notifications.

---

## 🚀 Features

- 🔐 JWT Authentication  
- 🧑‍💼 Role-Based & Permission-Based Access Control  
- 📆 Leave & Team Calendar Management  
- 📊 Dashboard & Reports  
- 🔔 Real-time Notifications via WebSocket  
- ✉️ Automated Email Sending via Resend API  
- 🏗️ Ready for Deployment on Railway / Docker  

---

## 🧰 Tech Stack

- **Language:** Go 1.22+  
- **Framework:** Gin  
- **Database:** MySQL  
- **WebSocket:** Gorilla/WebSocket  
- **Email Service:** Resend  
- **Deployment:** Railway (Docker-based)  

---

## 📁 Project Structure

backend/
├── database/
│ └── database.go
├── handlers/
│ ├── auth.go
│ ├── calendar.go
│ ├── employee.go
│ ├── leave.go
│ ├── reports.go
│ └── email_test.go
├── middleware/
│ ├── auth.go
│ ├── roles.go
│ └── websocket_auth.go
├── models/
│ └── models.go
├── services/
│ └── email.go
├── websocket/
│ └── hub.go
├── .env
├── go.mod
├── go.sum
└── main.go

yaml
Copy code

---

## ⚙️ Local Setup

### 1. Clone the Repository
```bash
git clone https://github.com/yourusername/leavemaster.git
cd leavemaster/backend
2. Install Dependencies
bash
Copy code
go mod tidy
3. Set Up the Database
Create a new MySQL database:

sql
Copy code
CREATE DATABASE leavemaster;
4. Configure Environment Variables
Create a .env file inside the backend/ folder:

env
Copy code
# Resend Configuration
RESEND_API_KEY=your_resend_api_key
RESEND_FROM=LeaveMaster <onboarding@resend.dev>

# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=leavemaster

# JWT Secret
JWT_SECRET=your-super-secret-jwt-key
5. Run the Application
bash
Copy code
go run main.go
The server will run at:

arduino
Copy code
http://localhost:8080
🌐 Main API Endpoints
Endpoint	Method	Description
/api/login	POST	User login
/api/profile	GET	Fetch user profile
/api/leave	POST	Submit a leave request
/api/reports/dashboard-stats	GET	Dashboard statistics
/health	GET	Check API & WebSocket status

🔌 WebSocket
WebSocket server runs at:

bash
Copy code
ws://localhost:8080/ws
Protected with JWT authentication middleware.

🐳 Deployment on Railway
1. Connect Repository to Railway
Go to https://railway.app

Create a new project → Deploy from GitHub

Select your repository (leavemaster)

2. Add Environment Variables
In the Variables tab, add:

Key	Value
RESEND_API_KEY	re_...
RESEND_FROM	LeaveMaster <onboarding@resend.dev>
DB_HOST	containers-us-west-21.railway.app
DB_PORT	3306
DB_USER	root
DB_PASSWORD	yourpassword
DB_NAME	leavemaster
JWT_SECRET	your-super-secret-jwt-key
APP_ENV	production
PORT	8080

3. (Optional) Add MySQL Plugin
Click “Add Plugin” → “MySQL”

Railway will automatically generate database credentials and fill them into your environment variables.

4. Deploy
Railway will automatically build and deploy your Go app using the provided Dockerfile.

You can verify deployment by visiting:

arduino
Copy code
https://<your-app-name>.up.railway.app/health
📄 License
This project is licensed under the MIT License.
Feel free to modify, distribute, and use it for your organization.

💡 Contributors
Yozabel Hamuda — Lead Developer

💬 Contact: GitHub