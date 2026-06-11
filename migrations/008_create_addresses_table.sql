-- Migration: Create addresses table
-- Description: User-owned addresses with full hierarchy references

CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    country_id UUID NOT NULL REFERENCES countries(id),
    province_id UUID NOT NULL REFERENCES provinces(id),
    city_id UUID NOT NULL REFERENCES cities(id),
    area_id UUID REFERENCES areas(id),
    zipcode VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_addresses_user_id ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_addresses_city_id ON addresses(city_id);

COMMENT ON TABLE addresses IS 'User-owned addresses with full location hierarchy';
COMMENT ON COLUMN addresses.name IS 'Label for the address (e.g. Home, Office)';
COMMENT ON COLUMN addresses.area_id IS 'Optional sub-district reference';

-- DOWN
-- DROP TABLE IF EXISTS addresses;
