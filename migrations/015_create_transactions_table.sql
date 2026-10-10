-- Migration: Create transactions table
-- Description: Realized expenses ("Realisasi") against a project/cost type.
-- bulan_realisasi and the management-fee fields are computed server-side from
-- the owning project's fee rate — see helpers/transaction_helper.go.

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    cost_type_id UUID NOT NULL REFERENCES cost_types(id),
    tanggal_transaksi DATE NOT NULL,
    tanggal_po_tagihan DATE,
    bulan_realisasi VARCHAR(7),
    deskripsi_realisasi VARCHAR(500),
    jumlah_realisasi NUMERIC(20,2) NOT NULL,
    persentase_management_fee NUMERIC(5,2),
    nilai_management_fee NUMERIC(20,2),
    jumlah_tenaga_kerja INTEGER,
    bukti_transaksi_url VARCHAR(500),
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT transactions_jumlah_realisasi_minimum CHECK (jumlah_realisasi > 100000)
);

CREATE INDEX IF NOT EXISTS idx_transactions_project_id ON transactions(project_id);
CREATE INDEX IF NOT EXISTS idx_transactions_cost_type_id ON transactions(cost_type_id);
CREATE INDEX IF NOT EXISTS idx_transactions_tanggal_transaksi ON transactions(tanggal_transaksi);
CREATE INDEX IF NOT EXISTS idx_transactions_bulan_realisasi ON transactions(bulan_realisasi);

COMMENT ON TABLE transactions IS 'Realized expenses against a project/cost type; drives absorption, remaining budget, and management-fee calculations';

-- DOWN
-- DROP TABLE IF EXISTS transactions;
