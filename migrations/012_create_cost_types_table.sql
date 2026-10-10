-- Migration: Create cost_types table
-- Description: Expense category master data referenced by budget items and transactions

CREATE TABLE IF NOT EXISTS cost_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama_biaya VARCHAR(255) NOT NULL,
    kode VARCHAR(50),
    deskripsi TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cost_types_nama_biaya ON cost_types(nama_biaya);

COMMENT ON TABLE cost_types IS 'Expense category master data referenced by budget items and transactions';

-- DOWN
-- DROP TABLE IF EXISTS cost_types;
