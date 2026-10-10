package integration

import (
	"testing"

	"gatra-budget-backend/models"
	"gatra-budget-backend/services"

	"gorm.io/gorm"
)

// newBudgetItemTestProject creates a project with a 4-month duration
// (2024-01-01 to 2024-05-01) and a cost type to exercise the splitter against.
// Both records are created inside tx so they are rolled back with everything
// else the test does.
func newBudgetItemTestProject(t *testing.T, tx *gorm.DB) (project models.Project, costType models.CostType) {
	t.Helper()

	jangkaWaktu := 4
	project = models.Project{
		NoSP2K:         "SP2K-001",
		JudulPekerjaan: "Pembangunan Gedung",
		TanggalMulai:   models.Date{Time: mustParseDate(t, "2024-01-01")},
		NilaiPekerjaan: 1_000_000_000,
		JangkaWaktu:    &jangkaWaktu,
		StatusKontrak:  "Active",
	}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	costType = models.CostType{NamaBiaya: "Gaji & Tunjangan"}
	if err := tx.Create(&costType).Error; err != nil {
		t.Fatalf("failed to create cost type: %v", err)
	}

	return project, costType
}

func TestBudgetItemSplitterCreateMonthly(t *testing.T) {
	tx := withTx(t)
	project, costType := newBudgetItemTestProject(t, tx)

	splitter := services.NewBudgetItemSplitter(tx, &project)
	parent, err := splitter.Create(&costType, models.KategoriAnggaranMonthly, 1_000_000, nil)
	if err != nil {
		t.Fatalf("failed to create monthly budget item: %v", err)
	}
	if !parent.IsParent {
		t.Fatal("expected the created row to be the parent")
	}

	var children []models.BudgetItem
	if err := tx.Where("parent_budget_id = ?", parent.ID).Order("bulan_ke ASC").Find(&children).Error; err != nil {
		t.Fatalf("failed to load children: %v", err)
	}

	if len(children) != 4 {
		t.Fatalf("expected 4 children (project.jangka_waktu), got %d", len(children))
	}

	expectedMonths := []string{"2024-01", "2024-02", "2024-03", "2024-04"}
	var sum float64
	for i, child := range children {
		if child.PeriodeBulan == nil || *child.PeriodeBulan != expectedMonths[i] {
			t.Fatalf("child %d: expected periode_bulan %s, got %v", i, expectedMonths[i], child.PeriodeBulan)
		}
		if child.BulanKe == nil || *child.BulanKe != i+1 {
			t.Fatalf("child %d: expected bulan_ke %d, got %v", i, i+1, child.BulanKe)
		}
		if child.IsParent {
			t.Fatalf("child %d: expected is_parent false", i)
		}
		sum += child.JumlahAnggaran
	}

	if diff := sum - 1_000_000; diff > 0.01 || diff < -0.01 {
		t.Fatalf("expected children to sum to 1,000,000, got %v", sum)
	}

	// The remainder lands on the last child: 1,000,000 / 4 = 250,000 exactly,
	// so re-verify with a non-evenly-divisible total that it actually differs.
	lastAmount := children[3].JumlahAnggaran
	firstAmount := children[0].JumlahAnggaran
	if lastAmount != firstAmount {
		t.Fatalf("expected an evenly divisible total to split equally, got first=%v last=%v", firstAmount, lastAmount)
	}
}

func TestBudgetItemSplitterCreateMonthlyWithRemainder(t *testing.T) {
	tx := withTx(t)
	project, costType := newBudgetItemTestProject(t, tx)

	splitter := services.NewBudgetItemSplitter(tx, &project)
	parent, err := splitter.Create(&costType, models.KategoriAnggaranMonthly, 1_000_000.01, nil)
	if err != nil {
		t.Fatalf("failed to create monthly budget item: %v", err)
	}

	var children []models.BudgetItem
	if err := tx.Where("parent_budget_id = ?", parent.ID).Order("bulan_ke ASC").Find(&children).Error; err != nil {
		t.Fatalf("failed to load children: %v", err)
	}

	var sum float64
	for _, child := range children {
		sum += child.JumlahAnggaran
	}
	if diff := sum - 1_000_000.01; diff > 0.001 || diff < -0.001 {
		t.Fatalf("expected children to sum exactly to 1,000,000.01, got %v", sum)
	}
	if children[0].JumlahAnggaran == children[3].JumlahAnggaran {
		t.Fatal("expected the rounding remainder to make the last child differ from the others")
	}
}

