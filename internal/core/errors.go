package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Custom error types for better error handling

// ErrNoCredentials is returned when no valid credentials are found
var ErrNoCredentials = fmt.Errorf("no valid credentials found")

// ErrRateLimited is returned when API rate limits are exceeded
var ErrRateLimited = fmt.Errorf("API rate limit exceeded")

// ErrPartialResult is returned when discovery partially completed
var ErrPartialResult = fmt.Errorf("discovery partially completed")

// ErrNativeToolUnavailable is returned when cloud-native tools are not available
var ErrNativeToolUnavailable = fmt.Errorf("cloud-native tool not available")

// ErrProviderNotSupported is returned when a provider is not supported
var ErrProviderNotSupported = fmt.Errorf("provider not supported")

// ErrInvalidConfiguration is returned when configuration is invalid
var ErrInvalidConfiguration = fmt.Errorf("invalid configuration")

// ErrDiscoveryTimeout is returned when discovery times out
var ErrDiscoveryTimeout = fmt.Errorf("discovery timeout")

// ErrStorageError is returned when storage operations fail
var ErrStorageError = fmt.Errorf("storage error")

// ErrQueryError is returned when query execution fails
var ErrQueryError = fmt.Errorf("query error")

// PartialError represents a partial failure with multiple errors
type PartialError struct {
	Errors []error
}

func (e *PartialError) Error() string {
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("partial failure: %s", strings.Join(messages, "; "))
}

// DiscoveryError represents an error during discovery
type DiscoveryError struct {
	Provider string
	Account  string
	Region   string
	Service  string
	Err      error
}

func (e *DiscoveryError) Error() string {
	return fmt.Sprintf("discovery error in %s/%s/%s/%s: %v",
		e.Provider, e.Account, e.Region, e.Service, e.Err)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s (value: %v): %s",
		e.Field, e.Value, e.Message)
}

// ConfigurationError represents a configuration error
type ConfigurationError struct {
	Section string
	Key     string
	Value   interface{}
	Message string
}

func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error in %s.%s (value: %v): %s",
		e.Section, e.Key, e.Value, e.Message)
}

// StorageError represents a storage error
type StorageError struct {
	Operation string
	Table     string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error during %s on %s: %v",
		e.Operation, e.Table, e.Err)
}

// QueryError represents a query error
type QueryError struct {
	Query string
	Args  []interface{}
	Err   error
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("query error for query %s with args %v: %v",
		e.Query, e.Args, e.Err)
}

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider  string
	Service   string
	Operation string
	Err       error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider error in %s.%s during %s: %v",
		e.Provider, e.Service, e.Operation, e.Err)
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error: %v (retry after %v)", e.Err, e.RetryAfter)
}

// Unwrap returns the underlying error
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var retryable *RetryableError
	return err != nil && errors.As(err, &retryable)
}

// GetRetryAfter returns the retry delay for a retryable error
func GetRetryAfter(err error) time.Duration {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return retryable.RetryAfter
	}
	return 0
}
