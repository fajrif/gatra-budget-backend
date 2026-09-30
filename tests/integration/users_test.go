package integration

import (
	"net/http"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TestUserPhotoURLRoundTrip is the regression test for the missing
// users.photo_url column. models.User has always had the field and the avatar
// handler has always written to it, but no migration created the column, so
// every upload failed with a database error. sqlmock happily accepted the
// UPDATE, so only a real database catches this.
func TestUserPhotoURLRoundTrip(t *testing.T) {
	tx := withTx(t)
	user := newTestUser(t, tx, "photo@gatrahitawasana.com", "user")

	a := app()
	a.Post("/upload/:resource/:field/:id", handlers.UploadResourceField)

	// The handler writes relative to the working directory; move it to a temp
	// dir so the test does not litter the repository with uploads.
	chdirTemp(t)

	body, contentType := multipartPNG(t)
	req := newMultipartRequest(t, "POST", "/upload/users/photo/"+user.ID.String(), body, contentType)
	resp, err := a.Test(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 from upload, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	payload := readBody(t, resp)
	fileURL, _ := payload["file_url"].(string)
	if fileURL == "" {
		t.Fatalf("expected a file_url in the response, got %v", payload)
	}

	// Re-read through GORM: this is what actually failed before the migration.
	var reloaded models.User
	if err := tx.Where("id = ?", user.ID).First(&reloaded).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.PhotoUrl == nil {
		t.Fatal("expected photo_url to be persisted, got NULL")
	}
	if *reloaded.PhotoUrl != fileURL {
		t.Fatalf("photo_url mismatch: stored %q, response %q", *reloaded.PhotoUrl, fileURL)
	}
}

// TestDuplicateEmailRejected proves the real UNIQUE index on users.email is
// enforced. The handler's duplicate check is faked in the unit tests, so this
// is the only place the database constraint itself is verified.
func TestDuplicateEmailRejected(t *testing.T) {
	tx := withTx(t)
	newTestUser(t, tx, "dupe@gatrahitawasana.com", "user")

	a := app()
	a.Post("/users", handlers.CreateUser)

	resp := doJSON(t, a, "POST", "/users", `{
		"email": "dupe@gatrahitawasana.com",
		"password": "Secret123!",
		"full_name": "Duplicate",
		"role": "user"
	}`)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for a duplicate email, got %d: %s",
			resp.StatusCode, readRawBody(t, resp))
	}
}

// TestDeleteUserCascadesAddresses verifies the ON DELETE CASCADE from
// addresses.user_id, which no unit test can observe.
func TestDeleteUserCascadesAddresses(t *testing.T) {
	tx := withTx(t)
	user := newTestUser(t, tx, "cascade@gatrahitawasana.com", "user")

	seedAddressReferences(t, tx)

	address := models.Address{
		UserID:     user.ID,
		Name:       "Home",
		Address:    "Jl. Test No. 1",
		CountryID:  uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		ProvinceID: uuid.MustParse("20000000-0000-0000-0000-000000000001"),
		CityID:     uuid.MustParse("30000000-0000-0000-0000-000000000001"),
	}
	if err := tx.Create(&address).Error; err != nil {
		t.Fatalf("failed to create address: %v", err)
	}

	var before int64
	if err := tx.Model(&models.Address{}).Where("user_id = ?", user.ID).Count(&before).Error; err != nil {
		t.Fatalf("failed to count addresses: %v", err)
	}
	if before != 1 {
		t.Fatalf("expected 1 address before delete, got %d", before)
	}

	if err := tx.Delete(&models.User{}, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	var after int64
	if err := tx.Model(&models.Address{}).Where("user_id = ?", user.ID).Count(&after).Error; err != nil {
		t.Fatalf("failed to count addresses after delete: %v", err)
	}
	if after != 0 {
		t.Fatalf("expected addresses to cascade-delete, %d still present", after)
	}
}
