package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
    Email        string    `gorm:"unique;not null" json:"email"`
    PasswordHash string    `gorm:"not null" json:"-"`
    FullName     string    `gorm:"not null" json:"full_name"`
    Role         string    `gorm:"default:'user'" json:"role"`
    PhotoUrl     *string   `json:"photo_url,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type UserRequest struct {
	Email    string `json:"email,omitempty" example:"admin@example.com"`
	Password string `json:"password,omitempty" example:"newpassword123"`
	FullName string `json:"full_name,omitempty" example:"John Doe Updated"`
	Role     string `json:"role,omitempty" example:"admin"`
}

func (User) TableName() string {
    return "users"
}
