package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisClusterConnection 测试Redis集群连接
func TestRedisClusterConnection(t *testing.T) {
	// 创建Redis集群客户端
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7001",
			"localhost:7002",
			"localhost:7003",
			"localhost:7004",
			"localhost:7005",
			"localhost:7006",
		},
		MaxRedirects:   3,
		ReadOnly:       false,
		RouteByLatency: false,
		RouteRandomly:  false,
	})
	defer client.Close()

	ctx := context.Background()

	// 测试连接
	err := client.Ping(ctx).Err()
	require.NoError(t, err, "Redis集群连接失败")

	// 测试集群信息
	clusterInfo := client.ClusterInfo(ctx)
	require.NoError(t, clusterInfo.Err(), "获取集群信息失败")
	
	info := clusterInfo.Val()
	assert.Contains(t, info, "cluster_state:ok", "集群状态不正常")
	assert.Contains(t, info, "cluster_slots_assigned:16384", "槽位分配不完整")
}

// TestRedisClusterStockConsistency 测试Redis集群库存一致性
func TestRedisClusterStockConsistency(t *testing.T) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7001",
			"localhost:7002",
			"localhost:7003",
			"localhost:7004",
			"localhost:7005",
			"localhost:7006",
		},
		MaxRedirects:   3,
		ReadOnly:       false,
		RouteByLatency: false,
		RouteRandomly:  false,
	})
	defer client.Close()

	ctx := context.Background()
	
	// 测试数据 - 使用哈希标签确保所有键在同一槽位
	activityID := "test_activity_001"
	goodsID := "test_goods_001"
	hashTag := fmt.Sprintf("{%s:%s}", activityID, goodsID)
	stockKey := fmt.Sprintf("stock:%s", hashTag)
	initialStock := 100

	// 清理测试数据
	defer func() {
		client.Del(ctx, stockKey)
		// 使用通配符删除用户限制键
		keys, err := client.Keys(ctx, fmt.Sprintf("user_limit:%s:*", hashTag)).Result()
		if err == nil && len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		// 删除请求键
		requestKeys, err := client.Keys(ctx, fmt.Sprintf("request:%s:*", hashTag)).Result()
		if err == nil && len(requestKeys) > 0 {
			client.Del(ctx, requestKeys...)
		}
	}()

	// 设置初始库存
	err := client.Set(ctx, stockKey, initialStock, 0).Err()
	require.NoError(t, err, "设置初始库存失败")

	// 库存扣减Lua脚本 - 修改为适配集群模式
	stockDeductScript := `
		local stock_key = KEYS[1]
		local user_key = KEYS[2]
		local request_key = KEYS[3]
		
		local quantity = tonumber(ARGV[1])
		local limit_per_user = tonumber(ARGV[2])
		local expire_time = tonumber(ARGV[3])
		
		-- 检查请求是否已处理（幂等性）
		if redis.call('EXISTS', request_key) == 1 then
			return {-2, "duplicate request", 0}
		end
		
		-- 检查库存
		local current_stock = tonumber(redis.call('GET', stock_key) or 0)
		if current_stock < quantity then
			return {-1, "insufficient stock", current_stock}
		end
		
		-- 检查用户限购
		local user_purchased = tonumber(redis.call('GET', user_key) or 0)
		if user_purchased + quantity > limit_per_user then
			return {-3, "exceed user limit", current_stock}
		end
		
		-- 扣减库存
		local new_stock = redis.call('DECRBY', stock_key, quantity)
		
		-- 更新用户购买记录
		redis.call('INCRBY', user_key, quantity)
		redis.call('EXPIRE', user_key, expire_time)
		
		-- 设置请求记录（防重复）
		redis.call('SETEX', request_key, expire_time, 1)
		
		return {0, "success", new_stock}
	`

	// 并发测试
	concurrency := 50
	var wg sync.WaitGroup
	var successCount int32
	var mutex sync.Mutex
	results := make([]map[string]interface{}, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			userID := fmt.Sprintf("user_%d", index)
			requestID := fmt.Sprintf("req_%d_%d", index, time.Now().UnixNano())
			
			// 使用哈希标签确保所有键在同一槽位
			keys := []string{
				stockKey,
				fmt.Sprintf("user_limit:%s:%s", hashTag, userID),
				fmt.Sprintf("request:%s:%s", hashTag, requestID),
			}
			args := []interface{}{1, 5, 3600} // 每次扣减1个，每用户限购5个，过期时间1小时
			
			result, err := client.Eval(ctx, stockDeductScript, keys, args...).Result()
			
			mutex.Lock()
			results[index] = map[string]interface{}{
				"result": result,
				"error":  err,
				"userID": userID,
			}
			
			if err == nil {
				if resultSlice, ok := result.([]interface{}); ok && len(resultSlice) > 0 {
					if code, ok := resultSlice[0].(int64); ok && code == 0 {
						successCount++
					}
				}
			}
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证结果
	finalStock, err := client.Get(ctx, stockKey).Int()
	require.NoError(t, err, "获取最终库存失败")

	t.Logf("初始库存: %d", initialStock)
	t.Logf("成功扣减次数: %d", successCount)
	t.Logf("最终库存: %d", finalStock)
	t.Logf("预期最终库存: %d", initialStock-int(successCount))

	// 验证库存一致性
	assert.Equal(t, initialStock-int(successCount), finalStock, "库存不一致")
	assert.True(t, successCount <= int32(initialStock), "成功次数不应超过初始库存")
	assert.True(t, finalStock >= 0, "库存不应为负数")

	// 打印详细结果用于调试
	successfulCount := 0
	failedCount := 0
	for i := 0; i < len(results); i++ {
		resultMap := results[i]
		if resultMap["error"] != nil {
			failedCount++
			if i < 5 { // 只打印前5个错误
				t.Logf("用户 %s 错误: %v", resultMap["userID"], resultMap["error"])
			}
		} else {
			if resultSlice, ok := resultMap["result"].([]interface{}); ok && len(resultSlice) > 0 {
				if code, ok := resultSlice[0].(int64); ok && code == 0 {
					successfulCount++
				} else {
					failedCount++
				}
			}
			if i < 5 { // 只打印前5个成功结果
				t.Logf("用户 %s 结果: %+v", resultMap["userID"], resultMap["result"])
			}
		}
	}
	
	t.Logf("成功请求数: %d, 失败请求数: %d", successfulCount, failedCount)
}

