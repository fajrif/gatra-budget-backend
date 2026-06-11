package models

import (
	"time"

	"github.com/google/uuid"
)

type City struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	ProvinceID uuid.UUID `gorm:"type:uuid;not null" json:"province_id"`
	Province   Province  `gorm:"foreignKey:ProvinceID" json:"province,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (City) TableName() string {
	return "cities"
}
