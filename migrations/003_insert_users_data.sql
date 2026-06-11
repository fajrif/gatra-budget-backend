-- Migration: Insert initial users data
-- Description: Inserts initial user accounts for existing users in the system
-- Note: Passwords are hashed using bcrypt with default password "Password123!"

-- Insert initial users
-- Password for all users: "Password123!"
-- Bcrypt hash: $2a$10$QCG0U7KDYgu88G7K5Y7THe8MIs84KVvuJy1ljoJY7XnK/uAjPmpsW
-- (You should change this after first login)

INSERT INTO users (id, email, password_hash, full_name, role, created_at) VALUES
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'admin@boilerplates.dev', '$2a$10$QCG0U7KDYgu88G7K5Y7THe8MIs84KVvuJy1ljoJY7XnK/uAjPmpsW', 'ADMIN', 'admin', '2006-01-01 00:00:00'),
('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'user@boilerplates.dev', '$2a$10$QCG0U7KDYgu88G7K5Y7THe8MIs84KVvuJy1ljoJY7XnK/uAjPmpsW', 'USER', 'user', '2005-10-17 00:00:00')
ON CONFLICT (id) DO NOTHING;

-- Create index on full_name for searching by user name
CREATE INDEX IF NOT EXISTS idx_users_full_name ON users(full_name);

-- Verify the insert
SELECT id, email, full_name, role, created_at FROM users ORDER BY full_name;