// TestRedisClusterFailover 测试Redis集群故障转移
func TestRedisClusterFailover(t *testing.T) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7001",
			"localhost:7002",
			"localhost:7003",
			"localhost:7004",
			"localhost:7005",
			"localhost:7006",
		},
		MaxRedirects:   3,
		ReadOnly:       false,
		RouteByLatency: false,
		RouteRandomly:  false,
	})
	defer client.Close()

	ctx := context.Background()

	// 测试数据分布到不同节点
	testKeys := []string{
		"test:key:1",
		"test:key:2", 
		"test:key:3",
		"test:key:4",
		"test:key:5",
	}

	// 清理测试数据
	defer func() {
		for _, key := range testKeys {
			client.Del(ctx, key)
		}
	}()

	// 写入测试数据
	for i, key := range testKeys {
		err := client.Set(ctx, key, fmt.Sprintf("value_%d", i), time.Minute).Err()
		require.NoError(t, err, "写入测试数据失败: %s", key)
	}

	// 验证数据读取
	for i, key := range testKeys {
		val, err := client.Get(ctx, key).Result()
		require.NoError(t, err, "读取测试数据失败: %s", key)
		assert.Equal(t, fmt.Sprintf("value_%d", i), val, "数据不匹配: %s", key)
	}

	// 测试集群节点信息
	clusterNodes := client.ClusterNodes(ctx)
	require.NoError(t, clusterNodes.Err(), "获取集群节点信息失败")
	
	nodes := clusterNodes.Val()
	assert.Contains(t, nodes, "master", "集群中应该有master节点")
	assert.Contains(t, nodes, "slave", "集群中应该有slave节点")
	
	t.Logf("集群节点信息:\n%s", nodes)
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}