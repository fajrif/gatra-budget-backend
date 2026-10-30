package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID                uuid.UUID     `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	NoSP2K            string        `gorm:"column:no_sp2k;not null" json:"no_sp2k"`
	NoPerjanjian      *string       `json:"no_perjanjian,omitempty"`
	NoAmandemen       *string       `json:"no_amandemen,omitempty"`
	TanggalPerjanjian *Date         `json:"tanggal_perjanjian,omitempty"`
	JudulPekerjaan    string        `gorm:"not null" json:"judul_pekerjaan"`
	JangkaWaktu       *int          `json:"jangka_waktu,omitempty"`
	TanggalMulai      Date          `gorm:"not null" json:"tanggal_mulai"`
	TanggalSelesai    Date          `gorm:"not null" json:"tanggal_selesai"`
	NilaiPekerjaan    float64       `gorm:"type:numeric(20,2);not null" json:"nilai_pekerjaan"`
	ClientID          *uuid.UUID    `gorm:"type:uuid" json:"client_id,omitempty"`
	Client            *Client       `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	ContractTypeID    *uuid.UUID    `gorm:"type:uuid" json:"contract_type_id,omitempty"`
	ContractType      *ContractType `gorm:"foreignKey:ContractTypeID" json:"contract_type,omitempty"`
	CreatedBy         string        `json:"created_by"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

func (Project) TableName() string {
	return "projects"
}

// ProjectRequest is the client-bindable subset of Project. It deliberately
// excludes JangkaWaktu (always server-computed from TanggalMulai/TanggalSelesai)
// and CreatedBy (set from the authenticated user), mirroring UserRequest's role.
type ProjectRequest struct {
	NoSP2K            string  `json:"no_sp2k"`
	NoPerjanjian      *string `json:"no_perjanjian"`
	NoAmandemen       *string `json:"no_amandemen"`
	TanggalPerjanjian *string `json:"tanggal_perjanjian"`
	JudulPekerjaan    string  `json:"judul_pekerjaan"`
	TanggalMulai      string  `json:"tanggal_mulai"`
	TanggalSelesai    string  `json:"tanggal_selesai"`
	NilaiPekerjaan    float64 `json:"nilai_pekerjaan"`
	ClientID          *string `json:"client_id"`
	ContractTypeID    *string `json:"contract_type_id"`
}
