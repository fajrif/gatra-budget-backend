package models

import (
	"time"

	"github.com/google/uuid"
)

// CostOperation is an operational expense that belongs to no project.
type CostOperation struct {
	ID                    uuid.UUID            `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	OperationalCostTypeID uuid.UUID            `gorm:"type:uuid;not null" json:"operational_cost_type_id"`
	OperationalCostType   *OperationalCostType `gorm:"foreignKey:OperationalCostTypeID" json:"operational_cost_type,omitempty"`
	TanggalTransaksi      Date                 `gorm:"not null" json:"tanggal_transaksi"`
	BulanRealisasi        *string              `json:"bulan_realisasi,omitempty"`
	Deskripsi             *string              `json:"deskripsi,omitempty"`
	Jumlah                float64              `gorm:"type:numeric(20,2);not null" json:"jumlah"`
	CreatedBy             string               `json:"created_by"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

func (CostOperation) TableName() string {
	return "cost_operations"
}

// CostOperationRequest is the client-bindable subset of CostOperation.
type CostOperationRequest struct {
	OperationalCostTypeID string  `json:"operational_cost_type_id"`
	TanggalTransaksi      string  `json:"tanggal_transaksi"`
	Deskripsi             *string `json:"deskripsi"`
	Jumlah                float64 `json:"jumlah"`
}
