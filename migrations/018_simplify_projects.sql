-- Migration: Simplify projects
-- Description: Management fee, fee rate and manual status are removed from
-- projects (the fee now lives on invoices; status is derived from dates), and
-- tanggal_selesai becomes required because it decides when a contract ends.

UPDATE projects SET tanggal_selesai = tanggal_mulai WHERE tanggal_selesai IS NULL;

ALTER TABLE projects ALTER COLUMN tanggal_selesai SET NOT NULL;

DROP INDEX IF EXISTS idx_projects_status_kontrak;
ALTER TABLE projects
    DROP COLUMN IF EXISTS management_fee,
    DROP COLUMN IF EXISTS tarif_management_fee_persen,
    DROP COLUMN IF EXISTS status_kontrak;

-- DOWN
-- Not reversible: dropped column data is lost.
