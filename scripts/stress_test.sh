#!/bin/bash

# 压力测试脚本
# 测试系统在高并发下的性能表现

echo "开始压力测试..."

# 清理之前的测试数据
echo "清理测试数据..."
redis-cli FLUSHALL

# 重置库存
echo "重置库存..."
mysql -u root seckill -e "UPDATE seckill_activities SET seckill_stock = 1000 WHERE id = 1;"

# 创建测试用户并获取token
echo "创建测试用户..."
users=()
tokens=()

for i in {51..100}; do
    username="stressuser$i"
    phone="1380000$(printf '%04d' $i)"
    
    # 注册用户
    register_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$username\",
            \"password\": \"password123\",
            \"phone\": \"$phone\",
            \"email\": \"$username@example.com\"
        }")
    
    # 检查注册是否成功
    register_code=$(echo $register_response | jq -r '.code')
    if [ "$register_code" = "0" ]; then
        # 登录获取token
        login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
            -H "Content-Type: application/json" \
            -d "{
                \"account\": \"$username\",
                \"password\": \"password123\"
            }")
        
        token=$(echo $login_response | jq -r '.data.access_token')
        if [ "$token" != "null" ] && [ "$token" != "" ]; then
            users+=($username)
            tokens+=($token)
            echo "用户 $username 创建成功"
        else
            echo "用户 $username 登录失败: $login_response"
        fi
    else
        echo "用户 $username 注册失败: $register_response"
    fi
done

echo "成功创建 ${#users[@]} 个测试用户"

# 使用第一个用户的token预热活动
if [ ${#tokens[@]} -gt 0 ]; then
    echo "预热活动..."
    curl -X POST http://localhost:8080/api/v1/seckill/prewarm/1 \
        -H "Authorization: Bearer ${tokens[0]}"
    
    # 等待预热完成
    sleep 2
    
    # 检查初始库存
    echo "检查初始库存..."
    initial_stock=$(redis-cli GET "stock:1")
    echo "初始库存: $initial_stock"
else
    echo "错误: 没有成功创建任何用户"
    exit 1
fi

echo "成功创建 ${#users[@]} 个测试用户"

# 压力测试 - 阶段1: 中等并发 (50个用户)
echo "开始阶段1: 中等并发测试 (50个用户)..."
start_time=$(date +%s)
pids=()

for i in "${!users[@]}"; do
    (
        for j in {1..5}; do
            response=$(curl -s -X POST http://localhost:8080/api/v1/seckill/execute \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer ${tokens[$i]}" \
                -d "{
                    \"activity_id\": 1,
                    \"quantity\": 1,
                    \"request_id\": \"stress-${users[$i]}-$j-$(date +%s)-$RANDOM\"
                }")
            
            code=$(echo $response | jq -r '.code')
            if [ "$code" == "0" ]; then
                echo "用户 ${users[$i]} 请求 $j: 成功"
            else
                message=$(echo $response | jq -r '.message')
                echo "用户 ${users[$i]} 请求 $j: 失败 ($message)"
            fi
            
            # 随机延迟 0.1-0.5 秒
            sleep $(echo "scale=1; $(($RANDOM % 5 + 1)) / 10" | bc)
        done
    ) &
    pids+=($!)
    
    # 每10个用户启动后稍作延迟
    if [ $((($i + 1) % 10)) -eq 0 ]; then
        sleep 1
    fi
done

# 等待所有请求完成
echo "等待所有请求完成..."
for pid in "${pids[@]}"; do
    wait $pid
done

end_time=$(date +%s)
duration=$((end_time - start_time))

echo "阶段1完成，耗时: ${duration}秒"

# 等待订单处理完成
echo "等待订单处理完成..."
sleep 10

# 检查最终库存
final_stock=$(redis-cli GET "stock:1")
reserved_stock=$(redis-cli GET "stock:reserved:1")
stock_change=$((initial_stock - final_stock))

echo "=== 阶段1测试结果 ==="
echo "初始库存: $initial_stock"
echo "最终库存: $final_stock"
echo "预留库存: $reserved_stock"
echo "库存变化: $stock_change"
echo "测试耗时: ${duration}秒"
echo "平均QPS: $(echo "scale=2; 250 / $duration" | bc)"

# 检查订单数量
order_count=$(mysql -u root seckill -se "SELECT COUNT(*) FROM orders;")
order_detail_count=$(mysql -u root seckill -se "SELECT COUNT(*) FROM order_details;")

echo "总订单数: $order_count"
echo "总订单详情数: $order_detail_count"

# 检查错误率
success_count=$(mysql -u root seckill -se "SELECT COUNT(*) FROM orders WHERE status = 'pending';")
error_rate=$(echo "scale=4; (250 - $success_count) / 250 * 100" | bc)

echo "成功订单数: $success_count"
echo "错误率: ${error_rate}%"

echo "压力测试完成！"