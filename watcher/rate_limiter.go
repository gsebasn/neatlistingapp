package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// OperationType represents the type of operation being rate limited
type OperationType string

const (
	OperationInsert OperationType = "insert"
	OperationUpdate OperationType = "update"
	OperationDelete OperationType = "delete"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	GlobalRatePerSecond      float64
	InsertRatePerSecond      float64
	UpdateRatePerSecond      float64
	DeleteRatePerSecond      float64
	BurstMultiplier          float64 // How many tokens can be accumulated
	OperationBurstMultiplier float64
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	rate          float64
	capacity      float64
	tokens        float64
	lastRefill    time.Time
	mu            sync.Mutex
	operationType OperationType
}

// RateLimiter manages rate limiting for different operation types
type RateLimiter struct {
	globalBucket *TokenBucket
	insertBucket *TokenBucket
	updateBucket *TokenBucket
	deleteBucket *TokenBucket
	metrics      *MetricsCollector
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(rate, burstMultiplier float64, operationType OperationType) *TokenBucket {
	return &TokenBucket{
		rate:          rate,
		capacity:      rate * burstMultiplier,
		tokens:        rate * burstMultiplier, // Start with full bucket
		lastRefill:    time.Now(),
		operationType: operationType,
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimiterConfig, metrics *MetricsCollector) *RateLimiter {
	return &RateLimiter{
		globalBucket: NewTokenBucket(config.GlobalRatePerSecond, config.BurstMultiplier, "global"),
		insertBucket: NewTokenBucket(config.InsertRatePerSecond, config.OperationBurstMultiplier, OperationInsert),
		updateBucket: NewTokenBucket(config.UpdateRatePerSecond, config.OperationBurstMultiplier, OperationUpdate),
		deleteBucket: NewTokenBucket(config.DeleteRatePerSecond, config.OperationBurstMultiplier, OperationDelete),
		metrics:      metrics,
	}
}

// refill adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.rate)
	tb.lastRefill = now
}

// Allow checks if an operation is allowed and consumes a token if it is
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}
	return false
}

// Wait blocks until an operation is allowed
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if tb.Allow() {
				return nil
			}
			// Sleep for a short duration before trying again
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// AllowOperation checks if an operation of the given type is allowed
func (rl *RateLimiter) AllowOperation(opType OperationType) bool {
	// First check global rate limit
	if !rl.globalBucket.Allow() {
		rl.metrics.RecordRateLimit("global")
		return false
	}

	// Then check operation-specific rate limit
	var allowed bool
	switch opType {
	case OperationInsert:
		allowed = rl.insertBucket.Allow()
	case OperationUpdate:
		allowed = rl.updateBucket.Allow()
	case OperationDelete:
		allowed = rl.deleteBucket.Allow()
	default:
		log.Warn().Str("operation_type", string(opType)).Msg("Unknown operation type for rate limiting")
		return false
	}

	if !allowed {
		rl.metrics.RecordRateLimit(string(opType))
	}
	return allowed
}

// WaitForOperation blocks until an operation of the given type is allowed
func (rl *RateLimiter) WaitForOperation(ctx context.Context, opType OperationType) error {
	// First wait for global rate limit
	if err := rl.globalBucket.Wait(ctx); err != nil {
		return fmt.Errorf("global rate limit wait: %w", err)
	}

	// Then wait for operation-specific rate limit
	switch opType {
	case OperationInsert:
		return rl.insertBucket.Wait(ctx)
	case OperationUpdate:
		return rl.updateBucket.Wait(ctx)
	case OperationDelete:
		return rl.deleteBucket.Wait(ctx)
	default:
		return fmt.Errorf("unknown operation type: %s", opType)
	}
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
