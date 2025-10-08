package lock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedis(t *testing.T) *redis.Client {
	s, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
		s.Close()
	})

	return client
}

func TestRedisLock(t *testing.T) {
	client := setupRedis(t)
	ctx := context.Background()

	t.Run("BasicLockUnlock", func(t *testing.T) {
		lock := NewRedisLock(client, "test_lock", "value1", time.Minute)

		// Should be able to acquire lock
		err := lock.Lock(ctx)
		assert.NoError(t, err)

		// Should be held
		held, err := lock.IsHeld(ctx)
		assert.NoError(t, err)
		assert.True(t, held)

		// Should be able to unlock
		err = lock.Unlock(ctx)
		assert.NoError(t, err)

		// Should not be held after unlock
		held, err = lock.IsHeld(ctx)
		assert.NoError(t, err)
		assert.False(t, held)
	})

	t.Run("LockConflict", func(t *testing.T) {
		lock1 := NewRedisLock(client, "conflict_lock", "value1", time.Minute)
		lock2 := NewRedisLock(client, "conflict_lock", "value2", time.Minute)

		// First lock should succeed
		err := lock1.Lock(ctx)
		assert.NoError(t, err)

		// Second lock should fail
		err = lock2.Lock(ctx)
		assert.Equal(t, ErrLockNotAcquired, err)

		// First lock should still be held
		held, err := lock1.IsHeld(ctx)
		assert.NoError(t, err)
		assert.True(t, held)

		// Second lock should not be held
		held, err = lock2.IsHeld(ctx)
		assert.NoError(t, err)
		assert.False(t, held)

		// Unlock first lock
		err = lock1.Unlock(ctx)
		assert.NoError(t, err)

		// Now second lock should succeed
		err = lock2.Lock(ctx)
		assert.NoError(t, err)

		// Clean up
		err = lock2.Unlock(ctx)
		assert.NoError(t, err)
	})

	t.Run("TryLockWithRetries", func(t *testing.T) {
		lock1 := NewRedisLock(client, "retry_lock", "value1", time.Minute)
		lock2 := NewRedisLock(client, "retry_lock", "value2", time.Minute)

		// First lock acquires
		err := lock1.Lock(ctx)
		assert.NoError(t, err)

		// Second lock tries with retries (should fail)
		start := time.Now()
		err = lock2.TryLock(ctx, 2, 50*time.Millisecond)
		duration := time.Since(start)

		// Should fail but take some time due to retries
		assert.Equal(t, ErrLockNotHeld, err)
		assert.True(t, duration >= 50*time.Millisecond, "Should have waited for retries")

		// Unlock first lock
		err = lock1.Unlock(ctx)
		assert.NoError(t, err)

		// Now second lock should succeed
		err = lock2.Lock(ctx)
		assert.NoError(t, err)

		// Clean up
		err = lock2.Unlock(ctx)
		assert.NoError(t, err)
	})

	t.Run("ExtendLock", func(t *testing.T) {
		lock := NewRedisLock(client, "extend_lock", "value1", 100*time.Millisecond)

		// Acquire lock
		err := lock.Lock(ctx)
		assert.NoError(t, err)

		// Extend the lock
		err = lock.Extend(ctx, time.Minute)
		assert.NoError(t, err)

		// Should still be held after original TTL
		time.Sleep(150 * time.Millisecond)
		held, err := lock.IsHeld(ctx)
		assert.NoError(t, err)
		assert.True(t, held, "Lock should still be held after extension")

		// Clean up
		err = lock.Unlock(ctx)
		assert.NoError(t, err)
	})

	t.Run("ExtendNonHeldLock", func(t *testing.T) {
		lock := NewRedisLock(client, "extend_fail_lock", "value1", time.Minute)

		// Try to extend without holding the lock
		err := lock.Extend(ctx, time.Minute)
		assert.Equal(t, ErrLockNotHeld, err)
	})

	t.Run("UnlockNonHeldLock", func(t *testing.T) {
		lock := NewRedisLock(client, "unlock_fail_lock", "value1", time.Minute)

		// Try to unlock without holding the lock
		err := lock.Unlock(ctx)
		assert.Equal(t, ErrLockNotHeld, err)
	})

	t.Run("UnlockWrongValue", func(t *testing.T) {
		lock1 := NewRedisLock(client, "wrong_value_lock", "value1", time.Minute)
		lock2 := NewRedisLock(client, "wrong_value_lock", "value2", time.Minute)

		// First lock acquires
		err := lock1.Lock(ctx)
		assert.NoError(t, err)

		// Second lock tries to unlock (wrong value)
		err = lock2.Unlock(ctx)
		assert.Equal(t, ErrLockNotHeld, err)

		// First lock should still be held
		held, err := lock1.IsHeld(ctx)
		assert.NoError(t, err)
		assert.True(t, held)

		// First lock can unlock successfully
		err = lock1.Unlock(ctx)
		assert.NoError(t, err)
	})

	t.Run("LockExpiration", func(t *testing.T) {
		// Skip this test as miniredis may not handle TTL expiration exactly like real Redis
		t.Skip("Skipping TTL expiration test with miniredis")
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		lock1 := NewRedisLock(client, "context_lock", "value1", time.Minute)
		lock2 := NewRedisLock(client, "context_lock", "value2", time.Minute)

		// First lock acquires
		err := lock1.Lock(ctx)
		assert.NoError(t, err)

		// Create cancelled context
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		// Second lock should fail due to cancelled context
		err = lock2.TryLock(cancelCtx, 3, 50*time.Millisecond)
		assert.Equal(t, context.Canceled, err)

		// Clean up
		err = lock1.Unlock(ctx)
		assert.NoError(t, err)
	})
}

func TestRedisLockInterface(t *testing.T) {
	client := setupRedis(t)

	t.Run("ImplementsLockInterface", func(t *testing.T) {
		var _ interface {
			Lock(context.Context) error
			Unlock(context.Context) error
			TryLock(context.Context, int, time.Duration) error
			Extend(context.Context, time.Duration) error
			IsHeld(context.Context) (bool, error)
		} = NewRedisLock(client, "test", "value", time.Minute)
	})
}
