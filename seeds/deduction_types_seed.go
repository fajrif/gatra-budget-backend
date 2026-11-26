package seeds

import (
	"fmt"

	"gatra-budget-backend/models"
	"gorm.io/gorm"
)

// DeductionTypesSeeder seeds the starter deduction types (master potongan).
// Safe to re-run: uses FirstOrCreate on the fixed ID.
func DeductionTypesSeeder(db *gorm.DB) error {
	fmt.Println("📝 Seeding deduction types...")

	items := []struct{ id, nama, deskripsi string }{
		{"50000000-0000-4000-8000-000000000001", "PPh Badan", "Pajak penghasilan badan yang dipotong pemberi kerja"},
		{"50000000-0000-4000-8000-000000000002", "Denda Administratif", "Denda administratif atas keterlambatan atau ketidaksesuaian dokumen"},
		{"50000000-0000-4000-8000-000000000003", "Denda", "Denda lain yang dipotong saat pembayaran"},
	}

	for _, n := range items {
		desc := n.deskripsi
		item := models.DeductionType{ID: id(n.id), Nama: n.nama, Deskripsi: &desc}
		if err := db.Where("id = ?", item.ID).FirstOrCreate(&item).Error; err != nil {
			return fmt.Errorf("seed deduction type %q: %w", n.nama, err)
		}
	}
	return nil
}
