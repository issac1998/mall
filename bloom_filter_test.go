package main

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seckill/internal/service/seckill"
)

func TestBloomFilterIntegration(t *testing.T) {
	// 创建Redis客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // 使用测试数据库
	})
	defer redisClient.Close()

	// 清理测试数据
	redisClient.FlushDB(context.Background())

	// 创建库存管理器
	inventory, err := seckill.NewMultiLevelInventory(redisClient)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("布隆过滤器防缓存穿透测试", func(t *testing.T) {
		// 测试不存在的商品ID
		nonExistentID := uint64(99999)

		// 不添加到布隆过滤器，直接检查
		available := inventory.LocalCheck(ctx, nonExistentID)
		assert.False(t, available, "不存在的商品应该被布隆过滤器拒绝")
	})

	t.Run("布隆过滤器和售罄缓存集成测试", func(t *testing.T) {
		activityID := uint64(1001)

		// 1. 添加商品到布隆过滤器
		inventory.AddToBloomFilter(activityID)

		// 2. 检查商品存在且未售罄
		available := inventory.LocalCheck(ctx, activityID)
		assert.True(t, available, "存在的商品且未售罄应该返回true")

		// 3. 标记商品为售罄
		err = inventory.MarkSoldOut(activityID)
		require.NoError(t, err)

		// 4. 再次检查，应该返回false（已售罄）
		available = inventory.LocalCheck(ctx, activityID)
		assert.False(t, available, "已售罄的商品应该返回false")
	})

	t.Run("完整秒杀流程测试", func(t *testing.T) {
		activityID := uint64(1002)

		// 1. 同步库存到Redis（会自动添加到布隆过滤器）
		err := inventory.SyncToRedis(ctx, activityID, 10)
		require.NoError(t, err)

		// 2. 检查商品可用
		available := inventory.LocalCheck(ctx, activityID)
		assert.True(t, available, "同步后的商品应该可用")

		// 3. 尝试扣减库存
		req := &seckill.DeductRequest{
			RequestID:  "test_request_001",
			ActivityID: activityID,
			UserID:     12345,
			Quantity:   2,
		}

		result, err := inventory.TryDeductWithLimit(ctx, req, 1)
		require.NoError(t, err)
		assert.True(t, result.Success, "库存扣减应该成功")
		assert.Equal(t, 8, result.RemainStock, "剩余库存应该正确")

		// 4. 模拟库存耗尽，标记售罄
		err = inventory.MarkSoldOut(activityID)
		require.NoError(t, err)

		// 5. 再次尝试扣减，应该被售罄检查拒绝
		req2 := &seckill.DeductRequest{
			RequestID:  "test_request_002",
			ActivityID: activityID,
			UserID:     12346,
			Quantity:   1,
		}

		result2, err := inventory.TryDeductWithLimit(ctx, req2, 1)
		require.NoError(t, err)
		assert.False(t, result2.Success, "售罄后的扣减应该失败")
		assert.Contains(t, result2.Message, "售罄", "错误消息应该包含售罄信息")
	})

	t.Run("布隆过滤器性能测试", func(t *testing.T) {
		// 添加大量商品到布隆过滤器
		start := time.Now()
		for i := 2000; i < 3000; i++ {
			inventory.AddToBloomFilter(uint64(i))
		}
		addDuration := time.Since(start)
		t.Logf("添加1000个商品到布隆过滤器耗时: %v", addDuration)

		// 测试查询性能
		start = time.Now()
		for i := 2000; i < 3000; i++ {
			available := inventory.LocalCheck(ctx, uint64(i))
			assert.True(t, available, "存在的商品应该返回true")
		}
		checkDuration := time.Since(start)
		t.Logf("检查1000个商品耗时: %v", checkDuration)

		// 测试不存在商品的查询性能
		start = time.Now()
		for i := 5000; i < 6000; i++ {
			available := inventory.LocalCheck(ctx, uint64(i))
			assert.False(t, available, "不存在的商品应该被拒绝")
		}
		nonExistentCheckDuration := time.Since(start)
		t.Logf("检查1000个不存在商品耗时: %v", nonExistentCheckDuration)
	})
}
