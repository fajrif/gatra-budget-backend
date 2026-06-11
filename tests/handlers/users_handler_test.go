package handlers_test

import (
	// fmt "fmt"
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"boilerplates-backend/handlers"
	"boilerplates-backend/models"
	"boilerplates-backend/tests/testutil"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Helper function to create an app with admin middleware
func setupAdminApp() *fiber.App {
	app := fiber.New()
	// Add middleware to set admin userID and role in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "550e8400-e29b-41d4-a716-446655440000")
		c.Locals("userEmail", "admin@example.com")
		c.Locals("userRole", "admin")
		return c.Next()
	})
	return app
}

func TestCreateUser(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupAdminApp()
	app.Post("/users", handlers.CreateUser)

	t.Run("Successfully create user", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":     "newuser@example.com",
			"password":  "Password123!",
			"full_name": "New User",
			"role":      "user",
		}

		// Mock: Check if user exists (should return error/not found)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
			WithArgs("newuser@example.com").
			WillReturnError(gorm.ErrRecordNotFound)

		// Mock: Create user
		userID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "full_name", "role", "created_at", "updated_at"}).
				AddRow(userID, "newuser@example.com", "New User", "user", time.Now(), time.Now()))
		mock.ExpectCommit()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "User created successfully", response["message"])
		assert.NotNil(t, response["user"])
	})

	t.Run("Create user with admin role", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":     "admin2@example.com",
			"password":  "Password123!",
			"full_name": "Admin User",
			"role":      "admin",
		}

		// Mock: Check if user exists
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
			WithArgs("admin2@example.com").
			WillReturnError(gorm.ErrRecordNotFound)

		// Mock: Create user
		userID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "full_name", "role", "created_at", "updated_at"}).
				AddRow(userID, "admin2@example.com", "Admin User", "admin", time.Now(), time.Now()))
		mock.ExpectCommit()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "User created successfully", response["message"])
		userMap := response["user"].(map[string]interface{})
		assert.Equal(t, "admin", userMap["role"])
	})

	t.Run("Invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Invalid request body", response["error"])
	})

	t.Run("Missing required fields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email": "newuser@example.com",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Password is required", response["error"])
	})

	t.Run("Email already exists", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":     "existing@example.com",
			"password":  "Password123!",
			"full_name": "Existing User",
		}

		// Mock: User already exists
		existingUser := models.User{
			ID:    uuid.New(),
			Email: "existing@example.com",
		}

		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(existingUser.ID, existingUser.Email, "hashedpass", "Existing User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
			WithArgs("existing@example.com").
			WillReturnRows(rows)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Email already exists", response["error"])
	})

	t.Run("Invalid role", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":     "newuser@example.com",
			"password":  "Password123!",
			"full_name": "New User",
			"role":      "superadmin",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Role must be either 'user' or 'admin' or 'operator'", response["error"])
	})
}

