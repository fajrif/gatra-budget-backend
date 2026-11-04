-- Migration: Create project_groups table
-- Description: A project group ties together the separate project records (contract
-- periods) of one piece of work, e.g. "Pengadaan TAD Sumbar" covering Jan-Feb, Mar-Apr
-- and May-Oct contracts. Projects reference it optionally.

CREATE TABLE IF NOT EXISTS project_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nama VARCHAR(255) NOT NULL,
    deskripsi TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_groups_nama_unique ON project_groups (LOWER(nama));

ALTER TABLE projects ADD COLUMN IF NOT EXISTS project_group_id UUID REFERENCES project_groups(id);
CREATE INDEX IF NOT EXISTS idx_projects_project_group_id ON projects(project_group_id);

COMMENT ON TABLE project_groups IS 'Groups the project records (contract periods) of one piece of work';

-- DOWN
-- DROP INDEX IF EXISTS idx_projects_project_group_id;
-- ALTER TABLE projects DROP COLUMN IF EXISTS project_group_id;
-- DROP TABLE IF EXISTS project_groups;