func TestBudgetItemSplitterCreateLumpsumHasNoChildren(t *testing.T) {
	tx := withTx(t)
	project, costType := newBudgetItemTestProject(t, tx)

	splitter := services.NewBudgetItemSplitter(tx, &project)
	parent, err := splitter.Create(&costType, models.KategoriAnggaranLumpsum, 500_000, nil)
	if err != nil {
		t.Fatalf("failed to create lumpsum budget item: %v", err)
	}

	var count int64
	if err := tx.Model(&models.BudgetItem{}).Where("parent_budget_id = ?", parent.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count children: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected a lumpsum budget item to have no children, got %d", count)
	}
}

func TestBudgetItemSplitterResyncRebuildsChildren(t *testing.T) {
	tx := withTx(t)
	project, costType := newBudgetItemTestProject(t, tx)

	splitter := services.NewBudgetItemSplitter(tx, &project)
	parent, err := splitter.Create(&costType, models.KategoriAnggaranMonthly, 1_000_000, nil)
	if err != nil {
		t.Fatalf("failed to create monthly budget item: %v", err)
	}

	var before []models.BudgetItem
	tx.Where("parent_budget_id = ?", parent.ID).Find(&before)
	oldIDs := make(map[string]bool)
	for _, c := range before {
		oldIDs[c.ID.String()] = true
	}

	if err := splitter.Resync(parent, 2_000_000, nil, nil); err != nil {
		t.Fatalf("failed to resync budget item: %v", err)
	}

	var after []models.BudgetItem
	if err := tx.Where("parent_budget_id = ?", parent.ID).Find(&after).Error; err != nil {
		t.Fatalf("failed to load children after resync: %v", err)
	}

	if len(after) != 4 {
		t.Fatalf("expected 4 rebuilt children, got %d", len(after))
	}

	var sum float64
	for _, c := range after {
		sum += c.JumlahAnggaran
		if oldIDs[c.ID.String()] {
			t.Fatalf("expected resync to destroy and recreate children, but found a surviving old child ID %s", c.ID)
		}
	}
	if diff := sum - 2_000_000; diff > 0.01 || diff < -0.01 {
		t.Fatalf("expected rebuilt children to sum to the new total 2,000,000, got %v", sum)
	}

	var reloadedParent models.BudgetItem
	if err := tx.Where("id = ?", parent.ID).First(&reloadedParent).Error; err != nil {
		t.Fatalf("failed to reload parent: %v", err)
	}
	if reloadedParent.JumlahAnggaran != 2_000_000 {
		t.Fatalf("expected parent jumlah_anggaran to be updated to 2,000,000, got %v", reloadedParent.JumlahAnggaran)
	}
}

func TestBudgetItemParentCascadeDeletesChildren(t *testing.T) {
	tx := withTx(t)
	project, costType := newBudgetItemTestProject(t, tx)

	splitter := services.NewBudgetItemSplitter(tx, &project)
	parent, err := splitter.Create(&costType, models.KategoriAnggaranMonthly, 1_000_000, nil)
	if err != nil {
		t.Fatalf("failed to create monthly budget item: %v", err)
	}

	if err := tx.Delete(&models.BudgetItem{}, "id = ?", parent.ID).Error; err != nil {
		t.Fatalf("failed to delete parent: %v", err)
	}

	var count int64
	if err := tx.Model(&models.BudgetItem{}).Where("parent_budget_id = ?", parent.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count children: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected ON DELETE CASCADE to remove children when the parent is deleted, found %d", count)
	}
}

func TestBudgetItemProjectCascadeDeletesAll(t *testing.T) {
	tx := withTx(t)
	project, costType := newBudgetItemTestProject(t, tx)

	splitter := services.NewBudgetItemSplitter(tx, &project)
	if _, err := splitter.Create(&costType, models.KategoriAnggaranMonthly, 1_000_000, nil); err != nil {
		t.Fatalf("failed to create monthly budget item: %v", err)
	}

	if err := tx.Delete(&models.Project{}, "id = ?", project.ID).Error; err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	var count int64
	if err := tx.Model(&models.BudgetItem{}).Where("project_id = ?", project.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count budget items: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected ON DELETE CASCADE from project to remove all its budget items, found %d", count)
	}
}
