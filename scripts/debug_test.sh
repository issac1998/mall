#!/bin/bash

echo "开始调试测试..."

# 清理Redis数据
echo "清理Redis数据..."
redis-cli FLUSHALL

# 重置活动库存
echo "重置活动库存..."
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 1000 WHERE id = 1;"

# 登录用户
echo "登录用户..."
login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"account": "testuser0", "password": "password123"}')

token=$(echo $login_response | jq -r '.data.access_token')
echo "登录响应: $login_response"
echo "Token: $token"

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
echo "初始库存: $initial_stock"

# 执行多次秒杀测试
echo "执行多次秒杀测试..."
for i in {1..5}; do
    echo "第 $i 次秒杀..."
    request_id="debug_test_${i}_$(date +%s%N)"
    
    response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"activity_id\": 1,
            \"quantity\": 1,
            \"request_id\": \"$request_id\"
        }")
    
    echo "响应: $response"
    
    # 检查当前库存
    current_stock=$(redis-cli GET "stock:1")
    reserved_stock=$(redis-cli GET "stock:reserved:1")
    echo "当前库存: $current_stock, 预留库存: $reserved_stock"
    
    sleep 1
done

echo "调试测试完成！"