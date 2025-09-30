-- Seckill system database initialization script

-- Create database
CREATE DATABASE IF NOT EXISTS seckill DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE seckill;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'User ID',
    username VARCHAR(50) NOT NULL COMMENT 'Username',
    email VARCHAR(100) NOT NULL COMMENT 'Email',
    password_hash VARCHAR(255) NOT NULL COMMENT 'Password hash',
    phone VARCHAR(20) DEFAULT NULL COMMENT 'Phone number',
    status TINYINT NOT NULL DEFAULT 1 COMMENT 'Status: 0-disabled, 1-enabled',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    PRIMARY KEY (id),
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email),
    KEY idx_phone (phone),
    KEY idx_status (status),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Users table';

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Product ID',
    name VARCHAR(255) NOT NULL COMMENT 'Product name',
    description TEXT COMMENT 'Product description',
    price DECIMAL(10,2) NOT NULL COMMENT 'Product price',
    stock INT NOT NULL DEFAULT 0 COMMENT 'Stock quantity',
    status TINYINT NOT NULL DEFAULT 1 COMMENT 'Status: 0-offline, 1-online',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    PRIMARY KEY (id),
    KEY idx_name (name),
    KEY idx_status (status),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Products table';

-- Seckill activities table
CREATE TABLE IF NOT EXISTS seckill_activities (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Activity ID',
    product_id BIGINT UNSIGNED NOT NULL COMMENT 'Product ID',
    name VARCHAR(255) NOT NULL COMMENT 'Activity name',
    seckill_price DECIMAL(10,2) NOT NULL COMMENT 'Seckill price',
    seckill_stock INT NOT NULL COMMENT 'Seckill stock',
    start_time TIMESTAMP NOT NULL COMMENT 'Start time',
    end_time TIMESTAMP NOT NULL COMMENT 'End time',
    status TINYINT NOT NULL DEFAULT 0 COMMENT 'Status: 0-not started, 1-in progress, 2-ended',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    PRIMARY KEY (id),
    KEY idx_product_id (product_id),
    KEY idx_start_time (start_time),
    KEY idx_end_time (end_time),
    KEY idx_status (status),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Seckill activities table';

-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Order ID',
    order_no VARCHAR(64) NOT NULL COMMENT 'Order number',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'User ID',
    product_id BIGINT UNSIGNED NOT NULL COMMENT 'Product ID',
    seckill_id BIGINT UNSIGNED DEFAULT NULL COMMENT 'Seckill activity ID',
    quantity INT NOT NULL DEFAULT 1 COMMENT 'Purchase quantity',
    price DECIMAL(10,2) NOT NULL COMMENT 'Unit price',
    total_amount DECIMAL(10,2) NOT NULL COMMENT 'Total amount',
    status TINYINT NOT NULL DEFAULT 0 COMMENT 'Status: 0-pending payment, 1-paid, 2-cancelled, 3-refunded',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
    PRIMARY KEY (id),
    UNIQUE KEY uk_order_no (order_no),
    KEY idx_user_id (user_id),
    KEY idx_product_id (product_id),
    KEY idx_seckill_id (seckill_id),
    KEY idx_status (status),
    KEY idx_created_at (created_at),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (seckill_id) REFERENCES seckill_activities(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Orders table';

-- Stock records table
CREATE TABLE IF NOT EXISTS stock_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Record ID',
    product_id BIGINT UNSIGNED NOT NULL COMMENT 'Product ID',
    seckill_id BIGINT UNSIGNED DEFAULT NULL COMMENT 'Seckill activity ID',
    change_type TINYINT NOT NULL COMMENT 'Change type: 1-increase, 2-decrease',
    change_quantity INT NOT NULL COMMENT 'Change quantity',
    before_stock INT NOT NULL COMMENT 'Stock before change',
    after_stock INT NOT NULL COMMENT 'Stock after change',
    reason VARCHAR(255) DEFAULT NULL COMMENT 'Change reason',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
    PRIMARY KEY (id),
    KEY idx_product_id (product_id),
    KEY idx_seckill_id (seckill_id),
    KEY idx_created_at (created_at),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (seckill_id) REFERENCES seckill_activities(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Stock records table';

-- Insert test data
INSERT INTO users (username, email, password_hash, phone) VALUES
('admin', 'admin@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsxq/3/7.', '13800138000'),
('test1', 'test1@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsxq/3/7.', '13800138001'),
('test2', 'test2@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsxq/3/7.', '13800138002');

INSERT INTO products (name, description, price, stock) VALUES
('iPhone 15 Pro', 'Apple iPhone 15 Pro 256GB', 8999.00, 100),
('MacBook Pro', 'Apple MacBook Pro 14-inch M3', 14999.00, 50),
('AirPods Pro', 'Apple AirPods Pro (2nd generation)', 1899.00, 200);

INSERT INTO seckill_activities (product_id, name, seckill_price, seckill_stock, start_time, end_time, status) VALUES
(1, 'iPhone 15 Pro Seckill', 7999.00, 10, '2024-01-01 10:00:00', '2024-12-31 23:59:59', 1),
(2, 'MacBook Pro Seckill', 12999.00, 5, '2024-01-01 10:00:00', '2024-12-31 23:59:59', 1),
(3, 'AirPods Pro Seckill', 1599.00, 20, '2024-01-01 10:00:00', '2024-12-31 23:59:59', 1);