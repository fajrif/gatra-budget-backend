-- Migration: Create database with owner deployer
-- Description: Creates a new database with deployer as the owner
-- Note: This should be run by a superuser (e.g., postgres)

-- Create database (if not exists)
-- You may need to disconnect from the current database first
-- or run this from a different database like 'postgres'

-- CREATE DATABASE gatra_budget_db
    -- WITH
    -- OWNER = deployer
    -- ENCODING = 'UTF8'
    -- LC_COLLATE = 'en_US.UTF-8'
    -- LC_CTYPE = 'en_US.UTF-8'
    -- TABLESPACE = pg_default
    -- CONNECTION LIMIT = -1;

-- Grant all privileges to deployer
-- GRANT ALL PRIVILEGES ON DATABASE gatra_budget_db TO deployer;

-- Optional: Create deployer user if not exists
-- Uncomment if needed
-- CREATE USER deployer WITH PASSWORD 'deployer1234!';
-- ALTER USER deployer CREATEDB;

-- Ensure UUID extension is enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

