package core

import (
	"time"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryableErrors []ErrorType
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrorTypeNetwork,
			ErrorTypeRateLimit,
			ErrorTypeTimeout,
		},
	}
}

// RetryConfig returns a retry policy for a specific operation
func RetryConfig(operation string) *RetryPolicy {
	switch operation {
	case "discovery":
		return &RetryPolicy{
			MaxRetries:    5,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryableErrors: []ErrorType{
				ErrorTypeNetwork,
				ErrorTypeRateLimit,
				ErrorTypeTimeout,
				ErrorTypeProvider,
			},
		}
	case "analysis":
		return &RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  500 * time.Millisecond,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryableErrors: []ErrorType{
				ErrorTypeNetwork,
				ErrorTypeRateLimit,
				ErrorTypeTimeout,
			},
		}
	case "storage":
		return &RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      2 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryableErrors: []ErrorType{
				ErrorTypeStorage,
				ErrorTypeTimeout,
			},
		}
	default:
		return DefaultRetryPolicy()
	}
}

// Retry executes a function with retry logic
func Retry(operation string, fn func() error) error {
	policy := RetryConfig(operation)
	return RetryWithPolicy(policy, fn)
}

// RetryWithPolicy executes a function with a specific retry policy
func RetryWithPolicy(policy *RetryPolicy, fn func() error) error {
	var lastErr error
	delay := policy.InitialDelay

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isErrorRetryable(err, policy) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt == policy.MaxRetries {
			break
		}

		// Calculate delay with jitter
		sleepDuration := delay
		if policy.Jitter {
			sleepDuration = addJitter(delay)
		}

		time.Sleep(sleepDuration)

		// Calculate next delay
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}

	return lastErr
}

// isErrorRetryable checks if an error is retryable according to the policy
func isErrorRetryable(err error, policy *RetryPolicy) bool {
	if !IsRetryableError(err) {
		return false
	}

	// If no specific retryable errors are defined, use default logic
	if len(policy.RetryableErrors) == 0 {
		return true
	}

	// Check if error type is in the retryable list
	if cloudErr, ok := err.(*CloudReconError); ok {
		for _, retryableType := range policy.RetryableErrors {
			if cloudErr.Type == retryableType {
				return true
			}
		}
	}

	return false
}

// addJitter adds random jitter to a duration
func addJitter(duration time.Duration) time.Duration {
	// Add ±25% jitter
	jitter := time.Duration(float64(duration) * 0.25)
	return duration + time.Duration(float64(jitter)*(2*0.5-1)) // -1 to 1
}
