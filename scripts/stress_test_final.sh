#!/bin/bash

# 最终压力测试脚本 - 使用不同用户进行真正的并发测试

echo "========== 秒杀系统最终压力测试 =========="

# 清理环境
echo "清理环境..."
redis-cli FLUSHALL > /dev/null
mysql -u root seckill -e "DELETE FROM orders;" > /dev/null
mysql -u root seckill -e "DELETE FROM users WHERE username LIKE 'stressuser%';" > /dev/null

# 创建测试用户并获取token
echo "创建测试用户并获取token..."
declare -a tokens
user_count=5

for ((i=0; i<$user_count; i++)); do
    # 生成用户数据
    username="stressuser$i"
    password="password123"
    # 确保手机号是11位数字且唯一
    phone=$(printf "138%08d" $((10000000 + i)))
    email="stressuser$i@test.com"
    
    # 注册用户
    register_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$username\",
            \"password\": \"$password\",
            \"email\": \"$username@test.com\",
            \"phone\": \"$phone\"
        }")
    
    echo "Register response for $username: $register_response"
    
    # 登录获取token
    login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d "{
            \"account\": \"$username\",
            \"password\": \"$password\"
        }")
    
    echo "Login response for $username: $login_response"
    
    token=$(echo "$login_response" | jq -r '.data.access_token')
    if [ "$token" != "null" ] && [ -n "$token" ]; then
        tokens[$i]=$token
        echo "Token for $username: $token"
    else
        echo "用户 $username 登录失败"
        exit 1
    fi
done

echo "成功创建 $user_count 个测试用户"

# 预热活动
echo "预热活动..."
prewarm_response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer ${tokens[0]}" \
    -H "Content-Type: application/json")

if echo "$prewarm_response" | jq -e '.code == 0' > /dev/null; then
    echo "活动预热成功"
else
    echo "活动预热失败: $prewarm_response"
    exit 1
fi

sleep 2

# 压力测试函数
run_stress_test() {
    local concurrent_users=$1
    local test_name=$2
    
    echo "========== $test_name (并发用户: $concurrent_users) =========="
    
    # 检查初始库存
    initial_stock=$(redis-cli GET "stock:1")
    if [ -z "$initial_stock" ] || [ "$initial_stock" = "(nil)" ]; then
        echo "警告: Redis中没有库存数据，跳过此测试"
        return
    fi
    echo "初始库存: $initial_stock"
    
    # 记录开始时间
    start_time=$(date +%s.%N)
    
    # 并发执行秒杀
    for ((i=0; i<$concurrent_users; i++)); do
        {
            # 使用循环方式获取token，避免数组越界
            token_index=$((i % user_count))
            user_token=${tokens[$token_index]}
            
            if [ -z "$user_token" ] || [ "$user_token" = "null" ]; then
                echo "User $i: No valid token available"
                echo '{"code":401,"message":"No valid token"}' > "/tmp/stress_result_${concurrent_users}_${i}.json"
                continue
            fi
            
            response=$(curl -s -w "%{http_code}" -X POST http://localhost:8080/api/v1/seckill/execute \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $user_token" \
                -d "{
                    \"request_id\": \"req_stress_test_${concurrent_users}_${i}_$(date +%s%N)\",
                    \"activity_id\": 1,
                    \"user_id\": $((1182 + token_index)),
                    \"quantity\": 1
                }")
            
            echo "User $i (token_index: $token_index) response: $response"
            echo "$response" > "/tmp/stress_result_${concurrent_users}_${i}.json"
        } &
    done
    
    # 等待所有请求完成
    wait
    
    # 记录结束时间
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)
    
    # 等待订单处理完成
    sleep 3
    
    # 统计结果
    success_count=0
    error_count=0
    insufficient_stock_count=0
    purchase_limit_count=0
    other_error_count=0
    
    for ((i=0; i<$concurrent_users; i++)); do
        if [ -f "/tmp/stress_result_${concurrent_users}_${i}.json" ]; then
            code=$(jq -r '.code' "/tmp/stress_result_${concurrent_users}_${i}.json" 2>/dev/null)
            message=$(jq -r '.message' "/tmp/stress_result_${concurrent_users}_${i}.json" 2>/dev/null)
            
            if [ "$code" = "0" ]; then
                ((success_count++))
            else
                ((error_count++))
                case "$message" in
                    "insufficient_stock")
                        ((insufficient_stock_count++))
                        ;;
                    "purchase_limit_exceeded")
                        ((purchase_limit_count++))
                        ;;
                    *)
                        ((other_error_count++))
                        ;;
                esac
            fi
        fi
    done
    
    # 检查最终状态
    final_stock=$(redis-cli GET "stock:1")
    reserved_stock=$(redis-cli GET "stock:reserved:1")
    order_count=$(mysql -u root seckill -e "SELECT COUNT(*) as count FROM orders;" | tail -n 1)
    
    # 计算性能指标
    success_rate=$(echo "scale=2; $success_count * 100 / $concurrent_users" | bc)
    qps=$(echo "scale=2; $concurrent_users / $duration" | bc)
    
    echo "测试结果:"
    echo "  - 并发用户数: $concurrent_users"
    echo "  - 执行时间: ${duration}s"
    echo "  - QPS: $qps"
    echo "  - 成功请求: $success_count"
    echo "  - 失败请求: $error_count"
    echo "    - 库存不足: $insufficient_stock_count"
    echo "    - 购买限制: $purchase_limit_count"
    echo "    - 其他错误: $other_error_count"
    echo "  - 成功率: ${success_rate}%"
    echo "  - 初始库存: $initial_stock"
    echo "  - 最终库存: $final_stock"
    echo "  - 预留库存: $reserved_stock"
    echo "  - 订单数量: $order_count"
    echo "  - 库存一致性: $([ "$final_stock" = "$((initial_stock - success_count))" ] && echo "通过" || echo "失败")"
    echo ""
    
    # 清理临时文件
    rm -f /tmp/stress_result_${concurrent_users}_*.json
}

