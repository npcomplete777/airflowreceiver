// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// ScraperHealth tracks the health and performance of individual scrapers
type ScraperHealth struct {
	mu                sync.RWMutex
	scraperType       string
	logger            *zap.Logger
	
	// Success/failure counts
	totalScrapes      int64
	successfulScrapes int64
	failedScrapes     int64
	
	// Timing metrics
	lastScrapeDuration time.Duration
	avgScrapeDuration  time.Duration
	maxScrapeDuration  time.Duration
	
	// Error tracking
	lastError         error
	lastErrorTime     time.Time
	consecutiveErrors int64
	
	// Status
	lastSuccessTime   time.Time
	healthy           bool
}

func NewScraperHealth(scraperType string, logger *zap.Logger) *ScraperHealth {
	return &ScraperHealth{
		scraperType: scraperType,
		logger:      logger,
		healthy:     true,
	}
}

// RecordScrape records the result of a scrape operation
func (h *ScraperHealth) RecordScrape(duration time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.totalScrapes++
	h.lastScrapeDuration = duration
	
	if err != nil {
		h.failedScrapes++
		h.consecutiveErrors++
		h.lastError = err
		h.lastErrorTime = time.Now()
		
		// Mark unhealthy after 3 consecutive failures
		if h.consecutiveErrors >= 3 {
			h.healthy = false
			h.logger.Error("Scraper marked unhealthy",
				zap.String("scraper_type", h.scraperType),
				zap.Int64("consecutive_errors", h.consecutiveErrors),
				zap.Error(err))
		}
	} else {
		h.successfulScrapes++
		h.consecutiveErrors = 0
		h.lastSuccessTime = time.Now()
		h.healthy = true
		
		// Update average duration
		if h.avgScrapeDuration == 0 {
			h.avgScrapeDuration = duration
		} else {
			h.avgScrapeDuration = (h.avgScrapeDuration*9 + duration) / 10 // Moving average
		}
		
		// Track max duration
		if duration > h.maxScrapeDuration {
			h.maxScrapeDuration = duration
		}
	}
}

// GetMetrics returns current health metrics
func (h *ScraperHealth) GetMetrics() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	successRate := float64(0)
	if h.totalScrapes > 0 {
		successRate = float64(h.successfulScrapes) / float64(h.totalScrapes)
	}
	
	return map[string]interface{}{
		"scraper_type":        h.scraperType,
		"total_scrapes":       h.totalScrapes,
		"successful_scrapes":  h.successfulScrapes,
		"failed_scrapes":      h.failedScrapes,
		"success_rate":        successRate,
		"consecutive_errors":  h.consecutiveErrors,
		"last_duration_ms":    h.lastScrapeDuration.Milliseconds(),
		"avg_duration_ms":     h.avgScrapeDuration.Milliseconds(),
		"max_duration_ms":     h.maxScrapeDuration.Milliseconds(),
		"healthy":             h.healthy,
		"last_success":        h.lastSuccessTime,
	}
}

// EmitMetrics adds health metrics to the metrics builder
func (h *ScraperHealth) EmitMetrics(mb *MetricsBuilder, ts time.Time) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Success/failure counts
	mb.RecordScraperTotalScrapes(h.totalScrapes, h.scraperType, ts)
	mb.RecordScraperSuccessfulScrapes(h.successfulScrapes, h.scraperType, ts)
	mb.RecordScraperFailedScrapes(h.failedScrapes, h.scraperType, ts)
	
	// Health status (1=healthy, 0=unhealthy)
	healthValue := int64(0)
	if h.healthy {
		healthValue = 1
	}
	mb.RecordScraperHealthStatus(healthValue, h.scraperType, ts)
	
	// Timing metrics
	if h.lastScrapeDuration > 0 {
		mb.RecordScraperLastDuration(h.lastScrapeDuration.Seconds(), h.scraperType, ts)
	}
	if h.avgScrapeDuration > 0 {
		mb.RecordScraperAvgDuration(h.avgScrapeDuration.Seconds(), h.scraperType, ts)
	}
	
	// Consecutive errors
	mb.RecordScraperConsecutiveErrors(h.consecutiveErrors, h.scraperType, ts)
}

// IsHealthy returns current health status
func (h *ScraperHealth) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy
}

// GetLastError returns the last error and when it occurred
func (h *ScraperHealth) GetLastError() (error, time.Time) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastError, h.lastErrorTime
}

// Reset clears all metrics (useful for testing)
func (h *ScraperHealth) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.totalScrapes = 0
	h.successfulScrapes = 0
	h.failedScrapes = 0
	h.consecutiveErrors = 0
	h.lastScrapeDuration = 0
	h.avgScrapeDuration = 0
	h.maxScrapeDuration = 0
	h.lastError = nil
	h.lastErrorTime = time.Time{}
	h.healthy = true
}

// WithScrapeTracking wraps a scrape function with automatic health tracking
func (h *ScraperHealth) WithScrapeTracking(ctx context.Context, fn func(context.Context) (pmetric.Metrics, error)) (pmetric.Metrics, error) {
	start := time.Now()
	metrics, err := fn(ctx)
	duration := time.Since(start)
	
	h.RecordScrape(duration, err)
	
	if err != nil {
		h.logger.Warn("Scrape failed",
			zap.String("scraper_type", h.scraperType),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		h.logger.Debug("Scrape succeeded",
			zap.String("scraper_type", h.scraperType),
			zap.Duration("duration", duration))
	}
	
	return metrics, err
}