func TestGetAllUsers(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupAdminApp()
	app.Get("/users", handlers.GetAllUsers)

	t.Run("Successfully get all users", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()

		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID1, "user1@example.com", "hash1", "User One", "user", time.Now(), time.Now()).
			AddRow(userID2, "user2@example.com", "hash2", "User Two", "admin", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY created_at DESC LIMIT 20`)).
			WillReturnRows(userRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
			WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/users", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.NotNil(t, response["users"])
		assert.NotNil(t, response["pagination"])

		users := response["users"].([]interface{})
		assert.Equal(t, 2, len(users))
	})

	t.Run("Get users with pagination", func(t *testing.T) {
		db2, mock2, err := testutil.SetupMockDB()
		assert.NoError(t, err)
		defer testutil.CloseMockDB(db2)

		userID := uuid.New()

		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user1@example.com", "hash1", "User One", "user", time.Now(), time.Now())

		mock2.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY created_at DESC LIMIT 10 OFFSET 10`)).
			WillReturnRows(userRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(15)
		mock2.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
			WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/users?page=2&limit=10", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.NotNil(t, response["users"])
		pagination := response["pagination"].(map[string]interface{})
		assert.Equal(t, float64(2), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
	})

	t.Run("Search users by email", func(t *testing.T) {
		db3, mock3, err := testutil.SetupMockDB()
		assert.NoError(t, err)
		defer testutil.CloseMockDB(db3)

		userID := uuid.New()

		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "john@example.com", "hash1", "John Doe", "user", time.Now(), time.Now())

		mock3.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email ILIKE $1 OR full_name ILIKE $2 ORDER BY created_at DESC LIMIT 20`)).
			WithArgs("%john%", "%john%").
			WillReturnRows(userRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock3.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE email ILIKE $1 OR full_name ILIKE $2`)).
			WithArgs("%john%", "%john%").
			WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/users?search=john", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.NotNil(t, response["users"])
		users := response["users"].([]interface{})
		assert.Equal(t, 1, len(users))
	})

	t.Run("Empty list", func(t *testing.T) {
		db4, mock4, err := testutil.SetupMockDB()
		assert.NoError(t, err)
		defer testutil.CloseMockDB(db4)

		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"})

		mock4.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY created_at DESC LIMIT 20`)).
			WillReturnRows(userRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock4.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
			WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/users", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.NotNil(t, response["users"])
		assert.NotNil(t, response["pagination"])
	})
}

func TestGetUser(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupAdminApp()
	app.Get("/users/:id", handlers.GetUser)

	t.Run("Successfully get user by ID", func(t *testing.T) {
		userID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "user@example.com", response["email"])
		assert.Equal(t, "Test User", response["full_name"])
		assert.Equal(t, "user", response["role"])
	})

	t.Run("Invalid user ID format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/invalid-uuid", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Invalid user ID", response["error"])
	})

	t.Run("User not found", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "User not found", response["error"])
	})
}

func TestUpdateUser(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupAdminApp()
	app.Put("/users/:id", handlers.UpdateUser)

	t.Run("Successfully update user", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "oldhashedpass", "Old Name", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		// Mock: Update user
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		updateData := map[string]interface{}{
			"full_name": "Updated Name",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/" + userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		// _body, _ := io.ReadAll(resp.Body)
		// fmt.Println("Response:", string(_body))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "user@example.com", response["email"])
	})

	t.Run("Successfully update user password", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "oldhashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		// Mock: Update user with new password
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		updateData := map[string]interface{}{
			"password": "NewPassword123!",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "user@example.com", response["email"])
	})

	t.Run("Update user role to admin", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		// Mock: Update user
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		updateData := map[string]interface{}{
			"role": "admin",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Update user email", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "old@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		// Mock: Check if new email already exists (should return not found)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
			WithArgs("new@example.com").
			WillReturnError(gorm.ErrRecordNotFound)

		// Mock: Update user
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		updateData := map[string]interface{}{
			"email": "new@example.com",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Invalid user ID format", func(t *testing.T) {
		updateData := map[string]interface{}{
			"full_name": "Updated Name",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/invalid-uuid", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Invalid user ID", response["error"])
	})

	t.Run("User not found", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		updateData := map[string]interface{}{
			"full_name": "Updated Name",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "User not found", response["error"])
	})

	t.Run("Email already exists", func(t *testing.T) {
		userID := uuid.New()
		existingUserID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		// Mock: Check if new email already exists (should return existing user)
		existingRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(existingUserID, "existing@example.com", "hashedpass", "Existing User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1`)).
			WithArgs("existing@example.com").
			WillReturnRows(existingRows)

		updateData := map[string]interface{}{
			"email": "existing@example.com",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Email already exists", response["error"])
	})

	t.Run("Invalid role", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		updateData := map[string]interface{}{
			"role": "superadmin",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Role must be either 'user' or 'admin' or 'operator'", response["error"])
	})

	t.Run("Invalid request body", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("PUT", "/users/"+userID.String(), bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Invalid request body", response["error"])
	})
}

func TestDeleteUser(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupAdminApp()
	app.Delete("/users/:id", handlers.DeleteUser)

	t.Run("Successfully delete user", func(t *testing.T) {
		userID := uuid.New()

		// Mock: Find user
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "created_at", "updated_at"}).
			AddRow(userID, "user@example.com", "hashedpass", "Test User", "user", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		// Mock: Delete user
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/users/"+userID.String(), nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "User deleted successfully", response["message"])
	})

	t.Run("Invalid user ID format", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/invalid-uuid", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Invalid user ID", response["error"])
	})

	t.Run("User not found", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).
			WithArgs(userID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("DELETE", "/users/"+userID.String(), nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "User not found", response["error"])
	})
}

func TestPasswordHashing(t *testing.T) {
	t.Run("Password is properly hashed", func(t *testing.T) {
		password := "Testpassword123!"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		assert.NoError(t, err)
		assert.NotEqual(t, password, string(hashedPassword))

		// Verify that the hashed password can be compared correctly
		err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
		assert.NoError(t, err)

		// Verify that wrong password fails
		err = bcrypt.CompareHashAndPassword(hashedPassword, []byte("wrongpassword"))
		assert.Error(t, err)
	})
}
