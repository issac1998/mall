package lock

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrLockNotAcquired = errors.New("lock not acquired")
	ErrLockNotHeld     = errors.New("lock not held")
)

// RedisLock represents a distributed lock using Redis
type RedisLock struct {
	client redis.Cmdable
	key    string
	value  string
	ttl    time.Duration
}

// NewRedisLock creates a new Redis lock
func NewRedisLock(client redis.Cmdable, key, value string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		client: client,
		key:    key,
		value:  value,
		ttl:    ttl,
	}
}

// Lock acquires the lock
func (l *RedisLock) Lock(ctx context.Context) error {
	success, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return err
	}

	if !success {
		return ErrLockNotAcquired
	}

	return nil
}

// TryLock tries to acquire the lock with retries
func (l *RedisLock) TryLock(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := l.Lock(ctx)
		if err == nil {
			return nil
		}

		if err != ErrLockNotAcquired {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			// Continue to next retry
		}
	}

	return ErrLockNotAcquired
}

// Unlock releases the lock
func (l *RedisLock) Unlock(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Extend extends the lock TTL
func (l *RedisLock) Extend(ctx context.Context, ttl time.Duration) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, int(ttl.Milliseconds())).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// IsHeld checks if the lock is held
func (l *RedisLock) IsHeld(ctx context.Context) (bool, error) {
	value, err := l.client.Get(ctx, l.key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return value == l.value, nil
}

