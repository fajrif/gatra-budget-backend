-- Migration: Create cost_operations table
-- Description: Operational expenses (office costs, meals, ...) that are not
-- tied to any project. bulan_realisasi is computed server-side.

CREATE TABLE IF NOT EXISTS cost_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operational_cost_type_id UUID NOT NULL REFERENCES operational_cost_types(id),
    tanggal_transaksi DATE NOT NULL,
    bulan_realisasi VARCHAR(7),
    deskripsi VARCHAR(500),
    jumlah NUMERIC(20,2) NOT NULL,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT cost_operations_jumlah_minimum CHECK (jumlah > 100000)
);

CREATE INDEX IF NOT EXISTS idx_cost_operations_type_id ON cost_operations(operational_cost_type_id);
CREATE INDEX IF NOT EXISTS idx_cost_operations_tanggal ON cost_operations(tanggal_transaksi);
CREATE INDEX IF NOT EXISTS idx_cost_operations_bulan ON cost_operations(bulan_realisasi);

COMMENT ON TABLE cost_operations IS 'Operational expenses not tied to any project';

-- DOWN
-- DROP TABLE IF EXISTS cost_operations;
