package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// TestLoginAndFetchMe exercises the whole authenticated path against a real
// database: a real bcrypt comparison, a real signed JWT, and the real auth
// middleware populating the request context from it.
func TestLoginAndFetchMe(t *testing.T) {
	tx := withTx(t)
	user := newTestUser(t, tx, "login@gatrahitawasana.com", "admin")

	a := app()
	a.Post("/api/auth/login", handlers.Login)

	api := a.Group("/api", middleware.AuthRequired())
	api.Get("/me", handlers.GetMe)

	t.Run("valid credentials return a usable token", func(t *testing.T) {
		resp := doJSON(t, a, "POST", "/api/auth/login", `{
			"email": "login@gatrahitawasana.com",
			"password": "Integration123!"
		}`)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from login, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		payload := readBody(t, resp)
		token, _ := payload["token"].(string)
		if token == "" {
			t.Fatalf("expected a token in the login response, got %v", payload)
		}

		// Use the real token against the real middleware.
		req := httptest.NewRequest("GET", "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		meResp, err := a.Test(req)
		if err != nil {
			t.Fatalf("GET /api/me failed: %v", err)
		}
		if meResp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 from /api/me, got %d: %s", meResp.StatusCode, readRawBody(t, meResp))
		}

		me := readBody(t, meResp)
		if me["email"] != user.Email {
			t.Fatalf("expected /api/me to return %q, got %v", user.Email, me["email"])
		}
		if me["role"] != "admin" {
			t.Fatalf("expected role %q, got %v", "admin", me["role"])
		}
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		resp := doJSON(t, a, "POST", "/api/auth/login", `{
			"email": "login@gatrahitawasana.com",
			"password": "not-the-password"
		}`)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected 401 for a wrong password, got %d", resp.StatusCode)
		}
	})

	t.Run("missing token is rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/me", nil)
		resp, err := a.Test(req)
		if err != nil {
			t.Fatalf("GET /api/me failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected 401 without a token, got %d", resp.StatusCode)
		}
	})
}
