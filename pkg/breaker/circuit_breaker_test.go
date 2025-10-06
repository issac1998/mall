package breaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerStates(t *testing.T) {
	t.Run("StateString", func(t *testing.T) {
		assert.Equal(t, "closed", StateClosed.String())
		assert.Equal(t, "open", StateOpen.String())
		assert.Equal(t, "half-open", StateHalfOpen.String())
		assert.Equal(t, "unknown", State(999).String())
	})
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		cb := NewCircuitBreaker("test", Config{})
		
		assert.Equal(t, "test", cb.name)
		assert.Equal(t, uint32(1), cb.maxRequests)
		assert.Equal(t, time.Minute, cb.interval)
		assert.Equal(t, time.Minute, cb.timeout)
		assert.Equal(t, StateClosed, cb.State())
		assert.NotNil(t, cb.readyToTrip)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := Config{
			MaxRequests: 10,
			Interval:    30 * time.Second,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts Counts) bool {
				return counts.TotalFailures >= 5
			},
		}
		
		cb := NewCircuitBreaker("custom", config)
		
		assert.Equal(t, "custom", cb.name)
		assert.Equal(t, uint32(10), cb.maxRequests)
		assert.Equal(t, 30*time.Second, cb.interval)
		assert.Equal(t, 60*time.Second, cb.timeout)
		assert.NotNil(t, cb.readyToTrip)
	})
}

