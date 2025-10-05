package breaker

import (
	"context"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed normal state, requests are allowed
	StateClosed State = iota
	// StateOpen circuit is open, requests are blocked
	StateOpen
	// StateHalfOpen circuit is half-open, testing if service recovered
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config circuit breaker configuration
type Config struct {
	// MaxRequests maximum requests allowed in half-open state
	MaxRequests uint32
	// Interval time window for error rate calculation
	Interval time.Duration
	// Timeout timeout before transitioning from open to half-open
	Timeout time.Duration
	// ReadyToTrip function to determine if circuit should open
	ReadyToTrip func(counts Counts) bool
	// OnStateChange callback when state changes
	OnStateChange func(name string, from State, to State)
}

// Counts holds the numbers of requests and their outcomes
type Counts struct {
	Requests       uint32
	TotalSuccesses uint32
	TotalFailures  uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker circuit breaker implementation
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mu          sync.Mutex
	state       State
	generation  uint64
	counts      Counts
	expiry      time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		onStateChange: config.OnStateChange,
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}

	if cb.interval == 0 {
		cb.interval = time.Minute
	}

	if cb.timeout == 0 {
		cb.timeout = time.Minute
	}

	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		}
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Execute executes the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	err = fn()
	cb.afterRequest(generation, err == nil)
	return err
}

// Call is an alias for Execute
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	return cb.Execute(ctx, fn)
}

// State returns the current state
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.counts
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.Requests++
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if state == StateHalfOpen {
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.toNewGeneration(time.Now())
	cb.state = StateClosed
}

var (
	// ErrOpenState circuit breaker is open
	ErrOpenState = &CircuitBreakerError{message: "circuit breaker is open"}
	// ErrTooManyRequests too many requests in half-open state
	ErrTooManyRequests = &CircuitBreakerError{message: "too many requests"}
)

// CircuitBreakerError circuit breaker error
type CircuitBreakerError struct {
	message string
}

func (e *CircuitBreakerError) Error() string {
	return e.message
}

// IsCircuitBreakerError checks if error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	_, ok := err.(*CircuitBreakerError)
	return ok
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers sync.Map
	config   Config
}

// NewManager creates a new circuit breaker manager
func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
	}
}

// GetBreaker gets or creates a circuit breaker
func (m *Manager) GetBreaker(name string) *CircuitBreaker {
	if cb, ok := m.breakers.Load(name); ok {
		return cb.(*CircuitBreaker)
	}

	cb := NewCircuitBreaker(name, m.config)
	actual, loaded := m.breakers.LoadOrStore(name, cb)
	if loaded {
		return actual.(*CircuitBreaker)
	}
	return cb
}

// Execute executes the given function with the named circuit breaker
func (m *Manager) Execute(ctx context.Context, name string, fn func() error) error {
	cb := m.GetBreaker(name)
	return cb.Execute(ctx, fn)
}

// State returns the state of the named circuit breaker
func (m *Manager) State(name string) State {
	cb := m.GetBreaker(name)
	return cb.State()
}

// DefaultManager default circuit breaker manager
var DefaultManager = NewManager(Config{
	MaxRequests:   5,
	Interval:      time.Minute,
	Timeout:       30 * time.Second,
	ReadyToTrip:   nil, // Use default
	OnStateChange: nil,
})

// Execute executes the given function with the default manager
func Execute(ctx context.Context, name string, fn func() error) error {
	return DefaultManager.Execute(ctx, name, fn)
}

