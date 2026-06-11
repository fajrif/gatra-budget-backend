package models

import (
	"time"

	"github.com/google/uuid"
)

type Country struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Code      string    `gorm:"unique;not null;size:2" json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Country) TableName() string {
	return "countries"
}
