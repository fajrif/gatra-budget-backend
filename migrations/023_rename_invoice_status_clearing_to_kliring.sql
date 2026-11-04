-- Migration: Rename invoice status "Clearing" to "Kliring"
-- Description: The paid status was misspelled. Drops the two CHECK constraints
-- that mention it, rewrites existing rows, then re-adds them with 'Kliring'.

-- UP
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_valid;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_tgl_bayar_matches_status;

UPDATE invoices SET status = 'Kliring' WHERE status = 'Clearing';

ALTER TABLE invoices ADD CONSTRAINT invoices_status_valid CHECK (status IN ('Submit', 'Kliring'));
ALTER TABLE invoices ADD CONSTRAINT invoices_tgl_bayar_matches_status CHECK ((status = 'Kliring') = (tgl_bayar IS NOT NULL));

COMMENT ON TABLE invoices IS 'Invoices billed against a project; Kliring invoices count as received payments in their tgl_bayar month';

-- DOWN
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_valid;
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_tgl_bayar_matches_status;
-- UPDATE invoices SET status = 'Clearing' WHERE status = 'Kliring';
-- ALTER TABLE invoices ADD CONSTRAINT invoices_status_valid CHECK (status IN ('Submit', 'Clearing'));
-- ALTER TABLE invoices ADD CONSTRAINT invoices_tgl_bayar_matches_status CHECK ((status = 'Clearing') = (tgl_bayar IS NOT NULL));
