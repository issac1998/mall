package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// BenchmarkRedisClusterStockDeduct 基准测试Redis集群库存扣减性能
func BenchmarkRedisClusterStockDeduct(b *testing.B) {
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
	
	// 测试数据
	activityID := "bench_activity_001"
	goodsID := "bench_goods_001"
	hashTag := fmt.Sprintf("{%s:%s}", activityID, goodsID)
	stockKey := fmt.Sprintf("stock:%s", hashTag)
	initialStock := 10000

	// 设置初始库存
	err := client.Set(ctx, stockKey, initialStock, 0).Err()
	require.NoError(b, err, "设置初始库存失败")

	// 清理测试数据
	defer func() {
		client.Del(ctx, stockKey)
		keys, err := client.Keys(ctx, fmt.Sprintf("user_limit:%s:*", hashTag)).Result()
		if err == nil && len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		requestKeys, err := client.Keys(ctx, fmt.Sprintf("request:%s:*", hashTag)).Result()
		if err == nil && len(requestKeys) > 0 {
			client.Del(ctx, requestKeys...)
		}
	}()

	// 库存扣减Lua脚本
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

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		userID := fmt.Sprintf("bench_user_%d", time.Now().UnixNano())
		counter := 0
		
		for pb.Next() {
			counter++
			requestID := fmt.Sprintf("bench_req_%s_%d", userID, counter)
			
			keys := []string{
				stockKey,
				fmt.Sprintf("user_limit:%s:%s", hashTag, userID),
				fmt.Sprintf("request:%s:%s", hashTag, requestID),
			}
			args := []interface{}{1, 100, 3600} // 每次扣减1个，每用户限购100个
			
			client.Eval(ctx, stockDeductScript, keys, args...)
		}
	})
}

// TestRedisClusterConcurrentStockDeduct 测试Redis集群并发库存扣减
func TestRedisClusterConcurrentStockDeduct(t *testing.T) {
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
	
	// 测试不同并发级别
	concurrencyLevels := []int{10, 50, 100, 200}
	
	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			// 测试数据
			activityID := fmt.Sprintf("concurrent_activity_%d", concurrency)
			goodsID := "concurrent_goods_001"
			hashTag := fmt.Sprintf("{%s:%s}", activityID, goodsID)
			stockKey := fmt.Sprintf("stock:%s", hashTag)
			initialStock := 1000

			// 清理测试数据
			defer func() {
				client.Del(ctx, stockKey)
				keys, err := client.Keys(ctx, fmt.Sprintf("user_limit:%s:*", hashTag)).Result()
				if err == nil && len(keys) > 0 {
					client.Del(ctx, keys...)
				}
				requestKeys, err := client.Keys(ctx, fmt.Sprintf("request:%s:*", hashTag)).Result()
				if err == nil && len(requestKeys) > 0 {
					client.Del(ctx, requestKeys...)
				}
			}()

			// 设置初始库存
			err := client.Set(ctx, stockKey, initialStock, 0).Err()
			require.NoError(t, err, "设置初始库存失败")

			// 库存扣减Lua脚本
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
			var wg sync.WaitGroup
			var successCount int32
			var mutex sync.Mutex
			startTime := time.Now()

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					
					userID := fmt.Sprintf("concurrent_user_%d_%d", concurrency, index)
					requestID := fmt.Sprintf("concurrent_req_%d_%d_%d", concurrency, index, time.Now().UnixNano())
					
					keys := []string{
						stockKey,
						fmt.Sprintf("user_limit:%s:%s", hashTag, userID),
						fmt.Sprintf("request:%s:%s", hashTag, requestID),
					}
					args := []interface{}{1, 10, 3600} // 每次扣减1个，每用户限购10个
					
					result, err := client.Eval(ctx, stockDeductScript, keys, args...).Result()
					
					if err == nil {
						if resultSlice, ok := result.([]interface{}); ok && len(resultSlice) > 0 {
							if code, ok := resultSlice[0].(int64); ok && code == 0 {
								mutex.Lock()
								successCount++
								mutex.Unlock()
							}
						}
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(startTime)

			// 验证结果
			finalStock, err := client.Get(ctx, stockKey).Int()
			require.NoError(t, err, "获取最终库存失败")

			t.Logf("并发级别: %d", concurrency)
			t.Logf("初始库存: %d", initialStock)
			t.Logf("成功扣减次数: %d", successCount)
			t.Logf("最终库存: %d", finalStock)
			t.Logf("执行时间: %v", duration)
			t.Logf("TPS: %.2f", float64(successCount)/duration.Seconds())
			
			// 验证库存一致性
			expectedStock := initialStock - int(successCount)
			if finalStock != expectedStock {
				t.Errorf("库存不一致: 期望 %d, 实际 %d", expectedStock, finalStock)
			}
		})
	}
}

// TestRedisClusterLatency 测试Redis集群延迟
func TestRedisClusterLatency(t *testing.T) {
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
	
	// 测试不同操作的延迟
	operations := []struct {
		name string
		op   func() error
	}{
		{
			name: "SET",
			op: func() error {
				return client.Set(ctx, "latency_test_set", "value", time.Minute).Err()
			},
		},
		{
			name: "GET",
			op: func() error {
				return client.Get(ctx, "latency_test_set").Err()
			},
		},
		{
			name: "INCR",
			op: func() error {
				return client.Incr(ctx, "latency_test_incr").Err()
			},
		},
		{
			name: "EVAL",
			op: func() error {
				script := "return redis.call('GET', KEYS[1])"
				return client.Eval(ctx, script, []string{"latency_test_set"}).Err()
			},
		},
	}

	// 清理测试数据
	defer func() {
		client.Del(ctx, "latency_test_set", "latency_test_incr")
	}()

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			iterations := 100
			var totalDuration time.Duration
			var minDuration = time.Hour
			var maxDuration time.Duration

			for i := 0; i < iterations; i++ {
				start := time.Now()
				err := operation.op()
				duration := time.Since(start)
				
				require.NoError(t, err, "操作失败")
				
				totalDuration += duration
				if duration < minDuration {
					minDuration = duration
				}
				if duration > maxDuration {
					maxDuration = duration
				}
			}

			avgDuration := totalDuration / time.Duration(iterations)
			
			t.Logf("操作: %s", operation.name)
			t.Logf("平均延迟: %v", avgDuration)
			t.Logf("最小延迟: %v", minDuration)
			t.Logf("最大延迟: %v", maxDuration)
			t.Logf("总执行时间: %v", totalDuration)
		})
	}
}