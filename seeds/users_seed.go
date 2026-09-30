package seeds

import (
	"fmt"
	"time"

	"gatra-budget-backend/models"
	"gorm.io/gorm"
)

// Bcrypt hash of "Password123!", shared by both seeded accounts.
// Change it after first login.
const defaultUserPasswordHash = "$2a$10$QCG0U7KDYgu88G7K5Y7THe8MIs84KVvuJy1ljoJY7XnK/uAjPmpsW"

// UsersSeeder seeds the initial admin and user accounts.
// Safe to re-run: uses FirstOrCreate on email so existing rows are left alone.
func UsersSeeder(db *gorm.DB) error {
	fmt.Println("📝 Seeding users...")

	adminCreatedAt := time.Date(2006, 1, 1, 0, 0, 0, 0, time.UTC)
	userCreatedAt := time.Date(2005, 10, 17, 0, 0, 0, 0, time.UTC)

	users := []models.User{
		{
			ID:           id("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			Email:        "admin@gatrahitawasana.com",
			PasswordHash: defaultUserPasswordHash,
			FullName:     "ADMIN",
			Role:         "admin",
			CreatedAt:    adminCreatedAt,
		},
		{
			ID:           id("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"),
			Email:        "user@gatrahitawasana.com",
			PasswordHash: defaultUserPasswordHash,
			FullName:     "USER",
			Role:         "user",
			CreatedAt:    userCreatedAt,
		},
	}

	for _, u := range users {
		if err := db.Where("email = ?", u.Email).FirstOrCreate(&u).Error; err != nil {
			return fmt.Errorf("user %s: %w", u.Email, err)
		}
	}

	fmt.Printf("  ✓ Users: %d records (password: \"Password123!\")\n", len(users))
	return nil
}
