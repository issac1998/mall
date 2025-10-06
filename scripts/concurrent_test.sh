#!/bin/bash

# 并发秒杀测试脚本
# 测试多个用户同时进行秒杀，验证库存一致性和防超卖机制

echo "开始并发秒杀测试..."

# 清理之前的测试数据
echo "清理测试数据..."
redis-cli DEL "purchase_count:1:6" "purchase_count:1:7" "purchase_count:1:8" "purchase_count:1:9" "purchase_count:1:10"

# 预热活动
echo "预热活动..."
curl -X POST http://localhost:8080/api/v1/seckill/prewarm/1

# 等待预热完成
sleep 2

# 检查初始库存
echo "检查初始库存..."
redis-cli GET "stock:1"

# 定义用户token数组
declare -a tokens=(
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo2LCJ1c2VybmFtZSI6InRlc3R1c2VyNiIsInJvbGUiOiJ1c2VyIiwiaXNzIjoic2Vja2lsbC1zeXN0ZW0iLCJzdWIiOiJ0ZXN0dXNlcjYiLCJleHAiOjE3NTk4NTU0MzEsIm5iZiI6MTc1OTc2OTAzMSwiaWF0IjoxNzU5NzY5MDMxfQ.test6"
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo3LCJ1c2VybmFtZSI6InRlc3R1c2VyNyIsInJvbGUiOiJ1c2VyIiwiaXNzIjoic2Vja2lsbC1zeXN0ZW0iLCJzdWIiOiJ0ZXN0dXNlcjciLCJleHAiOjE3NTk4NTU0MzEsIm5iZiI6MTc1OTc2OTAzMSwiaWF0IjoxNzU5NzY5MDMxfQ.test7"
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo4LCJ1c2VybmFtZSI6InRlc3R1c2VyOCIsInJvbGUiOiJ1c2VyIiwiaXNzIjoic2Vja2lsbC1zeXN0ZW0iLCJzdWIiOiJ0ZXN0dXNlcjgiLCJleHAiOjE3NTk4NTU0MzEsIm5iZiI6MTc1OTc2OTAzMSwiaWF0IjoxNzU5NzY5MDMxfQ.test8"
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo5LCJ1c2VybmFtZSI6InRlc3R1c2VyOSIsInJvbGUiOiJ1c2VyIiwiaXNzIjoic2Vja2lsbC1zeXN0ZW0iLCJzdWIiOiJ0ZXN0dXNlcjkiLCJleHAiOjE3NTk4NTU0MzEsIm5iZiI6MTc1OTc2OTAzMSwiaWF0IjoxNzU5NzY5MDMxfQ.test9"
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMCwidXNlcm5hbWUiOiJ0ZXN0dXNlcjEwIiwicm9sZSI6InVzZXIiLCJpc3MiOiJzZWNraWxsLXN5c3RlbSIsInN1YiI6InRlc3R1c2VyMTAiLCJleHAiOjE3NTk4NTU0MzEsIm5iZiI6MTc1OTc2OTAzMSwiaWF0IjoxNzU5NzY5MDMxfQ.test10"
)

# 并发执行秒杀请求
echo "开始并发秒杀测试..."
pids=()

for i in {6..10}; do
    token_index=$((i-6))
    (
        response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${tokens[$token_index]}" \
            -d "{
                \"activity_id\": 1,
                \"quantity\": 1,
                \"request_id\": \"concurrent-test-$i-$(date +%s%3N)\"
            }")
        echo "用户$i 响应: $response"
    ) &
    pids+=($!)
done

# 等待所有请求完成
for pid in "${pids[@]}"; do
    wait $pid
done

echo "等待订单处理完成..."
sleep 5

# 检查最终库存
echo "检查最终库存..."
final_stock=$(redis-cli GET "stock:1")
echo "最终库存: $final_stock"

# 检查订单数量
echo "检查订单数量..."
order_count=$(mysql -h 127.0.0.1 -P 3306 -u root -D seckill -se "SELECT COUNT(*) FROM orders;")
echo "总订单数: $order_count"

# 检查订单详情数量
echo "检查订单详情数量..."
detail_count=$(mysql -h 127.0.0.1 -P 3306 -u root -D seckill -se "SELECT COUNT(*) FROM order_details;")
echo "总订单详情数: $detail_count"

echo "并发测试完成！"