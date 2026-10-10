package models

import (
	"time"

	"github.com/google/uuid"
)

type BudgetItem struct {
	ID                uuid.UUID   `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ProjectID         uuid.UUID   `gorm:"type:uuid;not null" json:"project_id"`
	Project           *Project    `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	CostTypeID        uuid.UUID   `gorm:"type:uuid;not null" json:"cost_type_id"`
	CostType          *CostType   `gorm:"foreignKey:CostTypeID" json:"cost_type,omitempty"`
	JenisBiayaName    *string     `json:"jenis_biaya_name,omitempty"`
	KategoriAnggaran  *string     `json:"kategori_anggaran,omitempty"`
	TotalAnggaran     *float64    `gorm:"type:numeric(20,2)" json:"total_anggaran,omitempty"`
	DeskripsiAnggaran *string     `json:"deskripsi_anggaran,omitempty"`
	PeriodeBulan      *string     `json:"periode_bulan,omitempty"`
	JumlahAnggaran    float64     `gorm:"type:numeric(20,2);not null" json:"jumlah_anggaran"`
	BulanKe           *int        `json:"bulan_ke,omitempty"`
	ParentBudgetID    *uuid.UUID  `gorm:"type:uuid" json:"parent_budget_id,omitempty"`
	ParentBudget      *BudgetItem `gorm:"foreignKey:ParentBudgetID" json:"-"`
	IsParent          bool        `gorm:"not null;default:false" json:"is_parent"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

func (BudgetItem) TableName() string {
	return "budget_items"
}

const (
	KategoriAnggaranMonthly = "monthly"
	KategoriAnggaranLumpsum = "lumpsum"
)

// BudgetItemRequest is the client-bindable subset of BudgetItem. It excludes
// every server-computed field (IsParent, ParentBudgetID, JenisBiayaName,
// PeriodeBulan, BulanKe), which the BudgetItemSplitter service owns.
type BudgetItemRequest struct {
	ProjectID         string  `json:"project_id"`
	CostTypeID        string  `json:"cost_type_id"`
	KategoriAnggaran  string  `json:"kategori_anggaran"`
	TotalAnggaran     float64 `json:"total_anggaran"`
	DeskripsiAnggaran *string `json:"deskripsi_anggaran"`
}
