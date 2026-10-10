-- Migration: Create budget_items table
-- Description: Budget line items ("Anggaran") per project/cost type. A
-- "monthly" budget item is one parent row plus N evenly-split monthly child
-- rows; a "lumpsum" budget item is a single parent row with no children.

CREATE TABLE IF NOT EXISTS budget_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    cost_type_id UUID NOT NULL REFERENCES cost_types(id),
    jenis_biaya_name VARCHAR(255),
    kategori_anggaran VARCHAR(20) CHECK (kategori_anggaran IS NULL OR kategori_anggaran IN ('monthly', 'lumpsum')),
    total_anggaran NUMERIC(20,2),
    deskripsi_anggaran VARCHAR(500),
    periode_bulan VARCHAR(7),
    jumlah_anggaran NUMERIC(20,2) NOT NULL,
    bulan_ke INTEGER,
    parent_budget_id UUID REFERENCES budget_items(id) ON DELETE CASCADE,
    is_parent BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT budget_items_parent_presence_matches_is_parent
        CHECK ((is_parent AND parent_budget_id IS NULL) OR (NOT is_parent AND parent_budget_id IS NOT NULL)),
    CONSTRAINT budget_items_jumlah_within_total
        CHECK (total_anggaran IS NULL OR jumlah_anggaran <= total_anggaran)
);

CREATE INDEX IF NOT EXISTS idx_budget_items_project_id ON budget_items(project_id);
CREATE INDEX IF NOT EXISTS idx_budget_items_cost_type_id ON budget_items(cost_type_id);
CREATE INDEX IF NOT EXISTS idx_budget_items_parent_budget_id ON budget_items(parent_budget_id);

COMMENT ON TABLE budget_items IS 'Budget line items per project/cost type; monthly items split into child rows via the BudgetItemSplitter service';

-- DOWN
-- DROP TABLE IF EXISTS budget_items;
