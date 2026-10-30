-- Migration: Simplify transactions
-- Description: PO/invoice date, evidence upload and per-transaction management
-- fee are removed; transactions only record project expenses.

ALTER TABLE transactions
    DROP COLUMN IF EXISTS tanggal_po_tagihan,
    DROP COLUMN IF EXISTS bukti_transaksi_url,
    DROP COLUMN IF EXISTS persentase_management_fee,
    DROP COLUMN IF EXISTS nilai_management_fee;

-- DOWN
-- Not reversible: dropped column data is lost.
