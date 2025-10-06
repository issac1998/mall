#!/bin/bash

# 性能测试脚本 - 测试系统在不同并发级别下的性能表现

echo "========== 秒杀系统性能测试 =========="

# 清理环境
echo "清理环境..."
redis-cli FLUSHALL > /dev/null
mysql -u root seckill -e "DELETE FROM orders WHERE id > 0;" > /dev/null
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 1000 WHERE id = 1;" > /dev/null

# 创建测试用户（如果不存在）
echo "准备测试用户..."
for i in {0..49}; do
    mysql -u root seckill -e "INSERT IGNORE INTO users (username, phone, password_hash, created_at, updated_at) VALUES ('testuser$i', '1380000000$i', '\$2a\$10\$example.hash.for.password123', NOW(), NOW());" > /dev/null 2>&1
done

# 登录所有用户获取token
echo "登录用户获取token..."
declare -a tokens
for i in {0..49}; do
    login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"account\": \"testuser$i\", \"password\": \"password123\"}")
    token=$(echo $login_response | jq -r '.data.access_token')
    if [ "$token" = "null" ] || [ -z "$token" ]; then
        echo "用户 testuser$i 登录失败: $login_response"
        continue
    fi
    tokens[$i]=$token
    echo "用户 testuser$i 登录成功"
done

# 预热活动
echo "预热活动..."
prewarm_response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
    -H "Authorization: Bearer ${tokens[0]}")
echo "预热响应: $prewarm_response"
sleep 2

# 性能测试函数
run_performance_test() {
    local concurrent_users=$1
    local test_name=$2
    
    echo "========== $test_name (并发用户: $concurrent_users) =========="
    
    # 重置环境
    redis-cli FLUSHALL > /dev/null
    mysql -u root seckill -e "DELETE FROM orders WHERE id > 0;" > /dev/null
    mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 100 WHERE id = 1;" > /dev/null
    
    # 重新登录获取新token（避免token过期）
    login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d '{"account": "testuser0", "password": "password123"}')
    test_token=$(echo $login_response | jq -r '.data.access_token')
    
    if [ "$test_token" = "null" ] || [ -z "$test_token" ]; then
        echo "获取测试token失败，跳过此测试"
        return
    fi
    
    # 预热
    prewarm_response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
        -H "Authorization: Bearer $test_token")
    echo "预热响应: $prewarm_response"
    
    # 检查预热是否成功
    if echo "$prewarm_response" | grep -q '"code":0'; then
        echo "预热成功"
    else
        echo "预热失败，跳过此测试"
        return
    fi
    
    sleep 2
    
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
            # 使用新获取的token而不是数组中的token
            response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
                -H "Authorization: Bearer $test_token" \
                -H "Content-Type: application/json" \
                -d "{
                    \"activity_id\": 1,
                    \"quantity\": 1,
                    \"request_id\": \"perf_test_${concurrent_users}_${i}\"
                }")
            echo "$response" > "/tmp/perf_result_${concurrent_users}_${i}.json"
            echo "User $i response: $response"
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
    
    for ((i=0; i<$concurrent_users; i++)); do
        if [ -f "/tmp/perf_result_${concurrent_users}_${i}.json" ]; then
            code=$(jq -r '.code' "/tmp/perf_result_${concurrent_users}_${i}.json" 2>/dev/null)
            if [ "$code" = "0" ]; then
                ((success_count++))
            else
                ((error_count++))
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
    echo "  - 成功率: ${success_rate}%"
    echo "  - 最终库存: $final_stock"
    echo "  - 预留库存: $reserved_stock"
    echo "  - 订单数量: $order_count"
    echo "  - 库存一致性: $([ "$final_stock" = "$((initial_stock - success_count))" ] && echo "通过" || echo "失败")"
    echo ""
    
    # 清理临时文件
    rm -f /tmp/perf_result_${concurrent_users}_*.json
}

# 执行不同并发级别的性能测试
run_performance_test 5 "低并发测试"
run_performance_test 10 "中等并发测试"
run_performance_test 20 "高并发测试"
run_performance_test 50 "极高并发测试"

echo "========== 性能测试完成 =========="