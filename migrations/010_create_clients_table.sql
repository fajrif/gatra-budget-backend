-- Migration: Create clients table
-- Description: Client/customer master data referenced by projects

CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    contact_name VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(name);

COMMENT ON TABLE clients IS 'Client/customer master data referenced by projects';

-- DOWN
-- DROP TABLE IF EXISTS clients;
