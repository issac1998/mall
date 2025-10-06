-- ========================================
-- Seckill system test data initialization script
-- ========================================

USE `seckill_dev`;

-- ========================================
-- 1. Insert test user data
-- ========================================
INSERT INTO `users` (`username`, `phone`, `email`, `password_hash`, `salt`, `nickname`, `gender`, `level`, `points`, `balance`, `status`) VALUES
('testuser1', '13800138001', 'test1@example.com', 'hash1', 'salt1', 'Test User 1', 1, 1, 100, 100000, 1),
('testuser2', '13800138002', 'test2@example.com', 'hash2', 'salt2', 'Test User 2', 2, 2, 200, 200000, 1),
('testuser3', '13800138003', 'test3@example.com', 'hash3', 'salt3', 'Test User 3', 1, 3, 300, 300000, 1),
('testuser4', '13800138004', 'test4@example.com', 'hash4', 'salt4', 'Test User 4', 2, 1, 50, 50000, 1),
('testuser5', '13800138005', 'test5@example.com', 'hash5', 'salt5', 'Test User 5', 1, 2, 150, 150000, 1);

-- ========================================
-- 2. Insert test product data
-- ========================================
INSERT INTO `goods` (`name`, `description`, `category`, `brand`, `images`, `price`, `stock`, `sales`, `status`) VALUES
('iPhone 15 Pro Max', 'Apple latest flagship phone, 256GB storage', 'Mobile & Digital', 'Apple', '["https://example.com/iphone1.jpg", "https://example.com/iphone2.jpg"]', 999900, 1000, 0, 1),
('Huawei Mate60 Pro', 'Huawei flagship phone, 512GB storage', 'Mobile & Digital', 'Huawei', '["https://example.com/huawei1.jpg", "https://example.com/huawei2.jpg"]', 699900, 800, 0, 1),
('Xiaomi 14 Ultra', 'Xiaomi imaging flagship, 1TB storage', 'Mobile & Digital', 'Xiaomi', '["https://example.com/xiaomi1.jpg", "https://example.com/xiaomi2.jpg"]', 599900, 600, 0, 1),
('MacBook Pro M3', 'Apple laptop, 16GB+512GB', 'Computer & Office', 'Apple', '["https://example.com/macbook1.jpg", "https://example.com/macbook2.jpg"]', 1999900, 200, 0, 1),
('Dyson V15 Vacuum', 'Cordless vacuum cleaner with laser detection', 'Home Appliances', 'Dyson', '["https://example.com/dyson1.jpg", "https://example.com/dyson2.jpg"]', 399900, 300, 0, 1);

-- ========================================
-- 3. Insert test seckill activity data
-- ========================================
INSERT INTO `seckill_activities` (
    `name`, `goods_id`, `price`, `stock`, `start_time`, `end_time`, 
    `limit_per_user`, `status`, `prewarm_time`, `priority`, `risk_level`, 
    `max_qps`, `max_concurrent`, `shard_count`
) VALUES
(
    'iPhone 15 Pro Max Seckill', 1, 899900, 100, 
    DATE_ADD(NOW(), INTERVAL 1 HOUR), DATE_ADD(NOW(), INTERVAL 25 HOUR),
    1, 0, DATE_ADD(NOW(), INTERVAL 30 MINUTE), 5, 3, 
    50000, 10000, 4
),
(
    'Huawei Mate60 Pro Flash Sale', 2, 599900, 50,
    DATE_ADD(NOW(), INTERVAL 2 HOUR), DATE_ADD(NOW(), INTERVAL 26 HOUR),
    2, 0, DATE_ADD(NOW(), INTERVAL 1 HOUR), 4, 2,
    30000, 8000, 2
),
(
    'Xiaomi 14 Ultra Flash Sale', 3, 499900, 80,
    DATE_ADD(NOW(), INTERVAL 3 HOUR), DATE_ADD(NOW(), INTERVAL 27 HOUR),
    1, 0, DATE_ADD(NOW(), INTERVAL 2 HOUR), 3, 2,
    20000, 5000, 2
),
(
    'MacBook Pro M3 Education Discount', 4, 1799900, 30,
    DATE_ADD(NOW(), INTERVAL 4 HOUR), DATE_ADD(NOW(), INTERVAL 28 HOUR),
    1, 0, DATE_ADD(NOW(), INTERVAL 3 HOUR), 2, 1,
    10000, 2000, 1
),
(
    'Dyson V15 Home Appliance Festival', 5, 299900, 60,
    DATE_ADD(NOW(), INTERVAL 5 HOUR), DATE_ADD(NOW(), INTERVAL 29 HOUR),
    3, 0, DATE_ADD(NOW(), INTERVAL 4 HOUR), 1, 1,
    15000, 3000, 1
);

