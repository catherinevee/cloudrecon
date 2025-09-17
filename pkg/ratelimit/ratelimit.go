package ratelimit

import (
	"context"
	"sync"
	"time"
)

// RateLimiter represents a rate limiter
type RateLimiter struct {
	limit      int
	interval   time.Duration
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:      limit,
		interval:   interval,
		tokens:     limit,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.interval)

	if tokensToAdd > 0 {
		rl.tokens = min(rl.limit, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		// Calculate time until next token is available
		rl.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill)
		timeUntilNextToken := rl.interval - elapsed
		rl.mu.Unlock()

		if timeUntilNextToken <= 0 {
			continue
		}

		// Wait for next token or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeUntilNextToken):
			// Continue to next iteration
		}
	}
}

// Tokens returns the current number of available tokens
func (rl *RateLimiter) Tokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.interval)

	if tokensToAdd > 0 {
		rl.tokens = min(rl.limit, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	return rl.tokens
}

// Reset resets the rate limiter to its initial state
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.tokens = rl.limit
	rl.lastRefill = time.Now()
}

// MultiRateLimiter represents multiple rate limiters for different operations
type MultiRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// NewMultiRateLimiter creates a new multi-rate limiter
func NewMultiRateLimiter() *MultiRateLimiter {
	return &MultiRateLimiter{
		limiters: make(map[string]*RateLimiter),
	}
}

// AddLimiter adds a rate limiter for a specific operation
func (mrl *MultiRateLimiter) AddLimiter(operation string, limit int, interval time.Duration) {
	mrl.mu.Lock()
	defer mrl.mu.Unlock()

	mrl.limiters[operation] = NewRateLimiter(limit, interval)
}

// Allow checks if a request is allowed for the specified operation
func (mrl *MultiRateLimiter) Allow(operation string) bool {
	mrl.mu.RLock()
	limiter, exists := mrl.limiters[operation]
	mrl.mu.RUnlock()

	if !exists {
		return true // No rate limit for this operation
	}

	return limiter.Allow()
}

// Wait blocks until a token is available for the specified operation
func (mrl *MultiRateLimiter) Wait(ctx context.Context, operation string) error {
	mrl.mu.RLock()
	limiter, exists := mrl.limiters[operation]
	mrl.mu.RUnlock()

	if !exists {
		return nil // No rate limit for this operation
	}

	return limiter.Wait(ctx)
}

// Tokens returns the current number of available tokens for the specified operation
func (mrl *MultiRateLimiter) Tokens(operation string) int {
	mrl.mu.RLock()
	limiter, exists := mrl.limiters[operation]
	mrl.mu.RUnlock()

	if !exists {
		return -1 // No rate limit for this operation
	}

	return limiter.Tokens()
}

// Reset resets all rate limiters
func (mrl *MultiRateLimiter) Reset() {
	mrl.mu.Lock()
	defer mrl.mu.Unlock()

	for _, limiter := range mrl.limiters {
		limiter.Reset()
	}
}

// AdaptiveRateLimiter represents a rate limiter that adapts based on success/failure rates
type AdaptiveRateLimiter struct {
	baseLimit    int
	currentLimit int
	interval     time.Duration
	tokens       int
	lastRefill   time.Time
	successCount int
	failureCount int
	mu           sync.Mutex
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(baseLimit int, interval time.Duration) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimit:    baseLimit,
		currentLimit: baseLimit,
		interval:     interval,
		tokens:       baseLimit,
		lastRefill:   time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (arl *AdaptiveRateLimiter) Allow() bool {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(arl.lastRefill)
	tokensToAdd := int(elapsed / arl.interval)

	if tokensToAdd > 0 {
		arl.tokens = min(arl.currentLimit, arl.tokens+tokensToAdd)
		arl.lastRefill = now
	}

	// Check if we have tokens available
	if arl.tokens > 0 {
		arl.tokens--
		return true
	}

	return false
}

// RecordSuccess records a successful operation
func (arl *AdaptiveRateLimiter) RecordSuccess() {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	arl.successCount++
	arl.adjustLimit()
}

// RecordFailure records a failed operation
func (arl *AdaptiveRateLimiter) RecordFailure() {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	arl.failureCount++
	arl.adjustLimit()
}

// adjustLimit adjusts the rate limit based on success/failure rates
func (arl *AdaptiveRateLimiter) adjustLimit() {
	total := arl.successCount + arl.failureCount
	if total < 10 {
		return // Not enough data to adjust
	}

	successRate := float64(arl.successCount) / float64(total)

	// If success rate is high, increase limit
	if successRate > 0.9 && arl.currentLimit < arl.baseLimit*2 {
		arl.currentLimit = min(arl.baseLimit*2, arl.currentLimit+1)
	}

	// If success rate is low, decrease limit
	if successRate < 0.7 && arl.currentLimit > 1 {
		arl.currentLimit = max(1, arl.currentLimit-1)
	}

	// Reset counters periodically
	if total > 100 {
		arl.successCount = 0
		arl.failureCount = 0
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
