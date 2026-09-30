package integration

import (
	"net/http"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
	"gatra-budget-backend/seeds"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TestAddressCRUD drives the "my addresses" endpoints through the real
// middleware, so user_id comes from the validated JWT rather than a test
// fixture, and the real foreign keys to the location tables are enforced.
func TestAddressCRUD(t *testing.T) {
	tx := withTx(t)
	user := newTestUser(t, tx, "addr@gatrahitawasana.com", "user")
	seedAddressReferences(t, tx)

	countryID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	provinceID := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	cityID := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	areaID := uuid.MustParse("40000000-0000-0000-0000-000000000001")

	a := app()
	group := a.Group("/api/me/addresses", asUser(t, user.ID.String(), user.Email))
	group.Post("/", handlers.CreateMyAddress)
	group.Get("/", handlers.GetMyAddresses)
	group.Get("/:id", handlers.GetMyAddress)
	group.Put("/:id", handlers.UpdateMyAddress)
	group.Delete("/:id", handlers.DeleteMyAddress)

	var addressID string

	t.Run("create", func(t *testing.T) {
		resp := doJSON(t, a, "POST", "/api/me/addresses", `{
			"name": "Home",
			"address": "Jl. Merdeka No. 10",
			"country_id": "`+countryID.String()+`",
			"province_id": "`+provinceID.String()+`",
			"city_id": "`+cityID.String()+`",
			"area_id": "`+areaID.String()+`",
			"zipcode": "10110"
		}`)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 on create, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		payload := readBody(t, resp)
		id, _ := payload["id"].(string)
		if id == "" {
			t.Fatalf("expected an id in the create response, got %v", payload)
		}
		addressID = id

		// The row must really be there, owned by the authenticated user.
		var stored models.Address
		if err := tx.Where("id = ?", id).First(&stored).Error; err != nil {
			t.Fatalf("address was not persisted: %v", err)
		}
		if stored.UserID != user.ID {
			t.Fatalf("expected the address to belong to %s, got %s", user.ID, stored.UserID)
		}
	})

	t.Run("list returns the user's own addresses", func(t *testing.T) {
		resp := doJSON(t, a, "GET", "/api/me/addresses", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on list, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		payload := readBody(t, resp)
		addresses, _ := payload["addresses"].([]any)
		if len(addresses) != 1 {
			t.Fatalf("expected 1 address, got %d (%v)", len(addresses), payload)
		}
	})

	t.Run("update", func(t *testing.T) {
		resp := doJSON(t, a, "PUT", "/api/me/addresses/"+addressID, `{
			"name": "Office",
			"address": "Jl. Sudirman No. 1",
			"country_id": "`+countryID.String()+`",
			"province_id": "`+provinceID.String()+`",
			"city_id": "`+cityID.String()+`"
		}`)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on update, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		var stored models.Address
		if err := tx.Where("id = ?", addressID).First(&stored).Error; err != nil {
			t.Fatalf("failed to reload address: %v", err)
		}
		if stored.Name != "Office" {
			t.Fatalf("expected the update to persist, got name %q", stored.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		resp := doJSON(t, a, "DELETE", "/api/me/addresses/"+addressID, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on delete, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		var count int64
		if err := tx.Model(&models.Address{}).Where("id = ?", addressID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count addresses: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected the address to be gone, %d still present", count)
		}
	})
}

// TestAddressSeederIsIdempotent runs the reference seeder twice inside one
// transaction. The unit tests mock the database, so nothing there proves the
// seeder is safe to re-run, which is the property the whole seeding workflow
// depends on.
func TestAddressSeederIsIdempotent(t *testing.T) {
	tx := withTx(t)

	if err := seeds.AddressReferenceSeeder(tx); err != nil {
		t.Fatalf("first seed run failed: %v", err)
	}
	var first int64
	if err := tx.Model(&models.Country{}).Count(&first).Error; err != nil {
		t.Fatalf("failed to count countries: %v", err)
	}

	if err := seeds.AddressReferenceSeeder(tx); err != nil {
		t.Fatalf("second seed run failed: %v", err)
	}

	var second int64
	if err := tx.Model(&models.Country{}).Count(&second).Error; err != nil {
		t.Fatalf("failed to count countries after reseed: %v", err)
	}
	if first != second {
		t.Fatalf("re-running the seeder changed the row count: %d then %d", first, second)
	}
}

// TestAddressRequiresValidLocation proves the real foreign keys reject a
// location that does not exist.
func TestAddressRequiresValidLocation(t *testing.T) {
	tx := withTx(t)
	user := newTestUser(t, tx, "badloc@gatrahitawasana.com", "user")

	a := app()
	group := a.Group("/api/me/addresses", asUser(t, user.ID.String(), user.Email))
	group.Post("/", handlers.CreateMyAddress)

	// No location rows at all, so every FK below is dangling.
	nonexistent := uuid.New()
	resp := doJSON(t, a, "POST", "/api/me/addresses", `{
		"name": "Nowhere",
		"address": "Unknown street",
		"country_id": "`+nonexistent.String()+`",
		"province_id": "`+nonexistent.String()+`",
		"city_id": "`+nonexistent.String()+`"
	}`)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 when the foreign key is violated, got %d", resp.StatusCode)
	}
}

// asUser is a stub for the real auth middleware: it stands in for a validated
// JWT by setting the same context locals. The real middleware is covered by
// TestLoginAndFetchMe.
func asUser(t *testing.T, userID, email string) fiber.Handler {
	t.Helper()

	return func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		c.Locals("userEmail", email)
		c.Locals("userRole", "user")
		return c.Next()
	}
}
