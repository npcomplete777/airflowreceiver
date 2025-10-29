// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"math"
	"time"

	"go.uber.org/zap"
)

type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     10 * time.Second,
		Multiplier:      2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, logger *zap.Logger, operation string, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff
			backoff := time.Duration(float64(cfg.InitialInterval) * math.Pow(cfg.Multiplier, float64(attempt-1)))
			if backoff > cfg.MaxInterval {
				backoff = cfg.MaxInterval
			}
			
			logger.Debug("Retrying operation after backoff",
				zap.String("operation", operation),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", cfg.MaxAttempts),
				zap.Duration("backoff", backoff))
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
		
		lastErr = fn()
		if lastErr == nil {
			if attempt > 0 {
				logger.Info("Operation succeeded after retry",
					zap.String("operation", operation),
					zap.Int("attempts", attempt+1))
			}
			return nil
		}
		
		logger.Warn("Operation failed",
			zap.String("operation", operation),
			zap.Int("attempt", attempt+1),
			zap.Error(lastErr))
	}
	
	logger.Error("Operation failed after all retries",
		zap.String("operation", operation),
		zap.Int("attempts", cfg.MaxAttempts),
		zap.Error(lastErr))
	
	return lastErr
}
