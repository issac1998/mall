-- Add deduct_id column to orders table for TCC stock management
-- This column stores the deduction ID from Redis for stock confirmation/cancellation

-- Check if column exists before adding
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = 'seckill' 
  AND TABLE_NAME = 'orders' 
  AND COLUMN_NAME = 'deduct_id';

SET @query = IF(@col_exists = 0,
    'ALTER TABLE orders ADD COLUMN deduct_id VARCHAR(100) DEFAULT NULL COMMENT ''库存扣减ID（TCC）''',
    'SELECT ''Column deduct_id already exists'' AS message');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add index if not exists
SET @index_exists = 0;
SELECT COUNT(*) INTO @index_exists 
FROM INFORMATION_SCHEMA.STATISTICS 
WHERE TABLE_SCHEMA = 'seckill' 
  AND TABLE_NAME = 'orders' 
  AND INDEX_NAME = 'idx_deduct_id';

SET @query = IF(@index_exists = 0,
    'ALTER TABLE orders ADD INDEX idx_deduct_id (deduct_id)',
    'SELECT ''Index idx_deduct_id already exists'' AS message');
PREPARE stmt FROM @query;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

