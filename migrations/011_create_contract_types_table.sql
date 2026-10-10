-- Migration: Create contract_types table
-- Description: Contract type master data referenced by projects (e.g. PO / Non-PO)

CREATE TABLE IF NOT EXISTS contract_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    code VARCHAR(50),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contract_types_name ON contract_types(name);

COMMENT ON TABLE contract_types IS 'Contract type master data referenced by projects';

-- DOWN
-- DROP TABLE IF EXISTS contract_types;
