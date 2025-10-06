#!/bin/bash

echo "开始边界情况测试..."

# 测试1: 库存为0的情况
echo "========== 测试1: 库存为0的情况 =========="
redis-cli FLUSHALL
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 0 WHERE id = 1;"

# 登录用户
login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"account": "testuser0", "password": "password123"}')
token=$(echo $login_response | jq -r '.data.access_token')
echo "登录响应: $login_response"
echo "Token: $token"

# 预热活动
curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer $token" > /dev/null
sleep 1

# 尝试秒杀
response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "boundary_test_zero_stock"
    }')
echo "库存为0时秒杀响应: $response"

# 测试2: 库存为1的情况（边界值）
echo "========== 测试2: 库存为1的情况 =========="
redis-cli FLUSHALL
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 1 WHERE id = 1;"

# 重新登录获取新token
login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"account": "testuser0", "password": "password123"}')
token=$(echo $login_response | jq -r '.data.access_token')

# 预热活动
curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer $token" > /dev/null
sleep 1

# 第一次秒杀
response1=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "boundary_test_one_stock_1"
    }')
echo "库存为1时第一次秒杀响应: $response1"

sleep 2

# 第二次秒杀（应该失败）
response2=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "boundary_test_one_stock_2"
    }')
echo "库存为1时第二次秒杀响应: $response2"

# 测试3: 无效的活动ID
echo "========== 测试3: 无效的活动ID =========="
response3=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 999,
        "quantity": 1,
        "request_id": "boundary_test_invalid_activity"
    }')
echo "无效活动ID响应: $response3"

# 测试4: 无效的数量（0和负数）
echo "========== 测试4: 无效的数量 =========="
response4=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 0,
        "request_id": "boundary_test_zero_quantity"
    }')
echo "数量为0响应: $response4"

response5=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": -1,
        "request_id": "boundary_test_negative_quantity"
    }')
echo "数量为负数响应: $response5"

# 测试5: 无效的token
echo "========== 测试5: 无效的token =========="
response6=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer invalid_token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "boundary_test_invalid_token"
    }')
echo "无效token响应: $response6"

# 测试6: 缺少Authorization头
echo "========== 测试6: 缺少Authorization头 =========="
response7=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "boundary_test_no_auth"
    }')
echo "缺少Authorization头响应: $response7"

# 测试7: 重复的request_id
echo "========== 测试7: 重复的request_id =========="
redis-cli FLUSHALL
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 10 WHERE id = 1;"

# 预热活动
curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer $token" > /dev/null
sleep 1

# 第一次请求
response8=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "duplicate_request_id"
    }')
echo "第一次重复request_id响应: $response8"

sleep 1

# 第二次相同请求
response9=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d '{
        "activity_id": 1,
        "quantity": 1,
        "request_id": "duplicate_request_id"
    }')
echo "第二次重复request_id响应: $response9"

echo "边界情况测试完成！"