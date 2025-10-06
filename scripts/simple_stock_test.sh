#!/bin/bash

echo "开始简单库存测试..."

# 登录获取token
echo "登录用户..."
login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"account": "testuser5", "password": "password123"}')

token=$(echo $login_response | jq -r '.data.access_token')
if [ "$token" = "null" ] || [ "$token" = "" ]; then
    echo "登录失败: $login_response"
    exit 1
fi

echo "登录成功，获取token: ${token:0:20}..."

# 预热活动
echo "预热活动..."
prewarm_response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer $token")
echo "预热响应: $prewarm_response"

# 等待预热完成
sleep 2

# 检查初始库存
echo "检查初始库存..."
initial_stock=$(redis-cli GET "stock:1")
initial_reserved=$(redis-cli GET "stock:reserved:1")
echo "初始库存: $initial_stock"
echo "初始预留库存: $initial_reserved"

# 执行秒杀
echo "执行秒杀..."
seckill_response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "test_'$(date +%s%N)'"
    }')

echo "秒杀响应: $seckill_response"

# 等待订单处理
echo "等待订单处理..."
sleep 5

# 检查最终库存状态
echo "检查最终库存状态..."
final_stock=$(redis-cli GET "stock:1")
final_reserved=$(redis-cli GET "stock:reserved:1")
echo "最终库存: $final_stock"
echo "最终预留库存: $final_reserved"

# 检查订单数据
echo "检查订单数据..."
order_count=$(mysql -u root seckill -se "SELECT COUNT(*) FROM orders;")
echo "订单数量: $order_count"

# 检查最新订单
echo "检查最新订单..."
mysql -u root seckill -e "SELECT id, order_no, user_id, status, deduct_id, created_at FROM orders ORDER BY created_at DESC LIMIT 1;"

echo "简单库存测试完成！"