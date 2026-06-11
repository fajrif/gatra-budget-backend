-- Migration: Create provinces table
-- Description: Reference table for provinces/states in the address hierarchy

CREATE TABLE IF NOT EXISTS provinces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_provinces_country_id ON provinces(country_id);
CREATE INDEX IF NOT EXISTS idx_provinces_name ON provinces(name);

COMMENT ON TABLE provinces IS 'Reference table for provinces/states, scoped to a country';

-- DOWN
-- DROP TABLE IF EXISTS provinces;
