-- Migration: Add received amount and invoice deductions
-- Description: nilai stays the billed amount (Nilai Tagihan). nilai_penerimaan is the
-- amount the client actually paid, filled when an invoice is cleared (Kliring). The
-- gap between the two is explained by rows in invoice_deductions (potongan), each
-- pointing at a deduction_types master row. Existing paid invoices receive no
-- deduction, so their nilai_penerimaan equals nilai. The rule that
-- nilai_penerimaan + SUM(potongan) = nilai spans rows, so the handler enforces it.

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS nilai_penerimaan NUMERIC(20,2);

UPDATE invoices SET nilai_penerimaan = nilai WHERE status = 'Kliring' AND nilai_penerimaan IS NULL;

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_nilai_penerimaan_matches_status;
ALTER TABLE invoices ADD CONSTRAINT invoices_nilai_penerimaan_matches_status CHECK (
    (status = 'Kliring') = (nilai_penerimaan IS NOT NULL)
);

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_nilai_penerimaan_non_negative;
ALTER TABLE invoices ADD CONSTRAINT invoices_nilai_penerimaan_non_negative CHECK (nilai_penerimaan >= 0);

CREATE TABLE IF NOT EXISTS invoice_deductions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    deduction_type_id UUID NOT NULL REFERENCES deduction_types(id) ON DELETE RESTRICT,
    jumlah NUMERIC(20,2) NOT NULL,
    keterangan TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT invoice_deductions_jumlah_positive CHECK (jumlah > 0)
);

CREATE INDEX IF NOT EXISTS idx_invoice_deductions_invoice_id ON invoice_deductions(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_deductions_deduction_type_id ON invoice_deductions(deduction_type_id);

COMMENT ON COLUMN invoices.nilai_penerimaan IS 'Amount actually received when the invoice is cleared (Kliring); NULL while Submit';
COMMENT ON TABLE invoice_deductions IS 'Deductions (potongan) that explain the gap between an invoice nilai and its nilai_penerimaan';

-- DOWN
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_nilai_penerimaan_non_negative;
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_nilai_penerimaan_matches_status;
-- DROP TABLE IF EXISTS invoice_deductions;
-- ALTER TABLE invoices DROP COLUMN IF EXISTS nilai_penerimaan;
