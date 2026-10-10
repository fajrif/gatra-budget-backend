package models

import (
	"time"

	"github.com/google/uuid"
)

type CostType struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	NamaBiaya string    `gorm:"not null" json:"nama_biaya"`
	Kode      *string   `json:"kode,omitempty"`
	Deskripsi *string   `json:"deskripsi,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CostType) TableName() string {
	return "cost_types"
}
