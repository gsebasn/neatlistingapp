package main

import (
	"context"
	"fmt"
	"math"
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
	metrics      MetricsCollector
}

// AdaptiveRateLimiterConfig extends RateLimiterConfig with adaptive parameters
type AdaptiveRateLimiterConfig struct {
	RateLimiterConfig
	AdjustmentInterval    time.Duration // How often to adjust rates
	MinRatePerSecond      float64       // Minimum allowed rate
	MaxRatePerSecond      float64       // Maximum allowed rate
	TargetSuccessRate     float64       // Target success rate (0.0 to 1.0)
	TargetLatencyMs       float64       // Target latency in milliseconds
	BurstAdjustmentFactor float64       // How aggressively to adjust burst limits
}

// AdaptiveRateLimiter extends RateLimiter with dynamic rate adjustment
type AdaptiveRateLimiter struct {
	*RateLimiter
	config             AdaptiveRateLimiterConfig
	metrics            MetricsCollector
	lastAdjustment     time.Time
	currentSuccessRate float64
	currentLatency     float64
	adjustmentTicker   *time.Ticker
	stopChan           chan struct{}
	mu                 sync.RWMutex
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
func NewRateLimiter(config RateLimiterConfig, metrics MetricsCollector) *RateLimiter {
	return &RateLimiter{
		globalBucket: NewTokenBucket(config.GlobalRatePerSecond, config.BurstMultiplier, "global"),
		insertBucket: NewTokenBucket(config.InsertRatePerSecond, config.OperationBurstMultiplier, OperationInsert),
		updateBucket: NewTokenBucket(config.UpdateRatePerSecond, config.OperationBurstMultiplier, OperationUpdate),
		deleteBucket: NewTokenBucket(config.DeleteRatePerSecond, config.OperationBurstMultiplier, OperationDelete),
		metrics:      metrics,
	}
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(config AdaptiveRateLimiterConfig, metrics MetricsCollector) *AdaptiveRateLimiter {
	baseLimiter := NewRateLimiter(config.RateLimiterConfig, metrics)

	adaptive := &AdaptiveRateLimiter{
		RateLimiter:      baseLimiter,
		config:           config,
		metrics:          metrics,
		lastAdjustment:   time.Now(),
		adjustmentTicker: time.NewTicker(config.AdjustmentInterval),
		stopChan:         make(chan struct{}),
	}

	// Start the adjustment loop
	go adaptive.adjustmentLoop()

	return adaptive
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

// adjustmentLoop periodically adjusts rate limits based on metrics
func (a *AdaptiveRateLimiter) adjustmentLoop() {
	for {
		select {
		case <-a.adjustmentTicker.C:
			a.adjustRates()
		case <-a.stopChan:
			a.adjustmentTicker.Stop()
			return
		}
	}
}

// adjustRates analyzes metrics and adjusts rate limits
func (a *AdaptiveRateLimiter) adjustRates() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Calculate current success rate and latency
	successRate := a.calculateSuccessRate()
	latency := a.calculateAverageLatency()

	// Log current metrics
	log.Info().
		Float64("success_rate", successRate).
		Float64("latency_ms", latency).
		Float64("target_success_rate", a.config.TargetSuccessRate).
		Float64("target_latency_ms", a.config.TargetLatencyMs).
		Msg("Adjusting rate limits based on metrics")

	// Adjust rates based on success rate and latency
	for _, opType := range []OperationType{OperationInsert, OperationUpdate, OperationDelete} {
		currentRate := a.getCurrentRate(opType)
		newRate := a.calculateNewRate(currentRate, successRate, latency, opType)

		// Apply the new rate
		a.updateRate(opType, newRate)

		log.Info().
			Str("operation", string(opType)).
			Float64("old_rate", currentRate).
			Float64("new_rate", newRate).
			Msg("Updated operation rate")
	}

	// Adjust burst multipliers if needed
	a.adjustBurstMultipliers(successRate, latency)

	a.lastAdjustment = time.Now()
}

// calculateSuccessRate calculates the current success rate from metrics
func (a *AdaptiveRateLimiter) calculateSuccessRate() float64 {
	// Query Prometheus for success rate over the last adjustment interval
	// This is a simplified version - in practice, you'd use Prometheus client
	successCount := float64(0)
	totalCount := float64(0)

	// For each operation type
	for _, opType := range []string{"insert", "update", "delete"} {
		// Get success and total counts from metrics
		success := a.metrics.getOperationCount(opType, "success")
		total := a.metrics.getOperationCount(opType, "success") +
			a.metrics.getOperationCount(opType, "error")

		successCount += success
		totalCount += total
	}

	if totalCount == 0 {
		return 1.0 // Default to 100% if no operations
	}

	return successCount / totalCount
}

// calculateAverageLatency calculates the current average latency from metrics
func (a *AdaptiveRateLimiter) calculateAverageLatency() float64 {
	// Query Prometheus for average latency over the last adjustment interval
	// This is a simplified version - in practice, you'd use Prometheus client
	totalLatency := float64(0)
	count := float64(0)

	// For each operation type
	for _, opType := range []string{"insert", "update", "delete"} {
		latency := a.metrics.getAverageProcessingDuration(opType)
		count++
		totalLatency += latency
	}

	if count == 0 {
		return 0
	}

	return (totalLatency / count) * 1000 // Convert to milliseconds
}

// calculateNewRate determines the new rate based on current metrics
func (a *AdaptiveRateLimiter) calculateNewRate(currentRate, successRate, latency float64, opType OperationType) float64 {
	// Start with current rate
	newRate := currentRate

	// Adjust based on success rate
	if successRate < a.config.TargetSuccessRate {
		// Reduce rate if success rate is too low
		newRate *= 0.9
	} else if successRate > a.config.TargetSuccessRate*1.1 {
		// Increase rate if success rate is well above target
		newRate *= 1.1
	}

	// Adjust based on latency
	if latency > a.config.TargetLatencyMs {
		// Reduce rate if latency is too high
		newRate *= 0.95
	} else if latency < a.config.TargetLatencyMs*0.8 {
		// Increase rate if latency is well below target
		newRate *= 1.05
	}

	// Ensure rate stays within bounds
	newRate = math.Max(a.config.MinRatePerSecond,
		math.Min(a.config.MaxRatePerSecond, newRate))

	return newRate
}

// adjustBurstMultipliers adjusts burst multipliers based on metrics
func (a *AdaptiveRateLimiter) adjustBurstMultipliers(successRate, latency float64) {
	// Adjust global burst multiplier
	if successRate < a.config.TargetSuccessRate {
		a.config.BurstMultiplier *= 0.9
	} else if successRate > a.config.TargetSuccessRate*1.1 {
		a.config.BurstMultiplier *= 1.1
	}

	// Adjust operation-specific burst multiplier
	if latency > a.config.TargetLatencyMs {
		a.config.OperationBurstMultiplier *= 0.9
	} else if latency < a.config.TargetLatencyMs*0.8 {
		a.config.OperationBurstMultiplier *= 1.1
	}

	// Update the rate limiter with new burst multipliers
	a.updateBurstMultipliers()
}

// updateRate updates the rate for a specific operation type
func (a *AdaptiveRateLimiter) updateRate(opType OperationType, newRate float64) {
	switch opType {
	case OperationInsert:
		a.insertBucket.rate = newRate
		a.insertBucket.capacity = newRate * a.config.OperationBurstMultiplier
	case OperationUpdate:
		a.updateBucket.rate = newRate
		a.updateBucket.capacity = newRate * a.config.OperationBurstMultiplier
	case OperationDelete:
		a.deleteBucket.rate = newRate
		a.deleteBucket.capacity = newRate * a.config.OperationBurstMultiplier
	}
}

// updateBurstMultipliers updates burst multipliers for all buckets
func (a *AdaptiveRateLimiter) updateBurstMultipliers() {
	a.globalBucket.capacity = a.globalBucket.rate * a.config.BurstMultiplier
	a.insertBucket.capacity = a.insertBucket.rate * a.config.OperationBurstMultiplier
	a.updateBucket.capacity = a.updateBucket.rate * a.config.OperationBurstMultiplier
	a.deleteBucket.capacity = a.deleteBucket.rate * a.config.OperationBurstMultiplier
}

// Stop stops the adaptive rate limiter
func (a *AdaptiveRateLimiter) Stop() {
	close(a.stopChan)
}

// getCurrentRate returns the current rate for an operation type
func (a *AdaptiveRateLimiter) getCurrentRate(opType OperationType) float64 {
	switch opType {
	case OperationInsert:
		return a.insertBucket.rate
	case OperationUpdate:
		return a.updateBucket.rate
	case OperationDelete:
		return a.deleteBucket.rate
	default:
		return a.globalBucket.rate
	}
}
