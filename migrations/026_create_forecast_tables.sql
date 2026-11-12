-- Migration: Create forecast tables
-- Description: "Forecast Kas" sheets. A sheet is a cash-forecast worksheet for one
-- month: an opening balance, planned expenses typed in by hand, and the unpaid
-- (Submit) invoices the user ticked as likely to be paid that month. Purely an
-- estimate; nothing here changes invoices or transactions.

CREATE TABLE IF NOT EXISTS forecast_sheets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(255) NOT NULL,
    bulan VARCHAR(7) NOT NULL,
    saldo NUMERIC(20,2) NOT NULL DEFAULT 0,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_forecast_sheets_nama_unique ON forecast_sheets (LOWER(nama));

CREATE TABLE IF NOT EXISTS forecast_expenses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    forecast_sheet_id UUID NOT NULL REFERENCES forecast_sheets(id) ON DELETE CASCADE,
    item VARCHAR(255) NOT NULL,
    tanggal DATE NOT NULL,
    jumlah NUMERIC(20,2) NOT NULL,
    urutan INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT forecast_expenses_jumlah_positive CHECK (jumlah > 0)
);

CREATE INDEX IF NOT EXISTS idx_forecast_expenses_sheet_id ON forecast_expenses(forecast_sheet_id);

-- Only the invoices the user ticked are stored; the invoice list itself is
-- always read live from invoices.
CREATE TABLE IF NOT EXISTS forecast_invoices (
    forecast_sheet_id UUID NOT NULL REFERENCES forecast_sheets(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    PRIMARY KEY (forecast_sheet_id, invoice_id)
);

CREATE INDEX IF NOT EXISTS idx_forecast_invoices_invoice_id ON forecast_invoices(invoice_id);

COMMENT ON TABLE forecast_sheets IS 'Monthly cash-forecast worksheets (estimates only)';

-- DOWN
-- DROP TABLE IF EXISTS forecast_invoices;
-- DROP TABLE IF EXISTS forecast_expenses;
-- DROP TABLE IF EXISTS forecast_sheets;
