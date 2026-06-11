-- Migration: Create cities table
-- Description: Reference table for cities/regencies in the address hierarchy

CREATE TABLE IF NOT EXISTS cities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    province_id UUID NOT NULL REFERENCES provinces(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cities_province_id ON cities(province_id);
CREATE INDEX IF NOT EXISTS idx_cities_name ON cities(name);

COMMENT ON TABLE cities IS 'Reference table for cities/regencies (kabupaten/kota), scoped to a province';

-- DOWN
-- DROP TABLE IF EXISTS cities;