# 执行不同级别的压力测试
run_stress_test 10 "轻度压力测试"
run_stress_test 50 "中度压力测试"
run_stress_test 100 "重度压力测试"

echo "========== 压力测试完成 =========="

# 最终系统状态检查
echo "========== 最终系统状态 =========="
final_stock=$(mysql -h localhost -u root -p123456 -D seckill_dev -e "SELECT seckill_stock FROM seckill_activities WHERE id = 1;" -s -N 2>/dev/null || echo "0")
reserved_stock=$(redis-cli -h localhost -p 6379 GET "stock:reserved:1" 2>/dev/null || echo "0")
total_orders=$(mysql -h localhost -u root -p123456 -D seckill_dev -e "SELECT COUNT(*) FROM orders WHERE seckill_id = 1;" -s -N 2>/dev/null || echo "0")
total_users=$(mysql -h localhost -u root -p123456 -D seckill_dev -e "SELECT COUNT(*) FROM users WHERE username LIKE 'stressuser%';" -s -N 2>/dev/null || echo "0")

echo "最终库存: $final_stock"
echo "预留库存: $reserved_stock"
echo "总订单数: $total_orders"
echo "测试用户数: $total_users"

# 检查是否有超卖
activity_stock=$(mysql -h localhost -u root -p123456 -D seckill_dev -e "SELECT seckill_stock FROM seckill_activities WHERE id = 1;" -s -N 2>/dev/null || echo "0")
activity_sold=$(mysql -h localhost -u root -p123456 -D seckill_dev -e "SELECT COUNT(*) FROM orders WHERE seckill_id = 1;" -s -N 2>/dev/null || echo "0")
echo "活动总库存: $activity_stock"
echo "活动已售: $activity_sold"

if [ "$total_orders" -gt "$activity_stock" ]; then
    echo "❌ 检测到超卖！订单数($total_orders) > 库存数($activity_stock)"
else
    echo "✅ 无超卖现象，系统运行正常"
fi