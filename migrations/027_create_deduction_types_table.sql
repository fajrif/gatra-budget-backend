-- Migration: Create deduction_types table
-- Description: Master data for deductions (potongan) a client may take from an
-- invoice when paying it, e.g. PPh Badan, Denda Administratif, Denda.

CREATE TABLE IF NOT EXISTS deduction_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(255) NOT NULL,
    deskripsi TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deduction_types_nama_unique ON deduction_types (LOWER(nama));

COMMENT ON TABLE deduction_types IS 'Master data for invoice deductions (potongan) taken by the client at payment';

-- DOWN
-- DROP TABLE IF EXISTS deduction_types;
