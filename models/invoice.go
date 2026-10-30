package models

import (
	"time"

	"github.com/google/uuid"
)

// Invoice statuses. Clearing means the invoice has been paid.
const (
	InvoiceStatusSubmit   = "Submit"
	InvoiceStatusClearing = "Clearing"
)

type Invoice struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	NoInvoice    string    `gorm:"not null;uniqueIndex" json:"no_invoice"`
	ProjectID    uuid.UUID `gorm:"type:uuid;not null" json:"project_id"`
	Project      *Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	NamaInvoice  string    `gorm:"not null" json:"nama_invoice"`
	TglTagihan   Date      `gorm:"not null" json:"tgl_tagihan"`
	TglBayar     *Date     `json:"tgl_bayar,omitempty"`
	Status       string    `gorm:"not null;default:'Submit'" json:"status"`
	Nilai        float64   `gorm:"type:numeric(20,2);not null" json:"nilai"`
	Description  *string   `json:"description,omitempty"`
	ManajemenFee float64   `gorm:"type:numeric(20,2);not null;default:0" json:"manajemen_fee"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Invoice) TableName() string {
	return "invoices"
}

// InvoiceRequest is the client-bindable subset of Invoice. It excludes
// CreatedBy, which is set from the authenticated user.
type InvoiceRequest struct {
	NoInvoice    string   `json:"no_invoice"`
	ProjectID    string   `json:"project_id"`
	NamaInvoice  string   `json:"nama_invoice"`
	TglTagihan   string   `json:"tgl_tagihan"`
	TglBayar     *string  `json:"tgl_bayar"`
	Status       string   `json:"status"`
	Nilai        float64  `json:"nilai"`
	Description  *string  `json:"description"`
	ManajemenFee *float64 `json:"manajemen_fee"`
}
