#!/bin/bash

# 简化的并发秒杀测试脚本
# 使用现有的testuser账号进行并发测试

echo "开始简化并发秒杀测试..."

# 获取testuser的token
echo "获取用户token..."
token_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "testuser",
    "password": "password123"
  }')

token=$(echo $token_response | jq -r '.data.access_token')
echo "Token获取成功"

# 预热活动
echo "预热活动..."
curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
  -H "Authorization: Bearer $token"

# 等待预热完成
sleep 2

# 检查初始库存
echo "检查初始库存..."
initial_stock=$(redis-cli GET "stock:1")
echo "初始库存: $initial_stock"

# 并发执行多个秒杀请求（使用同一个用户，测试防重复购买）
echo "开始并发秒杀测试（同一用户多次请求）..."
pids=()

for i in {1..5}; do
    (
        # 生成唯一的request_id，包含进程ID和纳秒时间戳
        unique_id="test-$i-$$-$(date +%s)-$RANDOM"
        response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "{
                \"activity_id\": 1,
                \"quantity\": 1,
                \"request_id\": \"$unique_id\"
            }")
        echo "请求$i (ID: $unique_id) 响应: $response"
    ) &
    pids+=($!)
done

# 等待所有请求完成
for pid in "${pids[@]}"; do
    wait $pid
done

echo "等待订单处理完成..."
sleep 3

# 检查最终库存
echo "检查最终库存..."
final_stock=$(redis-cli GET "stock:1")
echo "最终库存: $final_stock"

# 计算库存变化
if [ "$initial_stock" != "null" ] && [ "$final_stock" != "null" ]; then
    stock_used=$((initial_stock - final_stock))
    echo "库存变化: $stock_used (应该最多为1，因为同一用户有购买限制)"
else
    echo "库存数据异常"
fi

# 检查用户购买记录
echo "检查用户购买记录..."
purchase_count=$(redis-cli GET "purchase_count:1:5")
echo "用户5购买次数: $purchase_count"

echo "简化并发测试完成！"