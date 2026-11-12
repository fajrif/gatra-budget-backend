package models

import (
	"time"

	"github.com/google/uuid"
)

// ForecastSheet is a monthly cash-forecast worksheet: an opening balance, hand
// typed planned expenses and the ticked unpaid invoices. Estimates only.
type ForecastSheet struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nama      string    `gorm:"not null" json:"nama"`
	Bulan     string    `gorm:"not null" json:"bulan"` // YYYY-MM
	Saldo     float64   `gorm:"type:numeric(20,2);not null;default:0" json:"saldo"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ForecastSheet) TableName() string {
	return "forecast_sheets"
}

// ForecastExpense is one planned expense line of a sheet.
type ForecastExpense struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ForecastSheetID uuid.UUID `gorm:"type:uuid;not null" json:"forecast_sheet_id"`
	Item            string    `gorm:"not null" json:"item"`
	Tanggal         Date      `gorm:"not null" json:"tanggal"`
	Jumlah          float64   `gorm:"type:numeric(20,2);not null" json:"jumlah"`
	Urutan          int       `gorm:"not null;default:0" json:"urutan"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ForecastExpense) TableName() string {
	return "forecast_expenses"
}

// ForecastInvoice marks an unpaid invoice as ticked on a sheet.
type ForecastInvoice struct {
	ForecastSheetID uuid.UUID `gorm:"type:uuid;primaryKey" json:"forecast_sheet_id"`
	InvoiceID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"invoice_id"`
}

func (ForecastInvoice) TableName() string {
	return "forecast_invoices"
}

// ForecastExpenseRequest is one expense line in a save request.
type ForecastExpenseRequest struct {
	Item    string  `json:"item"`
	Tanggal string  `json:"tanggal"`
	Jumlah  float64 `json:"jumlah"`
}

// ForecastSheetRequest is the whole sheet as sent on save: the expenses and the
// ticked invoices replace whatever the sheet held before.
type ForecastSheetRequest struct {
	Nama       string                   `json:"nama"`
	Bulan      string                   `json:"bulan"`
	Saldo      float64                  `json:"saldo"`
	Expenses   []ForecastExpenseRequest `json:"expenses"`
	InvoiceIDs []string                 `json:"invoice_ids"`
}
