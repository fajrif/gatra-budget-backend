-- Migration: Create areas table
-- Description: Reference table for sub-district areas (kecamatan) in the address hierarchy

CREATE TABLE IF NOT EXISTS areas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_areas_city_id ON areas(city_id);
CREATE INDEX IF NOT EXISTS idx_areas_name ON areas(name);

COMMENT ON TABLE areas IS 'Reference table for sub-district areas (kecamatan), scoped to a city';

-- DOWN
-- DROP TABLE IF EXISTS areas;
