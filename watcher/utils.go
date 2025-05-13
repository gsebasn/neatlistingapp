package main

import (
	"context"
	"fmt"
	"time"
)

// retryOperation attempts to execute an operation with exponential backoff
func retryOperation(ctx context.Context, operation func() error, maxRetries int, backoff time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := operation()
			if err == nil {
				return nil
			}
			if i < maxRetries-1 {
				time.Sleep(backoff)
			}
		}
	}
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}
