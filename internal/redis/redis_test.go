package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seckill/internal/config"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid_config",
			config: &config.Config{
				Redis: config.RedisConfig{
					Host:         "127.0.0.1",
					Port:         6379,
					Password:     "",
					DB:           0,
					PoolSize:     10,
					MinIdleConns: 5,
					MaxRetries:   3,
					DialTimeout:  5,
					ReadTimeout:  3,
					WriteTimeout: 3,
					IdleTimeout:  300,
				},
			},
			wantErr: true, // 预期失败，因为没有Redis服务
		},
		{
			name: "invalid_config",
			config: &config.Config{
				Redis: config.RedisConfig{
					Host:         "invalid-host",
					Port:         6379,
					Password:     "",
					DB:           0,
					PoolSize:     10,
					MinIdleConns: 5,
					MaxRetries:   3,
					DialTimeout:  1,
					ReadTimeout:  1,
					WriteTimeout: 1,
					IdleTimeout:  300,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				defer Close()
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	// 测试未初始化的情况
	Client = nil
	client := GetClient()
	assert.Nil(t, client)
}

func TestClose(t *testing.T) {
	// 测试关闭连接
	err := Close()
	assert.NoError(t, err)

	// 测试Client为nil的情况
	Client = nil
	err = Close()
	assert.NoError(t, err)
}

func setupTestRedis(t *testing.T) *config.Config {
	return &config.Config{
		Redis: config.RedisConfig{
			Host:         "127.0.0.1",
			Port:         6379,
			Password:     "",
			DB:           1, // 使用测试数据库
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			DialTimeout:  5,
			ReadTimeout:  3,
			WriteTimeout: 3,
			IdleTimeout:  300,
		},
	}
}

func TestNewLuaScript(t *testing.T) {
	// 测试创建Lua脚本管理器
	ls := NewLuaScript(nil)
	assert.NotNil(t, ls)
	assert.NotNil(t, ls.sha1)
}

func TestLuaScriptLoadScripts(t *testing.T) {
	// 这个测试需要真实的Redis连接
	t.Skip("Integration test requires real Redis setup")

	ctx := context.Background()
	ls := NewLuaScript(Client)

	err := ls.LoadScripts(ctx)
	assert.NoError(t, err)

	// 检查脚本是否加载
	assert.NotEmpty(t, ls.sha1["stock_deduct"])
	assert.NotEmpty(t, ls.sha1["stock_revert"])
	assert.NotEmpty(t, ls.sha1["rate_limit"])
	assert.NotEmpty(t, ls.sha1["distributed_lock"])
	assert.NotEmpty(t, ls.sha1["release_lock"])
}

func TestStockDeductWithoutScripts(t *testing.T) {
	// 测试未加载脚本时的情况
	originalLuaScripts := LuaScripts
	defer func() { LuaScripts = originalLuaScripts }()

	LuaScripts = nil
	ctx := context.Background()

	code, message, stock, err := DeductStock(ctx, "1", "1", "1", "req1", 1, 5, time.Hour)
	assert.Error(t, err)
	assert.Equal(t, -1, code)
	assert.Equal(t, "lua scripts not initialized", message)
	assert.Equal(t, 0, stock)
}

func TestStockRevertWithoutScripts(t *testing.T) {
	// 测试未加载脚本时的情况
	originalLuaScripts := LuaScripts
	defer func() { LuaScripts = originalLuaScripts }()

	LuaScripts = nil
	ctx := context.Background()

	code, message, err := RevertStock(ctx, "1", "1", "1", "req1", 1, time.Hour)
	assert.Error(t, err)
	assert.Equal(t, -1, code)
	assert.Equal(t, "lua scripts not initialized", message)
}

func TestCheckRateLimitWithoutScripts(t *testing.T) {
	// 测试未加载脚本时的情况
	originalLuaScripts := LuaScripts
	defer func() { LuaScripts = originalLuaScripts }()

	LuaScripts = nil
	ctx := context.Background()

	allowed, remaining, err := CheckRateLimit(ctx, "test", time.Minute, 10)
	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
}

func TestTryLockWithoutScripts(t *testing.T) {
	// 测试未加载脚本时的情况
	originalLuaScripts := LuaScripts
	defer func() { LuaScripts = originalLuaScripts }()

	LuaScripts = nil
	ctx := context.Background()

	acquired, err := TryLock(ctx, "lock_key", "value", time.Minute)
	assert.Error(t, err)
	assert.False(t, acquired)
}

func TestUnlockWithoutScripts(t *testing.T) {
	// 测试未加载脚本时的情况
	originalLuaScripts := LuaScripts
	defer func() { LuaScripts = originalLuaScripts }()

	LuaScripts = nil
	ctx := context.Background()

	released, err := Unlock(ctx, "lock_key", "value")
	assert.Error(t, err)
	assert.False(t, released)
}

func TestInitStockWithoutClient(t *testing.T) {
	// 测试未初始化客户端时的情况
	originalClient := Client
	defer func() { Client = originalClient }()

	Client = nil
	ctx := context.Background()

	err := InitStock(ctx, "1", "1", 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis client not initialized")
}

func TestGetStockWithoutClient(t *testing.T) {
	// 测试未初始化客户端时的情况
	originalClient := Client
	defer func() { Client = originalClient }()

	Client = nil
	ctx := context.Background()

	stock, err := GetStock(ctx, "1", "1")
	assert.Error(t, err)
	assert.Equal(t, 0, stock)
	assert.Contains(t, err.Error(), "redis client not initialized")
}

func TestGetUserBoughtWithoutClient(t *testing.T) {
	// 测试未初始化客户端时的情况
	originalClient := Client
	defer func() { Client = originalClient }()

	Client = nil
	ctx := context.Background()

	bought, err := GetUserBought(ctx, "1", "1")
	assert.Error(t, err)
	assert.Equal(t, 0, bought)
	assert.Contains(t, err.Error(), "redis client not initialized")
}

func TestLuaScriptMethods(t *testing.T) {
	// 测试Lua脚本方法（不需要真实Redis连接）
	ls := NewLuaScript(nil)

	ctx := context.Background()

	// 测试未加载脚本的情况
	code, message, stock, err := ls.StockDeduct(ctx, "1", "1", "1", "req1", 1, 5, time.Hour)
	assert.Error(t, err)
	assert.Equal(t, -1, code)
	assert.Equal(t, "script not loaded", message)
	assert.Equal(t, 0, stock)

	code2, message2, err2 := ls.StockRevert(ctx, "1", "1", "1", "req1", 1, time.Hour)
	assert.Error(t, err2)
	assert.Equal(t, -1, code2)
	assert.Equal(t, "script not loaded", message2)

	allowed, remaining, err3 := ls.RateLimit(ctx, "test", time.Minute, 10)
	assert.Error(t, err3)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)

	acquired, err4 := ls.AcquireLock(ctx, "lock", "value", time.Minute)
	assert.Error(t, err4)
	assert.False(t, acquired)

	released, err5 := ls.ReleaseLock(ctx, "lock", "value")
	assert.Error(t, err5)
	assert.False(t, released)
}

func TestInitLuaScripts(t *testing.T) {
	// 这个测试需要真实的Redis连接
	t.Skip("Integration test requires real Redis setup")

	err := InitLuaScripts(Client)
	assert.NoError(t, err)
	assert.NotNil(t, LuaScripts)
}

// 集成测试（需要真实的Redis环境）
func TestLuaScriptIntegration(t *testing.T) {
	t.Skip("Integration test requires real Redis setup")

	ctx := context.Background()

	// 初始化Redis和Lua脚本
	cfg := setupTestRedis(t)
	err := Init(cfg)
	assert.NoError(t, err)
	defer Close()

	err = InitLuaScripts(Client)
	assert.NoError(t, err)

	// 测试库存操作
	activityID := "test_activity"
	goodsID := "test_goods"
	userID := "test_user"
	requestID := "test_request"

	// 初始化库存
	err = InitStock(ctx, activityID, goodsID, 100)
	assert.NoError(t, err)

	// 测试库存扣减
	code, message, stock, err := DeductStock(ctx, activityID, goodsID, userID, requestID, 5, 10, time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "success", message)
	assert.Equal(t, 95, stock)

	// 测试重复请求
	code, message, stock, err = DeductStock(ctx, activityID, goodsID, userID, requestID, 5, 10, time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, -4, code)
	assert.Equal(t, "duplicate request", message)

	// 测试库存回滚
	code2, message2, err2 := RevertStock(ctx, activityID, goodsID, userID, requestID, 5, time.Hour)
	assert.NoError(t, err2)
	assert.Equal(t, 0, code2)
	assert.Equal(t, "success", message2)

	// 验证库存恢复
	currentStock, err := GetStock(ctx, activityID, goodsID)
	assert.NoError(t, err)
	assert.Equal(t, 100, currentStock)

	// 测试限流
	allowed, remaining, err := CheckRateLimit(ctx, "test_rate_limit", time.Minute, 5)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 4, remaining)

	// 测试分布式锁
	acquired, err := TryLock(ctx, "test_lock", "lock_value", time.Minute)
	assert.NoError(t, err)
	assert.True(t, acquired)

	// 测试释放锁
	released, err := Unlock(ctx, "test_lock", "lock_value")
	assert.NoError(t, err)
	assert.True(t, released)

	// 清理测试数据
	if Client == nil {
		assert.Fail(t, "redis client not initialized")
		return
	}
	Client.Del(ctx, "stock:"+activityID+":"+goodsID)
	Client.Del(ctx, "user_buy:"+activityID+":"+userID)
	Client.Del(ctx, "request:"+requestID)
	Client.Del(ctx, "test_rate_limit")
	Client.Del(ctx, "test_lock")
}

func BenchmarkLuaScripts(b *testing.B) {
	b.Skip("Benchmark requires real Redis setup")

	ctx := context.Background()

	b.Run("StockDeduct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			DeductStock(ctx, "bench_activity", "bench_goods", "bench_user", "bench_req", 1, 10, time.Hour)
		}
	})

	b.Run("RateLimit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CheckRateLimit(ctx, "bench_rate_limit", time.Minute, 100)
		}
	})
}
