// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type DatabaseScraperWrapper struct {
	scraper *DatabaseScraper
	health  *ScraperHealth
	once    sync.Once
	started bool
}

func NewDatabaseScraperWrapper(scraper *DatabaseScraper) *DatabaseScraperWrapper {
	return &DatabaseScraperWrapper{
		scraper: scraper,
		health:  NewScraperHealth("database", scraper.settings.Logger),
	}
}

func (w *DatabaseScraperWrapper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// Ensure Start is called before first scrape
	w.once.Do(func() {
		if err := w.scraper.Start(ctx, component.Host(nil)); err != nil {
			w.health.RecordScrape(0, err)
			w.scraper.settings.Logger.Error("Failed to start database scraper", zap.Error(err))
		} else {
			w.started = true
		}
	})
	
	if !w.started {
		return pmetric.NewMetrics(), nil
	}
	
	// Use health tracking wrapper
	metrics, err := w.health.WithScrapeTracking(ctx, w.scraper.Scrape)
	
	// Add health metrics to output
	w.health.EmitMetrics(w.scraper.mb, time.Now())
	
	return metrics, err
}