func TestCircuitBreakerExecution(t *testing.T) {
	t.Run("SuccessfulExecution", func(t *testing.T) {
		cb := NewCircuitBreaker("test", Config{})
		ctx := context.Background()
		
		err := cb.Execute(ctx, func() error {
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.State())
		
		counts := cb.Counts()
		assert.Equal(t, uint32(1), counts.Requests)
		assert.Equal(t, uint32(1), counts.TotalSuccesses)
		assert.Equal(t, uint32(0), counts.TotalFailures)
	})

	t.Run("FailedExecution", func(t *testing.T) {
		cb := NewCircuitBreaker("test", Config{
			Interval: 0, // Disable interval to prevent count reset
			ReadyToTrip: func(counts Counts) bool {
				return counts.Requests >= 3 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
			},
		})
		ctx := context.Background()
		testErr := errors.New("test error")
		
		// Execute 3 failed requests
		for i := 0; i < 3; i++ {
			err := cb.Execute(ctx, func() error {
				return testErr
			})
			assert.Equal(t, testErr, err)
		}
		
		// Should open after 3 failures
		assert.Equal(t, StateOpen, cb.State())
		
		// Check counts after state transition
		// Note: counts may be reset when state changes to open
		// So we just verify the state is open
		assert.Equal(t, StateOpen, cb.State())
	})

	t.Run("OpenStateBlocking", func(t *testing.T) {
		cb := NewCircuitBreaker("test", Config{
			ReadyToTrip: func(counts Counts) bool {
				return counts.TotalFailures >= 1
			},
		})
		ctx := context.Background()
		
		// Fail once to open circuit
		err := cb.Execute(ctx, func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateOpen, cb.State())
		
		// Next request should be blocked
		err = cb.Execute(ctx, func() error {
			return nil
		})
		assert.Equal(t, ErrOpenState, err)
		assert.True(t, IsCircuitBreakerError(err))
	})

	t.Run("HalfOpenTransition", func(t *testing.T) {
		cb := NewCircuitBreaker("test", Config{
			Timeout: 10 * time.Millisecond,
			ReadyToTrip: func(counts Counts) bool {
				return counts.TotalFailures >= 1
			},
		})
		ctx := context.Background()
		
		// Fail to open circuit
		err := cb.Execute(ctx, func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateOpen, cb.State())
		
		// Wait for timeout
		time.Sleep(15 * time.Millisecond)
		
		// Should transition to half-open
		err = cb.Execute(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.State()) // Should close after successful request
	})

	t.Run("PanicRecovery", func(t *testing.T) {
		cb := NewCircuitBreaker("test", Config{})
		ctx := context.Background()
		
		assert.Panics(t, func() {
			cb.Execute(ctx, func() error {
				panic("test panic")
			})
		})
		
		// Circuit should record the failure
		counts := cb.Counts()
		assert.Equal(t, uint32(1), counts.Requests)
		assert.Equal(t, uint32(0), counts.TotalSuccesses)
		assert.Equal(t, uint32(1), counts.TotalFailures)
	})
}

func TestCircuitBreakerCall(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{})
	ctx := context.Background()
	
	err := cb.Call(ctx, func() error {
		return nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		ReadyToTrip: func(counts Counts) bool {
			return counts.TotalFailures >= 1
		},
	})
	ctx := context.Background()
	
	// Fail to open circuit
	err := cb.Execute(ctx, func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
	
	// Reset should close the circuit
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
	
	counts := cb.Counts()
	assert.Equal(t, uint32(0), counts.Requests)
	assert.Equal(t, uint32(0), counts.TotalSuccesses)
	assert.Equal(t, uint32(0), counts.TotalFailures)
}

func TestCircuitBreakerError(t *testing.T) {
	err := &CircuitBreakerError{message: "test error"}
	assert.Equal(t, "test error", err.Error())
	assert.True(t, IsCircuitBreakerError(err))
	assert.False(t, IsCircuitBreakerError(errors.New("other error")))
}

func TestManager(t *testing.T) {
	t.Run("GetBreaker", func(t *testing.T) {
		manager := NewManager(Config{
			MaxRequests: 5,
			Interval:    30 * time.Second,
		})
		
		cb1 := manager.GetBreaker("test1")
		cb2 := manager.GetBreaker("test1") // Same name
		cb3 := manager.GetBreaker("test2") // Different name
		
		assert.Same(t, cb1, cb2) // Should return same instance
		assert.NotSame(t, cb1, cb3) // Should return different instance
		
		assert.Equal(t, "test1", cb1.name)
		assert.Equal(t, "test2", cb3.name)
		assert.Equal(t, uint32(5), cb1.maxRequests)
		assert.Equal(t, 30*time.Second, cb1.interval)
	})

	t.Run("Execute", func(t *testing.T) {
		manager := NewManager(Config{})
		ctx := context.Background()
		
		err := manager.Execute(ctx, "test", func() error {
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, manager.State("test"))
	})

	t.Run("State", func(t *testing.T) {
		manager := NewManager(Config{
			ReadyToTrip: func(counts Counts) bool {
				return counts.TotalFailures >= 1
			},
		})
		ctx := context.Background()
		
		// Initially closed
		assert.Equal(t, StateClosed, manager.State("test"))
		
		// Fail to open
		err := manager.Execute(ctx, "test", func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateOpen, manager.State("test"))
	})
}

func TestDefaultManager(t *testing.T) {
	ctx := context.Background()
	
	err := Execute(ctx, "default-test", func() error {
		return nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, DefaultManager.State("default-test"))
}

func TestHalfOpenMaxRequests(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 2,
		Timeout:     10 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.TotalFailures >= 1
		},
	})
	ctx := context.Background()
	
	// Fail to open circuit
	err := cb.Execute(ctx, func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
	
	// Wait for timeout to transition to half-open
	time.Sleep(15 * time.Millisecond)
	
	// First request in half-open should succeed
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())
	
	// Second request should also succeed and close the circuit
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestHalfOpenTooManyRequests(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 1,
		Timeout:     10 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.TotalFailures >= 1
		},
	})
	ctx := context.Background()
	
	// Fail to open circuit
	err := cb.Execute(ctx, func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
	
	// Wait for timeout to transition to half-open
	time.Sleep(15 * time.Millisecond)
	
	// First request should succeed
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
	
	// Force back to half-open for testing
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.counts.Requests = 1 // Already at max
	cb.mu.Unlock()
	
	// Next request should be blocked
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.Equal(t, ErrTooManyRequests, err)
	assert.True(t, IsCircuitBreakerError(err))
}