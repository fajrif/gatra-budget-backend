-- Migration: Create countries table
-- Description: Reference table for countries in the address hierarchy

CREATE TABLE IF NOT EXISTS countries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    code CHAR(2) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_countries_code ON countries(code);
CREATE INDEX IF NOT EXISTS idx_countries_name ON countries(name);

COMMENT ON TABLE countries IS 'Reference table for countries';
COMMENT ON COLUMN countries.code IS 'ISO 3166-1 alpha-2 country code (e.g. ID, US)';

-- DOWN
-- DROP TABLE IF EXISTS countries;
