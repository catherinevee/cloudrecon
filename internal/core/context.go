package core

import (
	"context"
	"fmt"
	"time"
)

// OperationContext wraps context with additional metadata and timeout management
type OperationContext struct {
	ctx       context.Context
	cancel    context.CancelFunc
	timeout   time.Duration
	operation string
	startTime time.Time
	metadata  map[string]interface{}
}

// NewOperationContext creates a new operation context with timeout
func NewOperationContext(parent context.Context, operation string, timeout time.Duration) *OperationContext {
	ctx, cancel := context.WithTimeout(parent, timeout)
	return &OperationContext{
		ctx:       ctx,
		cancel:    cancel,
		timeout:   timeout,
		operation: operation,
		startTime: time.Now(),
		metadata:  make(map[string]interface{}),
	}
}

// NewOperationContextWithCancel creates a new operation context with cancellation
func NewOperationContextWithCancel(parent context.Context, operation string) *OperationContext {
	ctx, cancel := context.WithCancel(parent)
	return &OperationContext{
		ctx:       ctx,
		cancel:    cancel,
		operation: operation,
		startTime: time.Now(),
		metadata:  make(map[string]interface{}),
	}
}

// Context returns the underlying context
func (oc *OperationContext) Context() context.Context {
	return oc.ctx
}

// Cancel cancels the operation
func (oc *OperationContext) Cancel() {
	oc.cancel()
}

// Done returns a channel that's closed when the operation is done
func (oc *OperationContext) Done() <-chan struct{} {
	return oc.ctx.Done()
}

// Err returns the error if the context is cancelled or times out
func (oc *OperationContext) Err() error {
	return oc.ctx.Err()
}

// Deadline returns the deadline for the operation
func (oc *OperationContext) Deadline() (deadline time.Time, ok bool) {
	return oc.ctx.Deadline()
}

// Value returns the value for the given key
func (oc *OperationContext) Value(key interface{}) interface{} {
	return oc.ctx.Value(key)
}

// Operation returns the operation name
func (oc *OperationContext) Operation() string {
	return oc.operation
}

// Elapsed returns the elapsed time since the operation started
func (oc *OperationContext) Elapsed() time.Duration {
	return time.Since(oc.startTime)
}

// Remaining returns the remaining time before timeout
func (oc *OperationContext) Remaining() time.Duration {
	if oc.timeout == 0 {
		return 0
	}
	elapsed := oc.Elapsed()
	if elapsed >= oc.timeout {
		return 0
	}
	return oc.timeout - elapsed
}

// SetMetadata sets metadata for the operation
func (oc *OperationContext) SetMetadata(key string, value interface{}) {
	oc.metadata[key] = value
}

// GetMetadata gets metadata for the operation
func (oc *OperationContext) GetMetadata(key string) (interface{}, bool) {
	value, ok := oc.metadata[key]
	return value, ok
}

// GetMetadataString gets metadata as a string
func (oc *OperationContext) GetMetadataString(key string) (string, bool) {
	value, ok := oc.metadata[key]
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// GetMetadataInt gets metadata as an int
func (oc *OperationContext) GetMetadataInt(key string) (int, bool) {
	value, ok := oc.metadata[key]
	if !ok {
		return 0, false
	}
	i, ok := value.(int)
	return i, ok
}

// GetMetadataDuration gets metadata as a duration
func (oc *OperationContext) GetMetadataDuration(key string) (time.Duration, bool) {
	value, ok := oc.metadata[key]
	if !ok {
		return 0, false
	}
	d, ok := value.(time.Duration)
	return d, ok
}

// IsTimeout returns true if the operation timed out
func (oc *OperationContext) IsTimeout() bool {
	return oc.Err() == context.DeadlineExceeded
}

// IsCancelled returns true if the operation was cancelled
func (oc *OperationContext) IsCancelled() bool {
	return oc.Err() == context.Canceled
}

// String returns a string representation of the operation context
func (oc *OperationContext) String() string {
	return fmt.Sprintf("OperationContext{operation=%s, elapsed=%v, remaining=%v, err=%v}",
		oc.operation, oc.Elapsed(), oc.Remaining(), oc.Err())
}

// ContextManager manages operation contexts and provides utilities
type ContextManager struct {
	parentContext context.Context
	timeouts      map[string]time.Duration
}

// NewContextManager creates a new context manager
func NewContextManager(parent context.Context) *ContextManager {
	return &ContextManager{
		parentContext: parent,
		timeouts:      make(map[string]time.Duration),
	}
}

// SetTimeout sets the timeout for a specific operation type
func (cm *ContextManager) SetTimeout(operation string, timeout time.Duration) {
	cm.timeouts[operation] = timeout
}

// GetTimeout gets the timeout for a specific operation type
func (cm *ContextManager) GetTimeout(operation string) time.Duration {
	if timeout, ok := cm.timeouts[operation]; ok {
		return timeout
	}
	// Default timeouts
	switch operation {
	case "discovery":
		return 5 * time.Minute
	case "analysis":
		return 2 * time.Minute
	case "export":
		return 1 * time.Minute
	case "query":
		return 30 * time.Second
	default:
		return 30 * time.Second
	}
}

// NewOperationContext creates a new operation context with the appropriate timeout
func (cm *ContextManager) NewOperationContext(operation string) *OperationContext {
	timeout := cm.GetTimeout(operation)
	return NewOperationContext(cm.parentContext, operation, timeout)
}

// NewOperationContextWithTimeout creates a new operation context with a custom timeout
func (cm *ContextManager) NewOperationContextWithTimeout(operation string, timeout time.Duration) *OperationContext {
	return NewOperationContext(cm.parentContext, operation, timeout)
}

// NewOperationContextWithCancel creates a new operation context with cancellation
func (cm *ContextManager) NewOperationContextWithCancel(operation string) *OperationContext {
	return NewOperationContextWithCancel(cm.parentContext, operation)
}

// WithTimeout creates a new context with timeout
func (cm *ContextManager) WithTimeout(operation string) (context.Context, context.CancelFunc) {
	timeout := cm.GetTimeout(operation)
	return context.WithTimeout(cm.parentContext, timeout)
}

// WithCancel creates a new context with cancellation
func (cm *ContextManager) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(cm.parentContext)
}

// Default timeouts for different operations
const (
	DefaultDiscoveryTimeout = 5 * time.Minute
	DefaultAnalysisTimeout  = 2 * time.Minute
	DefaultExportTimeout    = 1 * time.Minute
	DefaultQueryTimeout     = 30 * time.Second
	DefaultStorageTimeout   = 10 * time.Second
)

// Operation types
const (
	OperationDiscovery = "discovery"
	OperationAnalysis  = "analysis"
	OperationExport    = "export"
	OperationQuery     = "query"
	OperationStorage   = "storage"
	OperationAuth      = "auth"
	OperationNetwork   = "network"
)
