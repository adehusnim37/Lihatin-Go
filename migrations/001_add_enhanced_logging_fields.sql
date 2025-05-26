-- Migration: Add enhanced logging fields to ActivityLog table
-- Author: Enhanced Activity Logger System
-- Date: 2025-05-26
-- Description: Adds request body, query params, route params, context locals, and response time fields

-- First, let's create the migrations table if it doesn't exist
CREATE TABLE IF NOT EXISTS migrations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    migration_name VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add new columns to ActivityLog table
ALTER TABLE ActivityLog 
ADD COLUMN IF NOT EXISTS requestbody TEXT COMMENT 'Request body content (sanitized)',
ADD COLUMN IF NOT EXISTS queryparams TEXT COMMENT 'Query parameters as JSON',
ADD COLUMN IF NOT EXISTS routeparams TEXT COMMENT 'Route parameters as JSON',
ADD COLUMN IF NOT EXISTS contextlocals TEXT COMMENT 'Context values as JSON',
ADD COLUMN IF NOT EXISTS responsetime BIGINT COMMENT 'Response time in milliseconds';

-- Record this migration as applied
INSERT IGNORE INTO migrations (migration_name) 
VALUES ('001_add_enhanced_logging_fields');

-- Verify the changes
DESCRIBE ActivityLog;
