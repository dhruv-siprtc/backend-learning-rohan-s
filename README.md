A simple REST API built with Go (Echo), GORM, and PostgreSQL.
Supports soft deletes, password hashing, and active-user email uniqueness.

🛠️ Features

Create, read, update, delete users

Soft delete support (deleted_at column)

Passwords stored securely using bcrypt

Prevents duplicate emails for active users

Separate dev and test databases

Fully environment-driven configuration

 Requirements

Go ≥ 1.20

PostgreSQL

psql CLI or GUI

 Postman  for API testing

⚙️ Setup
1. Clone repository
git clone <your-repo-url>
cd <repo-folder>

2. Create databases
CREATE DATABASE postgis_36_sample;      -- Development
CREATE DATABASE postgis_36_sample_test; -- Tests

3. Environment files
.env (development)
APP_ENV=dev
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=123
DB_NAME=postgis_36_sample
DB_PORT=5432

.env.test (tests)
APP_ENV=test
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=123
DB_NAME=postgis_36_sample_test
DB_PORT=5432





Server runs at: http://localhost:8080

You should see:

✅ Database connected
🚀 Server running at http://localhost:8080

🔍 API Endpoints
Method	Route	Description
GET	/users	List all active users
GET	/users/:id	Get user by ID
POST	/users	Create new user
PUT	/users/:id	Update existing user (incl. password)
DELETE	/users/:id	Soft delete user
🧪 Testing
Run tests
go test ./...


✅ Uses .env.test database (postgis_36_sample_test)
✅ Cleans DB before tests
✅ No dev data affected

Example Test Commands

Create user:

POST http://localhost:8080/users
Content-Type: application/json

{
  "name": "Naruto",
  "email": "naruto@test.com",
  "password": "123456"
}


Update user:

PUT http://localhost:8080/users/1
Content-Type: application/json

{
  "name": "Naruto Uzumaki",
  "email": "naruto@test.com",
  "password": "newpassword123"
}


Delete user:

DELETE http://localhost:8080/users/1


Get all users:

GET http://localhost:8080/users

🧾 Notes

Soft-deleted users are excluded from GET requests

Email uniqueness is enforced only among active users

Passwords are always hashed before saving

Updates are performed safely with explicit Updates, no accidental inserts

