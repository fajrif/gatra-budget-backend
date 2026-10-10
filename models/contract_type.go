package models

import (
	"time"

	"github.com/google/uuid"
)

type ContractType struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"unique;not null" json:"name"`
	Code        *string   `json:"code,omitempty"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ContractType) TableName() string {
	return "contract_types"
}
