package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lua脚本常量
const (
	// 库存扣减脚本
	StockDeductScript = `
		local activity_id = KEYS[1]
		local goods_id = KEYS[2]
		local user_id = KEYS[3]
		local request_id = KEYS[4]
		
		local quantity = tonumber(ARGV[1])
		local limit_per_user = tonumber(ARGV[2])
		local expire_time = tonumber(ARGV[3])
		
		-- 检查库存
		local stock_key = "stock:" .. activity_id .. ":" .. goods_id
		local current_stock = redis.call('GET', stock_key)
		if not current_stock then
			return {-1, "stock not found"}
		end
		
		current_stock = tonumber(current_stock)
		if current_stock < quantity then
			return {-2, "insufficient stock"}
		end
		
		-- 检查用户购买限制
		local user_buy_key = "user_buy:" .. activity_id .. ":" .. user_id
		local user_bought = redis.call('GET', user_buy_key)
		if not user_bought then
			user_bought = 0
		else
			user_bought = tonumber(user_bought)
		end
		
		if user_bought + quantity > limit_per_user then
			return {-3, "exceed user limit"}
		end
		
		-- 检查请求是否已处理（防重复）
		local request_key = "request:" .. request_id
		local request_exists = redis.call('EXISTS', request_key)
		if request_exists == 1 then
			return {-4, "duplicate request"}
		end
		
		-- 扣减库存
		local new_stock = redis.call('DECRBY', stock_key, quantity)
		if new_stock < 0 then
			-- 回滚库存
			redis.call('INCRBY', stock_key, quantity)
			return {-2, "insufficient stock"}
		end
		
		-- 更新用户购买数量
		redis.call('INCRBY', user_buy_key, quantity)
		redis.call('EXPIRE', user_buy_key, expire_time)
		
		-- 标记请求已处理
		redis.call('SETEX', request_key, expire_time, 1)
		
		-- 记录扣减日志
		local log_key = "stock_log:" .. activity_id .. ":" .. goods_id
		local log_data = user_id .. ":" .. quantity .. ":" .. os.time()
		redis.call('LPUSH', log_key, log_data)
		redis.call('EXPIRE', log_key, expire_time)
		
		return {0, "success", new_stock}
	`

	// 库存回滚脚本
	StockRevertScript = `
		local activity_id = KEYS[1]
		local goods_id = KEYS[2]
		local user_id = KEYS[3]
		local request_id = KEYS[4]
		
		local quantity = tonumber(ARGV[1])
		local expire_time = tonumber(ARGV[2])
		
		-- 检查请求是否存在
		local request_key = "request:" .. request_id
		local request_exists = redis.call('EXISTS', request_key)
		if request_exists == 0 then
			return {-1, "request not found"}
		end
		
		-- 回滚库存
		local stock_key = "stock:" .. activity_id .. ":" .. goods_id
		redis.call('INCRBY', stock_key, quantity)
		
		-- 回滚用户购买数量
		local user_buy_key = "user_buy:" .. activity_id .. ":" .. user_id
		redis.call('DECRBY', user_buy_key, quantity)
		
		-- 删除请求标记
		redis.call('DEL', request_key)
		
		-- 记录回滚日志
		local log_key = "stock_log:" .. activity_id .. ":" .. goods_id
		local log_data = user_id .. ":-" .. quantity .. ":" .. os.time()
		redis.call('LPUSH', log_key, log_data)
		redis.call('EXPIRE', log_key, expire_time)
		
		return {0, "success"}
	`

	// 限流脚本（滑动窗口）
	RateLimitScript = `
		local key = KEYS[1]
		local window = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		local current_time = tonumber(ARGV[3])
		
		-- 清理过期数据
		redis.call('ZREMRANGEBYSCORE', key, 0, current_time - window)
		
		-- 获取当前窗口内的请求数
		local current_count = redis.call('ZCARD', key)
		
		if current_count < limit then
			-- 添加当前请求
			redis.call('ZADD', key, current_time, current_time)
			redis.call('EXPIRE', key, window)
			return {0, limit - current_count - 1}
		else
			return {-1, 0}
		end
	`

	// 分布式锁脚本
	DistributedLockScript = `
		local key = KEYS[1]
		local value = ARGV[1]
		local expire = tonumber(ARGV[2])
		
		-- 尝试获取锁
		local result = redis.call('SET', key, value, 'PX', expire, 'NX')
		if result then
			return 1
		else
			return 0
		end
	`

	// 释放分布式锁脚本
	ReleaseLockScript = `
		local key = KEYS[1]
		local value = ARGV[1]
		
		-- 检查锁的值是否匹配
		local current_value = redis.call('GET', key)
		if current_value == value then
			return redis.call('DEL', key)
		else
			return 0
		end
	`
)

