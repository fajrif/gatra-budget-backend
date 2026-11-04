-- Migration: Add payment method and billing period to invoices
-- Description: metode_pembayaran is Kontrak (paid per contract) or SPR (client cash
-- advance). periode_tagihan lists the months (YYYY-MM) the invoice bills for, which can
-- differ from tgl_tagihan/tgl_bayar when billing or payment is late. Kontrak invoices
-- bill exactly one month; SPR invoices may bill several. Existing invoices become
-- Kontrak for the month of their tgl_tagihan.

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS metode_pembayaran VARCHAR(10) NOT NULL DEFAULT 'Kontrak';
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS periode_tagihan VARCHAR(7)[];

UPDATE invoices SET periode_tagihan = ARRAY[to_char(tgl_tagihan, 'YYYY-MM')] WHERE periode_tagihan IS NULL;

ALTER TABLE invoices ALTER COLUMN periode_tagihan SET NOT NULL;

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_metode_pembayaran_valid;
ALTER TABLE invoices ADD CONSTRAINT invoices_metode_pembayaran_valid CHECK (metode_pembayaran IN ('Kontrak', 'SPR'));

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_periode_tagihan_matches_metode;
ALTER TABLE invoices ADD CONSTRAINT invoices_periode_tagihan_matches_metode CHECK (
    COALESCE(cardinality(periode_tagihan), 0) >= 1
    AND (metode_pembayaran <> 'Kontrak' OR cardinality(periode_tagihan) = 1)
);

CREATE INDEX IF NOT EXISTS idx_invoices_periode_tagihan ON invoices USING GIN (periode_tagihan);

COMMENT ON COLUMN invoices.periode_tagihan IS 'Months (YYYY-MM) this invoice bills for; exactly one for Kontrak, one or more for SPR';

-- DOWN
-- DROP INDEX IF EXISTS idx_invoices_periode_tagihan;
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_periode_tagihan_matches_metode;
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_metode_pembayaran_valid;
-- ALTER TABLE invoices DROP COLUMN IF EXISTS periode_tagihan;
-- ALTER TABLE invoices DROP COLUMN IF EXISTS metode_pembayaran;
