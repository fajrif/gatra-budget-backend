-- Migration: Drop budget_items table
-- Description: Budget planning (Anggaran) is no longer part of the app; only
-- Project, Transaction and Invoice remain as core resources.

DROP TABLE IF EXISTS budget_items;

-- DOWN
-- Not reversible: recreate the table from 014_create_budget_items_table.sql.
