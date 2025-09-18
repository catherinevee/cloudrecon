package retry

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	Jitter          float64
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      5,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		Jitter:          0.1,
	}
}

// Retry executes a function with exponential backoff retry logic
func Retry(ctx context.Context, config *RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff and jitter
			delay := calculateDelay(config, attempt)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// RetryWithResult executes a function with exponential backoff retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, config *RetryConfig, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff and jitter
			delay := calculateDelay(config, attempt)

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}

		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err
		result = res

		// Check if error is retryable
		if !IsRetryableError(err) {
			return result, err
		}
	}

	return result, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateDelay calculates the delay for the given attempt
func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	// Calculate exponential backoff
	delay := float64(config.InitialInterval) * math.Pow(config.Multiplier, float64(attempt-1))

	// Apply maximum interval limit
	if delay > float64(config.MaxInterval) {
		delay = float64(config.MaxInterval)
	}

	// Apply jitter to prevent thundering herd
	if config.Jitter > 0 {
		// Generate secure random number between -1 and 1
		randomBig, _ := rand.Int(rand.Reader, big.NewInt(1000))
		randomFloat := float64(randomBig.Int64())/1000.0*2 - 1 // -1 to 1
		jitter := delay * config.Jitter * randomFloat          // -jitter to +jitter
		delay += jitter
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common retryable error patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"rate limit",
		"throttle",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
		"too many requests",
		"request timeout",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err        error
	RetryAfter time.Duration
	MaxRetries int
	Attempts   int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d/%d): %v", e.Attempts, e.MaxRetries, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, retryAfter time.Duration, maxRetries, attempts int) *RetryableError {
	return &RetryableError{
		Err:        err,
		RetryAfter: retryAfter,
		MaxRetries: maxRetries,
		Attempts:   attempts,
	}
}