// LuaScript Lua脚本管理器
type LuaScript struct {
	client redis.Cmdable
	
	stockDeductScript *redis.Script
	stockRevertScript *redis.Script
	
	rateLimitScript *redis.Script
	
	lockScript   *redis.Script
	unlockScript *redis.Script
}

// NewLuaScript 创建Lua脚本管理器
func NewLuaScript(client redis.Cmdable) *LuaScript {
	return &LuaScript{
		client:            client,
		stockDeductScript: redis.NewScript(StockDeductScript),
		stockRevertScript: redis.NewScript(StockRevertScript),
		rateLimitScript:   redis.NewScript(RateLimitScript),
		lockScript:        redis.NewScript(DistributedLockScript),
		unlockScript:      redis.NewScript(ReleaseLockScript),
	}
}

// LoadScripts 预加载所有脚本
func (ls *LuaScript) LoadScripts(ctx context.Context) error {
	scripts := []*redis.Script{
		ls.stockDeductScript,
		ls.stockRevertScript,
		ls.rateLimitScript,
		ls.lockScript,
		ls.unlockScript,
	}
	
	for _, script := range scripts {
		if err := script.Load(ctx, ls.client).Err(); err != nil {
			return fmt.Errorf("failed to load lua script: %w", err)
		}
	}
	
	return nil
}

