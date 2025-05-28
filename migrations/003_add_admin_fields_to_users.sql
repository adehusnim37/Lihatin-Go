-- Migration: Add admin fields to users table for role-based access control
-- Author: Admin Management System
-- Date: 2025-05-28
-- Description: Adds user locking, role management, and admin features to the users table

-- First, let's create the migrations table if it doesn't exist
CREATE TABLE IF NOT EXISTS migrations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    migration_name VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add admin-related columns to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS is_locked BOOLEAN DEFAULT FALSE COMMENT 'Whether the user account is locked',
ADD COLUMN IF NOT EXISTS locked_at TIMESTAMP NULL COMMENT 'When the account was locked',
ADD COLUMN IF NOT EXISTS locked_reason VARCHAR(500) COMMENT 'Reason for account lock',
ADD COLUMN IF NOT EXISTS role ENUM('user', 'admin', 'super_admin') DEFAULT 'user' COMMENT 'User role for access control';

-- Add indexes for performance
ALTER TABLE users 
ADD INDEX IF NOT EXISTS idx_users_is_locked (is_locked),
ADD INDEX IF NOT EXISTS idx_users_role (role),
ADD INDEX IF NOT EXISTS idx_users_locked_at (locked_at);

-- Create a view for admin user management (optional but helpful)
CREATE OR REPLACE VIEW admin_users_view AS
SELECT 
    u.id,
    u.username,
    u.email,
    u.role,
    u.is_locked,
    u.locked_at,
    u.locked_reason,
    u.created_at,
    u.updated_at,
    ua.is_email_verified,
    ua.last_login_at,
    ua.failed_login_attempts,
    ua.lockout_until,
    ua.is_active,
    (SELECT COUNT(*) FROM APIKeys ak WHERE ak.user_id = u.id AND ak.deleted_at IS NULL) as api_key_count,
    (SELECT COUNT(*) FROM LoginAttempts la WHERE la.user_id = u.id AND la.success = FALSE AND la.created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR)) as failed_attempts_today
FROM users u
LEFT JOIN UserAuth ua ON u.id = ua.user_id
WHERE u.deleted_at IS NULL;

-- Insert default super admin user if it doesn't exist (optional - remove if not needed)
-- This is useful for initial setup
-- Note: Replace with actual secure password hash in production
INSERT IGNORE INTO users (id, username, email, role, created_at, updated_at)
VALUES (
    UUID(),
    'superadmin',
    'admin@example.com',
    'super_admin',
    NOW(),
    NOW()
);

-- Record this migration as applied
INSERT IGNORE INTO migrations (migration_name) 
VALUES ('003_add_admin_fields_to_users');

-- Verify the changes
DESCRIBE users;
SELECT COUNT(*) as admin_users FROM users WHERE role IN ('admin', 'super_admin');
