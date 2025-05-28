-- Migration: Create UserAuth and AuthMethod tables for enhanced authentication system
-- Author: Authentication System Implementation
-- Date: 2025-01-28
-- Description: Creates tables for JWT tokens, TOTP 2FA, email verification, and multi-factor authentication

-- First, let's create the migrations table if it doesn't exist
CREATE TABLE IF NOT EXISTS migrations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    migration_name VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create UserAuth table for authentication-related data
CREATE TABLE IF NOT EXISTS UserAuth (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    
    -- Email verification
    is_email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    email_verification_token_expires_at TIMESTAMP NULL,
    
    -- Password reset
    password_reset_token VARCHAR(255),
    password_reset_token_expires_at TIMESTAMP NULL,
    
    -- Account status & security
    last_login_at TIMESTAMP NULL,
    failed_login_attempts INT DEFAULT 0,
    lockout_until TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    -- Foreign key constraint
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Indexes for performance
    INDEX idx_user_auth_user_id (user_id),
    INDEX idx_user_auth_email_verification_token (email_verification_token),
    INDEX idx_user_auth_password_reset_token (password_reset_token),
    INDEX idx_user_auth_last_login (last_login_at),
    INDEX idx_user_auth_lockout (lockout_until)
);

-- Create AuthMethod table for multi-factor authentication methods
CREATE TABLE IF NOT EXISTS AuthMethod (
    id VARCHAR(36) PRIMARY KEY,
    user_auth_id VARCHAR(36) NOT NULL,
    type ENUM('totp', 'email_otp', 'magic_link', 'oauth_google', 'oauth_github', 'fido2') NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    friendly_name VARCHAR(100),
    secret TEXT,
    recovery_codes JSON,
    provider_user_id VARCHAR(255),
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_auth_id) REFERENCES UserAuth(id) ON DELETE CASCADE,
    INDEX idx_auth_method_user_auth_id (user_auth_id),
    INDEX idx_auth_method_type (type),
    INDEX idx_auth_method_enabled (is_enabled),
    INDEX idx_auth_method_verified (is_verified),
    INDEX idx_auth_method_last_used (last_used_at),
    UNIQUE KEY unique_auth_method_per_user (user_auth_id, type)
);

-- Create API Keys table for API authentication
CREATE TABLE IF NOT EXISTS APIKeys (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    last_used_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    permissions JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_api_keys_user_id (user_id),
    INDEX idx_api_keys_hash (key_hash),
    INDEX idx_api_keys_active (is_active),
    INDEX idx_api_keys_expires (expires_at)
);

-- Create Login Attempts table for security monitoring
CREATE TABLE IF NOT EXISTS LoginAttempts (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    attempted_email_or_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_login_attempts_user_id (user_id),
    INDEX idx_login_attempts_ip (ip_address),
    INDEX idx_login_attempts_success (success),
    INDEX idx_login_attempts_created (created_at),
    INDEX idx_login_attempts_email_username (attempted_email_or_username)
);

-- Record this migration as applied
INSERT IGNORE INTO migrations (migration_name) 
VALUES ('002_create_auth_tables');

-- Verify the changes
SHOW TABLES LIKE '%Auth%';
SHOW TABLES LIKE '%API%';
SHOW TABLES LIKE '%Login%';
