-- Migration: Create index on users.full_name
-- Description: Speeds up lookups by user name. Schema only — the initial user
--              accounts live in seeds/users_seed.go, not in a migration.

-- UP
-- Create index on full_name for searching by user name
CREATE INDEX IF NOT EXISTS idx_users_full_name ON users(full_name);

-- DOWN
-- DROP INDEX IF EXISTS idx_users_full_name;
