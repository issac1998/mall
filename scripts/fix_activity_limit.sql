-- Fix activity limit_per_user field
-- This script fixes the limit_per_user field for activity ID 1

USE seckill_dev;

-- Check current value
SELECT id, name, limit_per_user, stock, sold, status 
FROM seckill_activities 
WHERE id = 1;

-- Update limit_per_user to 1
UPDATE seckill_activities 
SET limit_per_user = 1 
WHERE id = 1;

-- Verify the update
SELECT id, name, limit_per_user, stock, sold, status 
FROM seckill_activities 
WHERE id = 1;