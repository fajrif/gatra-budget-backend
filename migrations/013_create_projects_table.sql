-- Migration: Create projects table
-- Description: Projects being budgeted and tracked against a contract value

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    no_sp2k VARCHAR(255) NOT NULL,
    no_perjanjian VARCHAR(255),
    no_amandemen VARCHAR(255),
    tanggal_perjanjian DATE,
    judul_pekerjaan VARCHAR(500) NOT NULL,
    jangka_waktu INTEGER,
    tanggal_mulai DATE NOT NULL,
    tanggal_selesai DATE,
    nilai_pekerjaan NUMERIC(20,2) NOT NULL,
    management_fee NUMERIC(20,2),
    tarif_management_fee_persen NUMERIC(5,2),
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    contract_type_id UUID REFERENCES contract_types(id) ON DELETE SET NULL,
    status_kontrak VARCHAR(20) NOT NULL DEFAULT 'Active' CHECK (status_kontrak IN ('Active', 'Pending', 'Non Aktif')),
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_client_id ON projects(client_id);
CREATE INDEX IF NOT EXISTS idx_projects_contract_type_id ON projects(contract_type_id);
CREATE INDEX IF NOT EXISTS idx_projects_status_kontrak ON projects(status_kontrak);

COMMENT ON TABLE projects IS 'Projects being budgeted and tracked against a contract value (nilai_pekerjaan)';
COMMENT ON COLUMN projects.jangka_waktu IS 'Duration in months, server-computed from tanggal_mulai/tanggal_selesai';

-- DOWN
-- DROP TABLE IF EXISTS projects;
