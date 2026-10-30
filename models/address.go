package models

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Name       string     `gorm:"not null" json:"name"`
	Address    string     `gorm:"not null" json:"address"`
	CountryID  uuid.UUID  `gorm:"type:uuid;not null" json:"country_id"`
	ProvinceID uuid.UUID  `gorm:"type:uuid;not null" json:"province_id"`
	CityID     uuid.UUID  `gorm:"type:uuid;not null" json:"city_id"`
	AreaID     *uuid.UUID `gorm:"type:uuid" json:"area_id,omitempty"`
	Zipcode    string     `json:"zipcode"`
	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Country    Country    `gorm:"foreignKey:CountryID" json:"country,omitempty"`
	Province   Province   `gorm:"foreignKey:ProvinceID" json:"province,omitempty"`
	City       City       `gorm:"foreignKey:CityID" json:"city,omitempty"`
	Area       *Area      `gorm:"foreignKey:AreaID" json:"area,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (Address) TableName() string {
	return "addresses"
}