// StockDeduct 库存扣减
func (ls *LuaScript) StockDeduct(ctx context.Context, activityID, goodsID, userID, requestID string, quantity, limitPerUser int, expireTime time.Duration) (int, string, int, error) {
	keys := []string{activityID, goodsID, userID, requestID}
	args := []interface{}{quantity, limitPerUser, int(expireTime.Seconds())}

	result, err := ls.stockDeductScript.Run(ctx, ls.client, keys, args...).Result()
	if err != nil {
		return -1, "script error", 0, err
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 2 {
		return -1, "invalid result", 0, fmt.Errorf("invalid script result")
	}

	code, _ := resultSlice[0].(int64)
	message, _ := resultSlice[1].(string)
	
	var stock int
	if len(resultSlice) > 2 {
		if stockVal, ok := resultSlice[2].(int64); ok {
			stock = int(stockVal)
		}
	}

	return int(code), message, stock, nil
}

// StockRevert 库存回滚
func (ls *LuaScript) StockRevert(ctx context.Context, activityID, goodsID, userID, requestID string, quantity int, expireTime time.Duration) (int, string, error) {
	keys := []string{activityID, goodsID, userID, requestID}
	args := []interface{}{quantity, int(expireTime.Seconds())}

	result, err := ls.stockRevertScript.Run(ctx, ls.client, keys, args...).Result()
	if err != nil {
		return -1, "script error", err
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 2 {
		return -1, "invalid result", fmt.Errorf("invalid script result")
	}

	code, _ := resultSlice[0].(int64)
	message, _ := resultSlice[1].(string)

	return int(code), message, nil
}

// RateLimit 限流检查
func (ls *LuaScript) RateLimit(ctx context.Context, key string, window time.Duration, limit int) (bool, int, error) {
	currentTime := time.Now().Unix()
	keys := []string{key}
	args := []interface{}{int(window.Seconds()), limit, currentTime}

	result, err := ls.rateLimitScript.Run(ctx, ls.client, keys, args...).Result()
	if err != nil {
		return false, 0, err
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 2 {
		return false, 0, fmt.Errorf("invalid script result")
	}

	code, _ := resultSlice[0].(int64)
	remaining, _ := resultSlice[1].(int64)

	return code == 0, int(remaining), nil
}

// AcquireLock 获取分布式锁
func (ls *LuaScript) AcquireLock(ctx context.Context, key, value string, expire time.Duration) (bool, error) {
	keys := []string{key}
	args := []interface{}{value, int(expire.Milliseconds())}

	result, err := ls.lockScript.Run(ctx, ls.client, keys, args...).Result()
	if err != nil {
		return false, err
	}

	code, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("invalid script result")
	}

	return code == 1, nil
}

// ReleaseLock 释放分布式锁
func (ls *LuaScript) ReleaseLock(ctx context.Context, key, value string) (bool, error) {
	keys := []string{key}
	args := []interface{}{value}

	result, err := ls.unlockScript.Run(ctx, ls.client, keys, args...).Result()
	if err != nil {
		return false, err
	}

	code, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("invalid script result")
	}

	return code == 1, nil
}

// 全局Lua脚本管理器
var LuaScripts *LuaScript

// InitLuaScripts 初始化Lua脚本
func InitLuaScripts(client redis.Cmdable) error {
	LuaScripts = NewLuaScript(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return LuaScripts.LoadScripts(ctx)
}

// 便捷函数

// DeductStock 扣减库存
func DeductStock(ctx context.Context, activityID, goodsID, userID, requestID string, quantity, limitPerUser int, expireTime time.Duration) (int, string, int, error) {
	if LuaScripts == nil {
		return -1, "lua scripts not initialized", 0, fmt.Errorf("lua scripts not initialized")
	}
	return LuaScripts.StockDeduct(ctx, activityID, goodsID, userID, requestID, quantity, limitPerUser, expireTime)
}

// RevertStock 回滚库存
func RevertStock(ctx context.Context, activityID, goodsID, userID, requestID string, quantity int, expireTime time.Duration) (int, string, error) {
	if LuaScripts == nil {
		return -1, "lua scripts not initialized", fmt.Errorf("lua scripts not initialized")
	}
	return LuaScripts.StockRevert(ctx, activityID, goodsID, userID, requestID, quantity, expireTime)
}

// CheckRateLimit 检查限流
func CheckRateLimit(ctx context.Context, key string, window time.Duration, limit int) (bool, int, error) {
	if LuaScripts == nil {
		return false, 0, fmt.Errorf("lua scripts not initialized")
	}
	return LuaScripts.RateLimit(ctx, key, window, limit)
}

// TryLock 尝试获取锁
func TryLock(ctx context.Context, key, value string, expire time.Duration) (bool, error) {
	if LuaScripts == nil {
		return false, fmt.Errorf("lua scripts not initialized")
	}
	return LuaScripts.AcquireLock(ctx, key, value, expire)
}

// Unlock 释放锁
func Unlock(ctx context.Context, key, value string) (bool, error) {
	if LuaScripts == nil {
		return false, fmt.Errorf("lua scripts not initialized")
	}
	return LuaScripts.ReleaseLock(ctx, key, value)
}

// 辅助函数

// InitStock 初始化库存
func InitStock(ctx context.Context, activityID, goodsID string, stock int) error {
	if Client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	key := fmt.Sprintf("stock:%s:%s", activityID, goodsID)
	return Client.Set(ctx, key, stock, 0).Err()
}

// GetStock 获取当前库存
func GetStock(ctx context.Context, activityID, goodsID string) (int, error) {
	if Client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	
	key := fmt.Sprintf("stock:%s:%s", activityID, goodsID)
	result, err := Client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	
	return strconv.Atoi(result)
}

// GetUserBought 获取用户已购买数量
func GetUserBought(ctx context.Context, activityID, userID string) (int, error) {
	if Client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	
	key := fmt.Sprintf("user_buy:%s:%s", activityID, userID)
	result, err := Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	
	return strconv.Atoi(result)
}