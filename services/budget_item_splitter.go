// Package services holds stateful business-logic objects that need a bound
// *gorm.DB across several sequential writes — unlike helpers/, which holds
// pure, stateless functions.
package services

import (
	"fmt"
	"math"

	"gatra-budget-backend/models"

	"gorm.io/gorm"
)

// DefaultSplitMonths is the fallback number of months a monthly budget item
// splits into when the owning project has no jangka_waktu yet.
const DefaultSplitMonths = 12

// BudgetItemSplitter builds and re-syncs the parent/child BudgetItem rows
// behind a project's "Anggaran" (budget) entries. A "monthly" budget item is
// stored as one parent row plus N evenly-split child rows (one per month of
// the project's jangka_waktu); a "lumpsum" budget item is a single parent row
// with no children. Ported from the verified Rails BudgetItemSplitter.
type BudgetItemSplitter struct {
	DB      *gorm.DB
	Project *models.Project
}

func NewBudgetItemSplitter(db *gorm.DB, project *models.Project) *BudgetItemSplitter {
	return &BudgetItemSplitter{DB: db, Project: project}
}

// Create builds one parent BudgetItem row; when kategoriAnggaran is
// "monthly", it also builds the evenly-split monthly child rows.
func (s *BudgetItemSplitter) Create(costType *models.CostType, kategoriAnggaran string, totalAnggaran float64, deskripsi *string) (*models.BudgetItem, error) {
	var parent models.BudgetItem

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		periodeBulan := s.Project.TanggalMulai.Time.Format("2006-01")
		kategori := kategoriAnggaran
		jenisBiayaName := costType.NamaBiaya

		parent = models.BudgetItem{
			ProjectID:         s.Project.ID,
			CostTypeID:        costType.ID,
			JenisBiayaName:    &jenisBiayaName,
			KategoriAnggaran:  &kategori,
			TotalAnggaran:     &totalAnggaran,
			JumlahAnggaran:    totalAnggaran,
			DeskripsiAnggaran: deskripsi,
			PeriodeBulan:      &periodeBulan,
			IsParent:          true,
			ParentBudgetID:    nil,
		}
		if err := tx.Create(&parent).Error; err != nil {
			return err
		}

		if kategoriAnggaran == models.KategoriAnggaranMonthly {
			return s.buildChildren(tx, &parent, totalAnggaran, deskripsi)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &parent, nil
}

// Resync updates the parent row in place; if it is a monthly budget item, the
// existing children are destroyed and rebuilt against the new total — this is
// how edits to a monthly budget item's total propagate to its children.
func (s *BudgetItemSplitter) Resync(parent *models.BudgetItem, totalAnggaran float64, deskripsi *string, costType *models.CostType) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if costType != nil {
			parent.CostTypeID = costType.ID
			jenisBiayaName := costType.NamaBiaya
			parent.JenisBiayaName = &jenisBiayaName
		}
		parent.TotalAnggaran = &totalAnggaran
		parent.JumlahAnggaran = totalAnggaran
		if deskripsi != nil {
			parent.DeskripsiAnggaran = deskripsi
		}

		if err := tx.Save(parent).Error; err != nil {
			return err
		}

		if parent.KategoriAnggaran == nil || *parent.KategoriAnggaran != models.KategoriAnggaranMonthly {
			return nil
		}

		if err := tx.Where("parent_budget_id = ?", parent.ID).Delete(&models.BudgetItem{}).Error; err != nil {
			return err
		}
		return s.buildChildren(tx, parent, totalAnggaran, parent.DeskripsiAnggaran)
	})
}

// buildChildren creates n = Project.JangkaWaktu (or DefaultSplitMonths) child
// rows under parent, one per month starting at the project's start month,
// each holding an even slice of totalAnggaran with the rounding remainder
// pushed onto the last child so the children sum exactly to the total.
func (s *BudgetItemSplitter) buildChildren(tx *gorm.DB, parent *models.BudgetItem, totalAnggaran float64, deskripsi *string) error {
	n := DefaultSplitMonths
	if s.Project.JangkaWaktu != nil && *s.Project.JangkaWaktu > 0 {
		n = *s.Project.JangkaWaktu
	}

	amounts := SplitAmount(totalAnggaran, n)
	startMonth := s.Project.TanggalMulai.Time
	kategori := models.KategoriAnggaranMonthly

	for i, amount := range amounts {
		bulanKe := i + 1
		periodeBulan := startMonth.AddDate(0, i, 0).Format("2006-01")

		childDeskripsi := fmt.Sprintf("Bulan %d", bulanKe)
		if deskripsi != nil && *deskripsi != "" {
			childDeskripsi = *deskripsi + " - " + childDeskripsi
		}

		child := models.BudgetItem{
			ProjectID:         s.Project.ID,
			CostTypeID:        parent.CostTypeID,
			JenisBiayaName:    parent.JenisBiayaName,
			KategoriAnggaran:  &kategori,
			TotalAnggaran:     &totalAnggaran,
			DeskripsiAnggaran: &childDeskripsi,
			PeriodeBulan:      &periodeBulan,
			JumlahAnggaran:    amount,
			BulanKe:           &bulanKe,
			ParentBudgetID:    &parent.ID,
			IsParent:          false,
		}
		if err := tx.Create(&child).Error; err != nil {
			return err
		}
	}
	return nil
}

// SplitAmount divides total evenly across n parts, each rounded to 2 decimal
// places, with the rounding remainder pushed onto the last part so the parts
// always sum exactly to total.
func SplitAmount(total float64, n int) []float64 {
	if n <= 0 {
		return nil
	}

	perMonth := math.Round((total/float64(n))*100) / 100
	remainder := total - (perMonth * float64(n))

	amounts := make([]float64, n)
	for i := range amounts {
		amounts[i] = perMonth
	}
	amounts[n-1] = math.Round((amounts[n-1]+remainder)*100) / 100

	return amounts
}
