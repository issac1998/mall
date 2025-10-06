-- 先插入测试商品数据
INSERT INTO products (
    name, description, price, stock, status, created_at, updated_at
) VALUES 
('iPhone 15 Pro', '最新款iPhone，性能强劲', 999.90, 1000, 1, NOW(), NOW()),
('AirPods Pro', '主动降噪无线耳机', 179.90, 2000, 1, NOW(), NOW()),
('Apple Watch', '智能手表新品', 299.90, 1500, 1, NOW(), NOW());

-- 插入测试活动数据
INSERT INTO seckill_activities (
    product_id, name, seckill_price, seckill_stock, 
    start_time, end_time, status, created_at, updated_at
) VALUES 
(
    1,
    '限时秒杀活动1', 
    799.90,
    100,
    DATE_SUB(NOW(), INTERVAL 1 HOUR),
    DATE_ADD(NOW(), INTERVAL 23 HOUR),
    1,
    NOW(),
    NOW()
),
(
    2,
    '限时秒杀活动2', 
    139.90,
    200,
    DATE_SUB(NOW(), INTERVAL 30 MINUTE),
    DATE_ADD(NOW(), INTERVAL 1410 MINUTE),
    1,
    NOW(),
    NOW()
),
(
    3,
    '限时秒杀活动3', 
    249.90,
    150,
    DATE_ADD(NOW(), INTERVAL 1 HOUR),
    DATE_ADD(NOW(), INTERVAL 25 HOUR),
    0,
    NOW(),
    NOW()
);