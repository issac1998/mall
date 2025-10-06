#!/bin/bash

echo "开始防超卖测试..."

# 清理Redis数据
echo "清理Redis数据..."
redis-cli FLUSHALL

# 重置活动库存为10（小库存测试）
echo "重置活动库存为10..."
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 10 WHERE id = 1;"

# 使用现有用户进行测试
users=("testuser0" "testuser1" "testuser2" "testuser3" "testuser4" "testuser5" "testuser6" "testuser7" "testuser8" "testuser9")
tokens=()

# 登录所有用户获取token
echo "登录用户获取token..."
for user in "${users[@]}"; do
    login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"account\": \"$user\", \"password\": \"password123\"}")
    
    token=$(echo $login_response | jq -r '.data.access_token')
    if [ "$token" != "null" ] && [ "$token" != "" ]; then
        tokens+=("$token")
        echo "用户 $user 登录成功"
    else
        echo "用户 $user 登录失败: $login_response"
    fi
done

echo "成功登录 ${#tokens[@]} 个用户"

# 预热活动
echo "预热活动..."
prewarm_response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer ${tokens[0]}")
echo "预热响应: $prewarm_response"

# 等待预热完成
sleep 2

# 检查初始库存
echo "检查初始库存..."
initial_stock=$(redis-cli GET "stock:1")
echo "初始库存: $initial_stock"

# 并发执行秒杀（10个用户同时抢10个商品）
echo "开始防超卖测试（10个用户抢10个商品）..."
success_count=0
error_count=0

# 创建临时目录存储结果
mkdir -p /tmp/oversell_results
rm -f /tmp/oversell_results/*

# 并发执行秒杀
for i in $(seq 0 9); do
    {
        token=${tokens[$i]}
        request_id="oversell_test_${i}_$(date +%s%N)"
        
        response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "{
                \"activity_id\": 1,
                \"quantity\": 1,
                \"request_id\": \"$request_id\"
            }")
        
        echo "$response" > "/tmp/oversell_results/result_$i.json"
        echo "用户 $i 响应: $response"
    } &
done

# 等待所有请求完成
wait

# 统计结果
echo "统计测试结果..."
for result_file in /tmp/oversell_results/result_*.json; do
    if [ -f "$result_file" ]; then
        code=$(jq -r '.code' "$result_file" 2>/dev/null)
        if [ "$code" = "0" ]; then
            ((success_count++))
        else
            ((error_count++))
        fi
    fi
done

# 等待订单处理完成
echo "等待订单处理完成..."
sleep 5

# 检查最终状态
echo "检查最终库存状态..."
final_stock=$(redis-cli GET "stock:1")
final_reserved=$(redis-cli GET "stock:reserved:1")
echo "最终库存: $final_stock"
echo "最终预留库存: $final_reserved"

# 检查订单数据
echo "检查订单数据..."
order_count=$(mysql -u root seckill -se "SELECT COUNT(*) FROM orders WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 MINUTE);")
echo "最近1分钟订单数量: $order_count"

# 计算库存变化
stock_change=$((initial_stock - final_stock))

echo "========== 防超卖测试结果 =========="
echo "初始库存: $initial_stock"
echo "最终库存: $final_stock"
echo "库存变化: $stock_change"
echo "成功请求: $success_count"
echo "失败请求: $error_count"
echo "最近订单: $order_count"
echo "防超卖验证: $([ $stock_change -eq $success_count ] && echo "通过" || echo "失败")"
echo "库存一致性: $([ $final_stock -ge 0 ] && echo "通过" || echo "失败")"
echo "================================="

# 清理临时文件
rm -rf /tmp/oversell_results

echo "防超卖测试完成！"