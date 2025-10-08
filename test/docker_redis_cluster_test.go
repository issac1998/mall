package test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerRedisClusterConnection 测试Docker Redis集群连接
func TestDockerRedisClusterConnection(t *testing.T) {
	// 连接Docker Redis集群
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"127.0.0.1:7001",
			"127.0.0.1:7002",
			"127.0.0.1:7003",
			"127.0.0.1:7004",
			"127.0.0.1:7005",
			"127.0.0.1:7006",
		},
		Password: "",
	})
	defer client.Close()

	ctx := context.Background()

	// 测试连接
	pong, err := client.Ping(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)

	// 测试集群信息
	clusterInfo, err := client.ClusterInfo(ctx).Result()
	require.NoError(t, err)
	t.Logf("集群信息: %s", clusterInfo)

	// 测试节点信息
	clusterNodes, err := client.ClusterNodes(ctx).Result()
	require.NoError(t, err)
	t.Logf("节点信息: %s", clusterNodes)
}

// TestDockerRedisClusterStockConsistency 测试Docker Redis集群库存一致性
func TestDockerRedisClusterStockConsistency(t *testing.T) {
	// 连接Docker Redis集群
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"127.0.0.1:7001",
			"127.0.0.1:7002",
			"127.0.0.1:7003",
			"127.0.0.1:7004",
			"127.0.0.1:7005",
			"127.0.0.1:7006",
		},
		Password: "",
	})
	defer client.Close()

	ctx := context.Background()
	activityID := "activity_001"
	goodsID := "goods_001"
	hashTag := fmt.Sprintf("{%s:%s}", activityID, goodsID)
	
	stockKey := fmt.Sprintf("seckill:stock:%s", hashTag)
	userLimitKey := fmt.Sprintf("seckill:user_limit:%s:*", hashTag)
	requestKey := fmt.Sprintf("seckill:request:%s:*", hashTag)

	// 清理测试数据
	defer func() {
		// 删除库存键
		client.Del(ctx, stockKey)
		
		// 删除用户限制键
		keys, _ := client.Keys(ctx, userLimitKey).Result()
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		
		// 删除请求键
		keys, _ = client.Keys(ctx, requestKey).Result()
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
	}()

	// 设置初始库存
	initialStock := 100
	err := client.Set(ctx, stockKey, initialStock, 0).Err()
	require.NoError(t, err)

	// 库存扣减Lua脚本
	stockDeductScript := `
		local stock_key = KEYS[1]
		local user_limit_key = KEYS[2]
		local request_key = KEYS[3]
		local user_id = ARGV[1]
		local quantity = tonumber(ARGV[2])
		local user_limit = tonumber(ARGV[3])
		local request_ttl = tonumber(ARGV[4])

		-- 检查用户是否已经购买过
		local user_bought = redis.call('GET', user_limit_key)
		if user_bought then
			return {0, '用户已购买过'}
		end

		-- 检查是否重复请求
		local request_exists = redis.call('GET', request_key)
		if request_exists then
			return {0, '重复请求'}
		end

		-- 设置请求标记
		redis.call('SETEX', request_key, request_ttl, '1')

		-- 获取当前库存
		local current_stock = tonumber(redis.call('GET', stock_key) or 0)
		if current_stock < quantity then
			return {0, '库存不足'}
		end

		-- 扣减库存
		local new_stock = redis.call('DECRBY', stock_key, quantity)
		if new_stock < 0 then
			-- 回滚库存
			redis.call('INCRBY', stock_key, quantity)
			return {0, '库存不足'}
		end

		-- 设置用户购买标记
		redis.call('SETEX', user_limit_key, 86400, quantity)

		return {1, '扣减成功', new_stock}
	`

	// 并发测试
	userCount := 100
	var wg sync.WaitGroup
	results := make([]map[string]interface{}, userCount)

	for i := 0; i < userCount; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			userIDStr := fmt.Sprintf("user_%d", userID)
			userLimitKeyFull := fmt.Sprintf("seckill:user_limit:%s:%s", hashTag, userIDStr)
			requestKeyFull := fmt.Sprintf("seckill:request:%s:%s", hashTag, userIDStr)
			
			result, err := client.Eval(ctx, stockDeductScript, []string{
				stockKey,
				userLimitKeyFull,
				requestKeyFull,
			}, userIDStr, "1", "1", "60").Result()
			
			results[userID] = map[string]interface{}{
				"user_id": userIDStr,
				"result":  result,
				"error":   err,
			}
		}(i)
	}

	wg.Wait()

	// 检查最终库存
	finalStock, err := client.Get(ctx, stockKey).Int()
	require.NoError(t, err)

	// 统计成功次数
	successCount := 0
	for _, result := range results {
		if result["error"] == nil {
			if resultSlice, ok := result["result"].([]interface{}); ok && len(resultSlice) > 0 {
				if status, ok := resultSlice[0].(int64); ok && status == 1 {
					successCount++
				}
			}
		}
	}

	t.Logf("初始库存: %d", initialStock)
	t.Logf("成功扣减次数: %d", successCount)
	t.Logf("最终库存: %d", finalStock)
	t.Logf("预期最终库存: %d", initialStock-successCount)

	// 验证库存一致性
	assert.Equal(t, initialStock-successCount, finalStock, "库存不一致")
	assert.True(t, successCount > 0, "应该有成功的扣减")
	assert.True(t, successCount <= initialStock, "成功次数不应超过初始库存")
}

// TestDockerRedisClusterFailover 测试Docker Redis集群故障转移
func TestDockerRedisClusterFailover(t *testing.T) {
	// 连接Docker Redis集群
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"127.0.0.1:7001",
			"127.0.0.1:7002",
			"127.0.0.1:7003",
			"127.0.0.1:7004",
			"127.0.0.1:7005",
			"127.0.0.1:7006",
		},
		Password: "",
	})
	defer client.Close()

	ctx := context.Background()

	// 获取集群节点信息
	clusterNodes, err := client.ClusterNodes(ctx).Result()
	require.NoError(t, err)
	t.Logf("集群节点信息:\n%s", clusterNodes)

	// 测试数据写入和读取
	testKey := "test:failover:key"
	testValue := "test_value_" + strconv.FormatInt(time.Now().Unix(), 10)

	// 写入数据
	err = client.Set(ctx, testKey, testValue, time.Minute).Err()
	require.NoError(t, err)

	// 读取数据
	retrievedValue, err := client.Get(ctx, testKey).Result()
	require.NoError(t, err)
	assert.Equal(t, testValue, retrievedValue)

	// 清理测试数据
	client.Del(ctx, testKey)

	t.Log("故障转移测试完成")
}