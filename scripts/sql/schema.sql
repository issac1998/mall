-- ========================================
-- Seckill System Database Initialization Script
-- ========================================

-- Create database
CREATE DATABASE IF NOT EXISTS `seckill_dev` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `seckill_dev`;

-- ========================================
-- 1. Users table
-- ========================================
CREATE TABLE `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'User ID',
  `username` VARCHAR(50) NOT NULL COMMENT 'Username',
  `phone` VARCHAR(20) NOT NULL COMMENT 'Phone number',
  `email` VARCHAR(100) DEFAULT NULL COMMENT 'Email',
  `password_hash` VARCHAR(255) NOT NULL COMMENT 'Password hash',
  `salt` VARCHAR(32) NOT NULL COMMENT 'Password salt',
  `nickname` VARCHAR(50) DEFAULT NULL COMMENT 'Nickname',
  `avatar` VARCHAR(255) DEFAULT NULL COMMENT 'Avatar URL',
  `gender` TINYINT DEFAULT 0 COMMENT 'Gender: 0-unknown, 1-male, 2-female',
  `birthday` DATE DEFAULT NULL COMMENT 'Birthday',
  `level` INT DEFAULT 1 COMMENT 'User level',
  `points` INT DEFAULT 0 COMMENT 'Points',
  `balance` BIGINT DEFAULT 0 COMMENT 'Balance (cents)',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'Status: 1-active, 2-disabled, 3-deleted',
  `last_login_at` TIMESTAMP NULL DEFAULT NULL COMMENT 'Last login time',
  `last_login_ip` VARCHAR(45) DEFAULT NULL COMMENT 'Last login IP',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  UNIQUE KEY `uk_phone` (`phone`),
  UNIQUE KEY `uk_email` (`email`),
  KEY `idx_status` (`status`),
  KEY `idx_level` (`level`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Users table';

-- ========================================
-- 2. Goods table
-- ========================================
CREATE TABLE `goods` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Goods ID',
  `name` VARCHAR(200) NOT NULL COMMENT 'Goods name',
  `description` TEXT COMMENT 'Goods description',
  `category` VARCHAR(50) DEFAULT NULL COMMENT 'Goods category',
  `brand` VARCHAR(50) DEFAULT NULL COMMENT 'Brand',
  `images` JSON DEFAULT NULL COMMENT 'Goods images (JSON array)',
  `price` BIGINT NOT NULL COMMENT 'Original price (cents)',
  `stock` INT NOT NULL DEFAULT 0 COMMENT 'Stock quantity',
  `sales` INT NOT NULL DEFAULT 0 COMMENT 'Sales volume',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'Status: 1-on shelf, 2-off shelf, 3-deleted',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  KEY `idx_category` (`category`),
  KEY `idx_brand` (`brand`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Goods table';

-- ========================================
-- 3. Seckill activities table
-- ========================================
CREATE TABLE `seckill_activities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Activity ID',
  `name` VARCHAR(200) NOT NULL COMMENT 'Activity name',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT 'Goods ID',
  `price` BIGINT NOT NULL COMMENT 'Seckill price (cents)',
  `stock` INT NOT NULL COMMENT 'Seckill stock',
  `sold` INT NOT NULL DEFAULT 0 COMMENT 'Sold quantity',
  `start_time` TIMESTAMP NOT NULL COMMENT 'Start time',
  `end_time` TIMESTAMP NOT NULL COMMENT 'End time',
  `limit_per_user` INT NOT NULL DEFAULT 1 COMMENT 'Purchase limit per user',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT 'Status: 0-not started, 1-in progress, 2-ended, 3-paused, 4-cancelled',
  
  -- Industrial-grade extension fields
  `prewarm_time` TIMESTAMP NULL DEFAULT NULL COMMENT 'Prewarm time',
  `prewarm_status` TINYINT DEFAULT 0 COMMENT 'Prewarm status: 0-not prewarmed, 1-prewarmed',
  `priority` INT DEFAULT 0 COMMENT 'Priority (affects resource allocation)',
  `risk_level` TINYINT DEFAULT 1 COMMENT 'Risk level: 1-5',
  `max_qps` INT DEFAULT 10000 COMMENT 'Maximum QPS limit',
  `max_concurrent` INT DEFAULT 5000 COMMENT 'Maximum concurrent connections',
  `shard_count` INT DEFAULT 1 COMMENT 'Stock shard count',
  `shard_strategy` VARCHAR(20) DEFAULT 'hash' COMMENT 'Shard strategy',
  `degrade_threshold` DECIMAL(5,4) DEFAULT 0.5 COMMENT 'Degrade threshold',
  `degrade_strategy` VARCHAR(50) DEFAULT 'queue' COMMENT 'Degrade strategy',
  `gray_strategy` VARCHAR(50) DEFAULT NULL COMMENT 'Gray release strategy',
  `gray_ratio` DECIMAL(5,4) DEFAULT 0 COMMENT 'Gray release ratio',
  `gray_whitelist` JSON DEFAULT NULL COMMENT 'Gray release whitelist',
  `ext_config` JSON DEFAULT NULL COMMENT 'Extension configuration',
  
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  KEY `idx_goods_id` (`goods_id`),
  KEY `idx_start_time` (`start_time`),
  KEY `idx_end_time` (`end_time`),
  KEY `idx_status` (`status`),
  KEY `idx_priority` (`priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Seckill activities table';

-- ========================================
-- 4. Activity rules table
-- ========================================
CREATE TABLE `activity_rules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Rule ID',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT 'Activity ID',
  `rule_type` VARCHAR(50) NOT NULL COMMENT 'Rule type: time/region/user_level/whitelist',
  `rule_name` VARCHAR(100) NOT NULL COMMENT 'Rule name',
  `rule_content` JSON NOT NULL COMMENT 'Rule content (JSON format)',
  `priority` INT DEFAULT 0 COMMENT 'Priority',
  `enabled` TINYINT NOT NULL DEFAULT 1 COMMENT 'Enabled: 0-no, 1-yes',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  KEY `idx_activity_id` (`activity_id`),
  KEY `idx_rule_type` (`rule_type`),
  KEY `idx_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Activity rules table';

-- ========================================
-- 5. Orders table (supports table partitioning)
-- ========================================
CREATE TABLE `orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Order ID',
  `order_no` VARCHAR(32) NOT NULL COMMENT 'Order number',
  `request_id` VARCHAR(32) NOT NULL COMMENT 'Request ID (idempotent)',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT 'User ID',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT 'Activity ID',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT 'Goods ID',
  `quantity` INT NOT NULL COMMENT 'Purchase quantity',
  `price` BIGINT NOT NULL COMMENT 'Unit price (cents)',
  `total_amount` BIGINT NOT NULL COMMENT 'Total amount (cents)',
  `discount_amount` BIGINT DEFAULT 0 COMMENT 'Discount amount (cents)',
  `payment_amount` BIGINT NOT NULL COMMENT 'Payment amount (cents)',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'Status: 1-pending payment, 2-paid, 3-cancelled, 4-refunded, 5-completed',
  `payment_method` VARCHAR(20) DEFAULT NULL COMMENT 'Payment method',
  `payment_no` VARCHAR(64) DEFAULT NULL COMMENT 'Payment transaction number',
  `paid_at` TIMESTAMP NULL DEFAULT NULL COMMENT 'Payment time',
  `expire_at` TIMESTAMP NOT NULL COMMENT 'Expiration time',
  `cancel_reason` VARCHAR(255) DEFAULT NULL COMMENT 'Cancellation reason',
  `remark` VARCHAR(500) DEFAULT NULL COMMENT 'Remark',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_no` (`order_no`),
  UNIQUE KEY `uk_request_id` (`request_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_activity_id` (`activity_id`),
  KEY `idx_status` (`status`),
  KEY `idx_expire_at` (`expire_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Orders table';

-- ========================================
-- 6. Order details table
-- ========================================
CREATE TABLE `order_details` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Detail ID',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT 'Order ID',
  `order_no` VARCHAR(32) NOT NULL COMMENT 'Order number',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT 'Goods ID',
  `goods_name` VARCHAR(200) NOT NULL COMMENT 'Goods name',
  `goods_image` VARCHAR(255) DEFAULT NULL COMMENT 'Goods image',
  `price` BIGINT NOT NULL COMMENT 'Unit price (cents)',
  `quantity` INT NOT NULL COMMENT 'Quantity',
  `amount` BIGINT NOT NULL COMMENT 'Subtotal (cents)',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_order_no` (`order_no`),
  KEY `idx_goods_id` (`goods_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Order details table';

-- ========================================
-- 7. Stock logs table (for reconciliation)
-- ========================================
CREATE TABLE `stock_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Log ID',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT 'Activity ID',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT 'Goods ID',
  `operation_type` TINYINT NOT NULL COMMENT 'Operation type: 1-deduct, 2-replenish, 3-sync',
  `quantity` INT NOT NULL COMMENT 'Quantity (positive for increase, negative for decrease)',
  `before_stock` INT NOT NULL COMMENT 'Stock before operation',
  `after_stock` INT NOT NULL COMMENT 'Stock after operation',
  `request_id` VARCHAR(32) DEFAULT NULL COMMENT 'Request ID',
  `order_no` VARCHAR(32) DEFAULT NULL COMMENT 'Order number',
  `operator` VARCHAR(50) DEFAULT NULL COMMENT 'Operator',
  `remark` VARCHAR(255) DEFAULT NULL COMMENT 'Remark',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  PRIMARY KEY (`id`),
  KEY `idx_activity_id` (`activity_id`),
  KEY `idx_goods_id` (`goods_id`),
  KEY `idx_request_id` (`request_id`),
  KEY `idx_order_no` (`order_no`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Stock logs table';

-- ========================================
-- 8. User addresses table
-- ========================================
CREATE TABLE `user_addresses` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Address ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT 'User ID',
  `receiver_name` VARCHAR(50) NOT NULL COMMENT 'Receiver name',
  `receiver_phone` VARCHAR(20) NOT NULL COMMENT 'Receiver phone',
  `province` VARCHAR(50) NOT NULL COMMENT 'Province',
  `city` VARCHAR(50) NOT NULL COMMENT 'City',
  `district` VARCHAR(50) NOT NULL COMMENT 'District',
  `address` VARCHAR(255) NOT NULL COMMENT 'Detailed address',
  `postcode` VARCHAR(10) DEFAULT NULL COMMENT 'Postcode',
  `is_default` TINYINT DEFAULT 0 COMMENT 'Is default: 0-no, 1-yes',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_is_default` (`is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='User addresses table';

-- ========================================
-- 9. Risk records table
-- ========================================
CREATE TABLE `risk_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Record ID',
  `user_id` BIGINT UNSIGNED DEFAULT NULL COMMENT 'User ID',
  `ip` VARCHAR(45) DEFAULT NULL COMMENT 'IP address',
  `device_id` VARCHAR(64) DEFAULT NULL COMMENT 'Device ID',
  `request_id` VARCHAR(32) DEFAULT NULL COMMENT 'Request ID',
  `activity_id` BIGINT UNSIGNED DEFAULT NULL COMMENT 'Activity ID',
  `risk_type` VARCHAR(50) NOT NULL COMMENT 'Risk type',
  `risk_level` TINYINT NOT NULL COMMENT 'Risk level: 1-low, 2-medium, 3-high, 4-critical',
  `risk_score` INT DEFAULT 0 COMMENT 'Risk score',
  `hit_rules` JSON DEFAULT NULL COMMENT 'Hit rules',
  `action` VARCHAR(20) NOT NULL COMMENT 'Action: pass/warn/block',
  `remark` VARCHAR(500) DEFAULT NULL COMMENT 'Remark',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_ip` (`ip`),
  KEY `idx_device_id` (`device_id`),
  KEY `idx_activity_id` (`activity_id`),
  KEY `idx_risk_level` (`risk_level`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Risk records table';

-- ========================================
-- 10. Blacklist table
-- ========================================
CREATE TABLE `blacklist` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Blacklist ID',
  `type` TINYINT NOT NULL COMMENT 'Type: 1-user, 2-IP, 3-device',
  `value` VARCHAR(255) NOT NULL COMMENT 'Value',
  `reason` VARCHAR(255) DEFAULT NULL COMMENT 'Reason',
  `expire_at` TIMESTAMP NULL DEFAULT NULL COMMENT 'Expiration time (NULL for permanent)',
  `created_by` VARCHAR(50) DEFAULT NULL COMMENT 'Created by',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_type_value` (`type`, `value`),
  KEY `idx_expire_at` (`expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Blacklist table';

-- ========================================
-- Create views (optional)
-- ========================================

-- Activity statistics view
CREATE OR REPLACE VIEW `v_activity_stats` AS
SELECT 
    a.id,
    a.name,
    a.stock AS total_stock,
    a.sold,
    (a.stock - a.sold) AS remaining_stock,
    ROUND(a.sold / a.stock * 100, 2) AS sell_rate,
    a.status,
    a.start_time,
    a.end_time
FROM `seckill_activities` a;

-- User order statistics view
CREATE OR REPLACE VIEW `v_user_order_stats` AS
SELECT 
    u.id AS user_id,
    u.username,
    COUNT(o.id) AS order_count,
    SUM(o.payment_amount) AS total_amount,
    MAX(o.created_at) AS last_order_time
FROM `users` u
LEFT JOIN `orders` o ON u.id = o.user_id
GROUP BY u.id, u.username;