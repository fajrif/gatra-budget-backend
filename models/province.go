package models

import (
	"time"

	"github.com/google/uuid"
)

type Province struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CountryID uuid.UUID `gorm:"type:uuid;not null" json:"country_id"`
	Country   Country   `gorm:"foreignKey:CountryID" json:"country,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Province) TableName() string {
	return "provinces"
}
