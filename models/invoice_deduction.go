package models

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceDeduction is one deduction (potongan) that explains part of the gap
// between an invoice's Nilai Tagihan and its Nilai Penerimaan.
type InvoiceDeduction struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	InvoiceID       uuid.UUID      `gorm:"type:uuid;not null" json:"invoice_id"`
	DeductionTypeID uuid.UUID      `gorm:"type:uuid;not null" json:"deduction_type_id"`
	DeductionType   *DeductionType `gorm:"foreignKey:DeductionTypeID" json:"deduction_type,omitempty"`
	Jumlah          float64        `gorm:"type:numeric(20,2);not null" json:"jumlah"`
	Keterangan      *string        `json:"keterangan,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (InvoiceDeduction) TableName() string {
	return "invoice_deductions"
}
