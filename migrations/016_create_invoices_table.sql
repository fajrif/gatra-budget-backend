-- Migration: Create invoices table
-- Description: Invoices (tagihan) billed against a project. Status "Clearing"
-- means the invoice has been paid, and tgl_bayar is then required.

CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    no_invoice VARCHAR(100) NOT NULL UNIQUE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    nama_invoice VARCHAR(500) NOT NULL,
    tgl_tagihan DATE NOT NULL,
    tgl_bayar DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'Submit',
    nilai NUMERIC(20,2) NOT NULL,
    description TEXT,
    manajemen_fee NUMERIC(20,2) NOT NULL DEFAULT 0,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT invoices_status_valid CHECK (status IN ('Submit', 'Clearing')),
    CONSTRAINT invoices_nilai_positive CHECK (nilai > 0),
    CONSTRAINT invoices_manajemen_fee_range CHECK (manajemen_fee >= 0 AND manajemen_fee <= nilai),
    CONSTRAINT invoices_tgl_bayar_matches_status CHECK ((status = 'Clearing') = (tgl_bayar IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS idx_invoices_project_id ON invoices(project_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_tgl_tagihan ON invoices(tgl_tagihan);
CREATE INDEX IF NOT EXISTS idx_invoices_tgl_bayar ON invoices(tgl_bayar);

COMMENT ON TABLE invoices IS 'Invoices billed against a project; Clearing invoices count as received payments in their tgl_bayar month';
COMMENT ON COLUMN invoices.manajemen_fee IS 'Fixed management fee amount for this invoice; summed per project as its profit margin';

-- DOWN
-- DROP TABLE IF EXISTS invoices;
