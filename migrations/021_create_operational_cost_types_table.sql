-- Migration: Create operational_cost_types table
-- Description: Category master data for operational expenses (cost_operations),
-- kept separate from project cost_types.

CREATE TABLE IF NOT EXISTS operational_cost_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama_biaya VARCHAR(255) NOT NULL,
    deskripsi TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_operational_cost_types_nama_unique ON operational_cost_types (LOWER(nama_biaya));

COMMENT ON TABLE operational_cost_types IS 'Category master data for operational (non-project) expenses';

-- DOWN
-- DROP TABLE IF EXISTS operational_cost_types;
