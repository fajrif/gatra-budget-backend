package seeds

import (
	"fmt"

	"gatra-budget-backend/models"
	"gorm.io/gorm"
)

// OperationalCostTypesSeeder seeds the starter operational expense categories.
// Safe to re-run: uses FirstOrCreate on the fixed ID.
func OperationalCostTypesSeeder(db *gorm.DB) error {
	fmt.Println("📝 Seeding operational cost types...")

	names := []struct{ id, name string }{
		{"30000000-0000-4000-8000-000000000001", "Sewa Kantor"},
		{"30000000-0000-4000-8000-000000000002", "Listrik & Air"},
		{"30000000-0000-4000-8000-000000000003", "Konsumsi"},
		{"30000000-0000-4000-8000-000000000004", "ATK"},
		{"30000000-0000-4000-8000-000000000005", "Transportasi"},
		{"30000000-0000-4000-8000-000000000006", "Lainnya"},
	}

	for _, n := range names {
		item := models.OperationalCostType{ID: id(n.id), NamaBiaya: n.name}
		if err := db.Where("id = ?", item.ID).FirstOrCreate(&item).Error; err != nil {
			return fmt.Errorf("seed operational cost type %q: %w", n.name, err)
		}
	}
	return nil
}
