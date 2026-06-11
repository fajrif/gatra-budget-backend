package models

import (
	"time"

	"github.com/google/uuid"
)

type Area struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CityID    uuid.UUID `gorm:"type:uuid;not null" json:"city_id"`
	City      City      `gorm:"foreignKey:CityID" json:"city,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Area) TableName() string {
	return "areas"
}