-- ========================================
-- 4. Insert activity rule data
-- ========================================
INSERT INTO `activity_rules` (`activity_id`, `rule_type`, `rule_name`, `rule_content`, `priority`, `enabled`) VALUES
(1, 'user_level', 'User Level Restriction', '{"min_level": 1, "max_level": 5}', 1, 1),
(1, 'region', 'Region Restriction', '{"allowed_regions": ["Beijing", "Shanghai", "Guangzhou", "Shenzhen"]}', 2, 1),
(2, 'user_level', 'Member Exclusive', '{"min_level": 2}', 1, 1),
(3, 'whitelist', 'Beta Users', '{"user_ids": [1, 2, 3]}', 1, 0),
(4, 'user_level', 'Student Verification', '{"min_level": 1, "require_student": true}', 1, 1),
(5, 'time', 'Time Period Restriction', '{"allowed_hours": [9, 10, 11, 14, 15, 16, 20, 21, 22]}', 1, 1);

-- ========================================
-- 5. Insert user address data
-- ========================================
INSERT INTO `user_addresses` (`user_id`, `receiver_name`, `receiver_phone`, `province`, `city`, `district`, `address`, `is_default`) VALUES
(1, 'Zhang San', '13800138001', 'Beijing', 'Beijing', 'Chaoyang District', 'Sanlitun Street No.1', 1),
(1, 'Zhang San', '13800138001', 'Shanghai', 'Shanghai', 'Pudong New Area', 'Lujiazui Financial District No.2', 0),
(2, 'Li Si', '13800138002', 'Guangdong Province', 'Shenzhen', 'Nanshan District', 'Science Park South No.3', 1),
(3, 'Wang Wu', '13800138003', 'Zhejiang Province', 'Hangzhou', 'Xihu District', 'Wensan Road No.4', 1),
(4, 'Zhao Liu', '13800138004', 'Jiangsu Province', 'Nanjing', 'Gulou District', 'Zhongshan Road No.5', 1),
(5, 'Qian Qi', '13800138005', 'Sichuan Province', 'Chengdu', 'Jinjiang District', 'Chunxi Road No.6', 1);

-- ========================================
-- 6. Insert blacklist test data
-- ========================================
INSERT INTO `blacklist` (`type`, `value`, `reason`, `expire_at`, `created_by`) VALUES
(1, '999999', 'Malicious order brushing', DATE_ADD(NOW(), INTERVAL 30 DAY), 'system'),
(2, '192.168.1.100', 'Abnormal access', DATE_ADD(NOW(), INTERVAL 7 DAY), 'admin'),
(3, 'device_12345', 'Emulator detection', NULL, 'system');

-- ========================================
-- 7. Insert risk record test data
-- ========================================
INSERT INTO `risk_records` (`user_id`, `ip`, `device_id`, `activity_id`, `risk_type`, `risk_level`, `risk_score`, `hit_rules`, `action`, `remark`) VALUES
(1, '192.168.1.1', 'device_001', 1, 'frequency_limit', 2, 60, '["rule_freq_1min"]', 'warn', 'Too frequent requests within 1 minute'),
(2, '192.168.1.2', 'device_002', 1, 'ip_risk', 1, 30, '["rule_ip_geo"]', 'pass', 'IP geolocation check passed'),
(3, '192.168.1.3', 'device_003', 2, 'device_risk', 3, 80, '["rule_device_sim"]', 'block', 'Emulator characteristics detected');

-- ========================================
-- 8. Create indexes for optimization (if needed)
-- ========================================

-- Create composite indexes for high-frequency queries
CREATE INDEX `idx_activities_status_time` ON `seckill_activities` (`status`, `start_time`, `end_time`);
CREATE INDEX `idx_orders_user_status` ON `orders` (`user_id`, `status`, `created_at`);
CREATE INDEX `idx_stock_logs_activity_time` ON `stock_logs` (`activity_id`, `created_at`);

-- ========================================
-- 9. Set auto-increment starting values
-- ========================================
ALTER TABLE `users` AUTO_INCREMENT = 10001;
ALTER TABLE `goods` AUTO_INCREMENT = 10001;
ALTER TABLE `seckill_activities` AUTO_INCREMENT = 10001;
ALTER TABLE `orders` AUTO_INCREMENT = 100001;
ALTER TABLE `order_details` AUTO_INCREMENT = 100001;
ALTER TABLE `stock_logs` AUTO_INCREMENT = 100001;