package models

import (
	"time"

	"github.com/google/uuid"
)

// DeductionType is the master data for a deduction (potongan) a client may take
// when paying an invoice, e.g. PPh Badan or Denda.
type DeductionType struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nama      string    `gorm:"not null" json:"nama"`
	Deskripsi *string   `json:"deskripsi,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DeductionType) TableName() string {
	return "deduction_types"
}
