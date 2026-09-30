-- Migration: Add photo_url to users
-- Description: Users.PhotoUrl exists in the model and is written by the avatar
--              upload handler, but no migration ever created the column, so
--              every upload failed with a "column does not exist" error.

-- UP
ALTER TABLE users ADD COLUMN IF NOT EXISTS photo_url VARCHAR(255);

COMMENT ON COLUMN users.photo_url IS 'URL of the user avatar image';

-- DOWN
-- ALTER TABLE users DROP COLUMN IF EXISTS photo_url;
