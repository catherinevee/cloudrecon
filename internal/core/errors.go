package core

import (
	"fmt"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeAuth       ErrorType = "auth"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeStorage    ErrorType = "storage"
	ErrorTypeProvider   ErrorType = "provider"
)

// CloudReconError represents a standardized error with context
type CloudReconError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Cause   error                  `json:"cause,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
	Time    time.Time              `json:"time"`
}

// Error implements the error interface
func (e *CloudReconError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *CloudReconError) Unwrap() error {
	return e.Cause
}

// NewCloudReconError creates a new CloudReconError
func NewCloudReconError(errorType ErrorType, message string, cause error) *CloudReconError {
	return &CloudReconError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Time:    time.Now(),
	}
}

// WithContext adds context to the error
func (e *CloudReconError) WithContext(key string, value interface{}) *CloudReconError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// IsRetryable returns true if the error is retryable
func (e *CloudReconError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeNetwork, ErrorTypeRateLimit, ErrorTypeTimeout:
		return true
	case ErrorTypeAuth, ErrorTypeConfig, ErrorTypeValidation, ErrorTypeNotFound:
		return false
	case ErrorTypeInternal, ErrorTypeStorage, ErrorTypeProvider:
		// Check if the underlying error is retryable
		if e.Cause != nil {
			return IsRetryableError(e.Cause)
		}
		return false
	default:
		return false
	}
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a CloudReconError
	if cloudErr, ok := err.(*CloudReconError); ok {
		return cloudErr.IsRetryable()
	}

	// Check for common retryable error patterns
	errorStr := err.Error()
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
	}

	for _, pattern := range retryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// Error helpers for common error types
func NewConfigError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeConfig, message, cause)
}

func NewAuthError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeAuth, message, cause)
}

func NewNetworkError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeNetwork, message, cause)
}

func NewRateLimitError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeRateLimit, message, cause)
}

func NewNotFoundError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeNotFound, message, cause)
}

func NewInternalError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeInternal, message, cause)
}

func NewTimeoutError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeTimeout, message, cause)
}

func NewValidationError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeValidation, message, cause)
}

func NewStorageError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeStorage, message, cause)
}

func NewProviderError(message string, cause error) *CloudReconError {
	return NewCloudReconError(ErrorTypeProvider, message, cause)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr))))
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