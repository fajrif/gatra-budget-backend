package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ProjectID          uuid.UUID `gorm:"type:uuid;not null" json:"project_id"`
	Project            *Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	CostTypeID         uuid.UUID `gorm:"type:uuid;not null" json:"cost_type_id"`
	CostType           *CostType `gorm:"foreignKey:CostTypeID" json:"cost_type,omitempty"`
	TanggalTransaksi   Date      `gorm:"not null" json:"tanggal_transaksi"`
	BulanRealisasi     *string   `json:"bulan_realisasi,omitempty"`
	DeskripsiRealisasi *string   `json:"deskripsi_realisasi,omitempty"`
	JumlahRealisasi    float64   `gorm:"type:numeric(20,2);not null" json:"jumlah_realisasi"`
	JumlahTenagaKerja  *int      `json:"jumlah_tenaga_kerja,omitempty"`
	CreatedBy          string    `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (Transaction) TableName() string {
	return "transactions"
}

// TransactionRequest is the client-bindable subset of Transaction. It
// excludes the server-computed BulanRealisasi and CreatedBy (set from the
// authenticated user).
type TransactionRequest struct {
	ProjectID          string  `json:"project_id"`
	CostTypeID         string  `json:"cost_type_id"`
	TanggalTransaksi   string  `json:"tanggal_transaksi"`
	DeskripsiRealisasi *string `json:"deskripsi_realisasi"`
	JumlahRealisasi    float64 `json:"jumlah_realisasi"`
	JumlahTenagaKerja  *int    `json:"jumlah_tenaga_kerja"`
}
