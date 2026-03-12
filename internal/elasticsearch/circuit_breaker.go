package elasticsearch

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests are blocked
	StateOpen
	// StateHalfOpen means the circuit is testing if the service recovered
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for ES API calls
type CircuitBreaker struct {
	mu sync.RWMutex

	state         CircuitState
	failureCount  int
	successCount  int
	lastFailTime  time.Time
	lastStateTime time.Time

	// Configuration
	maxFailures  int           // Number of failures before opening
	timeout      time.Duration // Time to wait before half-open
	halfOpenMax  int           // Number of successes needed to close from half-open
	resetTimeout time.Duration // Time to reset failure count in closed state
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		maxFailures:  5,
		timeout:      30 * time.Second,
		halfOpenMax:  2,
		resetTimeout: 60 * time.Second,
	}
}

// Call executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.beforeCall(); err != nil {
		return err
	}

	err := fn(ctx)

	cb.afterCall(err)

	return err
}

// beforeCall checks if the call should be allowed
func (cb *CircuitBreaker) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Reset failure count if enough time has passed
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.failureCount = 0
		}
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastStateTime) > cb.timeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.lastStateTime = time.Now()
			return nil
		}
		return &CircuitBreakerError{
			State:   StateOpen,
			Message: "circuit breaker is open",
		}

	case StateHalfOpen:
		return nil

	default:
		return nil
	}
}

// afterCall updates the circuit breaker state based on the call result
func (cb *CircuitBreaker) afterCall(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles successful calls
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenMax {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
			cb.lastStateTime = time.Now()
		}
	}
}

// onFailure handles failed calls
func (cb *CircuitBreaker) onFailure() {
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.maxFailures {
			cb.state = StateOpen
			cb.lastStateTime = time.Now()
		}

	case StateHalfOpen:
		cb.state = StateOpen
		cb.successCount = 0
		cb.lastStateTime = time.Now()
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateTime = time.Now()
}

// CircuitBreakerError represents a circuit breaker error
type CircuitBreakerError struct {
	State   CircuitState
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker error: %s (state: %d)", e.Message, e.State)
}

// WithCircuitBreaker wraps a Client with circuit breaker protection
type clientWithCircuitBreaker struct {
	Client
	breaker *CircuitBreaker
}

// WithCircuitBreaker wraps an ES client with circuit breaker protection
func WithCircuitBreaker(client Client) Client {
	return &clientWithCircuitBreaker{
		Client:  client,
		breaker: NewCircuitBreaker(),
	}
}

// CreateOrUpdateUser wraps the call with circuit breaker
func (c *clientWithCircuitBreaker) CreateOrUpdateUser(ctx context.Context, req *UserRequest) error {
	return c.breaker.Call(ctx, func(ctx context.Context) error {
		return c.Client.CreateOrUpdateUser(ctx, req)
	})
}

// GetUser wraps the call with circuit breaker
func (c *clientWithCircuitBreaker) GetUser(ctx context.Context, username string) (*User, error) {
	var user *User
	err := c.breaker.Call(ctx, func(ctx context.Context) error {
		var err error
		user, err = c.Client.GetUser(ctx, username)
		return err
	})
	return user, err
}

// DeleteUser wraps the call with circuit breaker
func (c *clientWithCircuitBreaker) DeleteUser(ctx context.Context, username string) error {
	return c.breaker.Call(ctx, func(ctx context.Context) error {
		return c.Client.DeleteUser(ctx, username)
	})
}

// ValidateConnection wraps the call with circuit breaker
func (c *clientWithCircuitBreaker) ValidateConnection(ctx context.Context) error {
	return c.breaker.Call(ctx, func(ctx context.Context) error {
		return c.Client.ValidateConnection(ctx)
	})
}
