package models

import (
	"time"

	"github.com/google/uuid"
)

type OperationalCostType struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	NamaBiaya string    `gorm:"not null" json:"nama_biaya"`
	Deskripsi *string   `json:"deskripsi,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (OperationalCostType) TableName() string {
	return "operational_cost_types"
}
