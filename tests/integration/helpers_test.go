package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gatra-budget-backend/config"
	"gatra-budget-backend/models"
	"gatra-budget-backend/seeds"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultPasswordHash is the bcrypt hash of "Integration123!", precomputed so
// the tests do not pay for hashing on every row insert.
const defaultPasswordHash = "$2a$10$oo/pENGgTSSBkZvAaUkrmO9axNKrq/ffsmVmWqeICD6mJfYDKNljG"

// withTx runs fn inside a transaction that is always rolled back, and points
// the global config.DB at that transaction so the real handlers join it too.
//
// Every test in this package must go through this helper. It means each test
// sees the real schema and real constraints, but leaves nothing behind and
// cannot interfere with the next one.
//
// config.DB and the working directory are process-global, so these tests must
// not call t.Parallel().
func withTx(t *testing.T) *gorm.DB {
	t.Helper()

	tx := testDB.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}

	previous := config.DB
	config.DB = tx

	t.Cleanup(func() {
		config.DB = previous
		if err := tx.Rollback().Error; err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
			t.Errorf("failed to roll back transaction: %v", err)
		}
	})

	return tx
}

// newTestUser creates a user inside the test transaction and returns it.
func newTestUser(t *testing.T, tx *gorm.DB, email, role string) models.User {
	t.Helper()

	user := models.User{
		Email:        email,
		PasswordHash: defaultPasswordHash,
		FullName:     "Test User",
		Role:         role,
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

// app builds a bare Fiber app. Each test registers only the routes it needs.
func app() *fiber.App {
	return fiber.New()
}

// doJSON performs a request with a JSON body and returns the response.
func doJSON(t *testing.T, a *fiber.App, method, path, body string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Test(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, path, err)
	}
	return resp
}

// readBody decodes a JSON response into a map.
func readBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return out
}

// readRawBody returns the response body as a string, for failure messages.
func readRawBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("<unreadable body: %v>", err)
	}
	return string(raw)
}

// chdirTemp moves the process into a temporary directory for the duration of
// the test. The upload handler writes to a path relative to the working
// directory, so this keeps test artifacts out of the repository.
func chdirTemp(t *testing.T) {
	t.Helper()

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to read working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("failed to enter temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})
}

// pngBytes is a minimal 1x1 PNG. The handler validates the extension and size
// only, but a real PNG keeps the fixture honest.
var pngBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

// multipartPNG builds a multipart/form-data body with a single "file" part.
func multipartPNG(t *testing.T) (io.Reader, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(pngBytes); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return &buf, writer.FormDataContentType()
}

// newMultipartRequest builds a request carrying a multipart body.
func newMultipartRequest(t *testing.T, method, path string, body io.Reader, contentType string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", contentType)
	return req
}

// seedAddressReferences inserts the location hierarchy using the real seeder,
// inside the test transaction so it is rolled back afterwards.
func seedAddressReferences(t *testing.T, tx *gorm.DB) {
	t.Helper()

	if err := seeds.AddressReferenceSeeder(tx); err != nil {
		t.Fatalf("failed to seed address references: %v", err)
	}
}

// newTestProject creates a project and a cost type for tests that need rows
// to hang transactions or invoices on.
func newTestProject(t *testing.T, tx *gorm.DB) (models.Project, models.CostType) {
	t.Helper()

	project := models.Project{
		NoSP2K:         "SP2K-" + uuid.NewString()[:8],
		JudulPekerjaan: "Proyek Uji",
		TanggalMulai:   models.Date{Time: mustParseDate(t, "2024-01-01")},
		TanggalSelesai: models.Date{Time: mustParseDate(t, "2024-12-31")},
		NilaiPekerjaan: 1_000_000_000,
	}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	costType := models.CostType{NamaBiaya: "Gaji " + uuid.NewString()[:8]}
	if err := tx.Create(&costType).Error; err != nil {
		t.Fatalf("failed to create cost type: %v", err)
	}
	return project, costType
}
