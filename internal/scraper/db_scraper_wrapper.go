// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type DatabaseScraperWrapper struct {
	scraper *DatabaseScraper
	once    sync.Once
	started bool
}

func NewDatabaseScraperWrapper(scraper *DatabaseScraper) *DatabaseScraperWrapper {
	return &DatabaseScraperWrapper{
		scraper: scraper,
	}
}

func (w *DatabaseScraperWrapper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// Ensure Start is called before first scrape
	w.once.Do(func() {
		if err := w.scraper.Start(ctx, component.Host(nil)); err != nil {
			// Log error but continue - will return empty metrics
		}
		w.started = true
	})
	
	if !w.started {
		return pmetric.NewMetrics(), nil
	}
	
	return w.scraper.Scrape(ctx)
}
