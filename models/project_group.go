package models

import (
	"time"

	"github.com/google/uuid"
)

// ProjectGroup ties together the project records (contract periods) that make
// up one piece of work.
type ProjectGroup struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nama      string    `gorm:"not null" json:"nama"`
	Deskripsi *string   `json:"deskripsi,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ProjectGroup) TableName() string {
	return "project_groups"
}

// ProjectGroupRequest is the client-bindable subset of ProjectGroup.
type ProjectGroupRequest struct {
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi"`
}
