-- Migration: Make projects.no_sp2k unique
-- Description: Imports and the transaction/invoice templates reference a project by its
-- No SP2K, so it must identify exactly one project. Case-insensitive, matching how the
-- import resolves it. Fails if existing projects already share a No SP2K.

CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_no_sp2k_unique ON projects (LOWER(no_sp2k));

-- DOWN
-- DROP INDEX IF EXISTS idx_projects_no_sp2k_unique;
