// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"net"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type StatsDScraper struct {
	cfg      *StatsDConfig
	settings receiver.Settings
	conn     *net.UDPConn
	mb       *MetricsBuilder
}

type StatsDConfig struct {
	Endpoint            string
	AggregationInterval time.Duration
}

func NewStatsDScraper(cfg *StatsDConfig, settings receiver.Settings) *StatsDScraper {
	return &StatsDScraper{
		cfg:      cfg,
		settings: settings,
		mb:       NewMetricsBuilder(),
	}
}

func (s *StatsDScraper) Start(ctx context.Context, host component.Host) error {
	s.settings.Logger.Info("Starting StatsD scraper", zap.String("endpoint", s.cfg.Endpoint))
	// TODO: Start UDP listener
	return nil
}

func (s *StatsDScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// TODO: Aggregate and emit StatsD metrics
	return s.mb.Emit(), nil
}

func (s *StatsDScraper) Shutdown(ctx context.Context) error {
	s.settings.Logger.Info("Shutting down StatsD scraper")
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
