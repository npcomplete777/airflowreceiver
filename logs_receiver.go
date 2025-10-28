// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package airflowreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	
	scraper_internal "github.com/npcomplete777/airflowreceiver/internal/scraper"
)

type logsReceiver struct {
	settings receiver.Settings
	consumer consumer.Logs
	scraper  *scraper_internal.LogScraper
	cancel   context.CancelFunc
	interval time.Duration
}

func newLogsReceiver(
	settings receiver.Settings,
	cfg *LogConfig,
	consumer consumer.Logs,
) (*logsReceiver, error) {
	logCfg := &scraper_internal.LogScraperConfig{
		Host:               cfg.Host,
		Port:               cfg.Port,
		Database:           cfg.Database,
		Username:           cfg.Username,
		Password:           string(cfg.Password),
		SSLMode:            cfg.SSLMode,
		CollectionInterval: cfg.CollectionInterval,
	}
	
	return &logsReceiver{
		settings: settings,
		consumer: consumer,
		scraper:  scraper_internal.NewLogScraper(logCfg, settings),
		interval: cfg.CollectionInterval,
	}, nil
}

func (r *logsReceiver) Start(ctx context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting Airflow logs receiver",
		zap.Duration("interval", r.interval))
	
	// Start the log scraper's database connection
	if err := r.scraper.Start(ctx, host); err != nil {
		return err
	}
	
	// Create cancellable context for polling goroutine
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	
	// Start polling goroutine
	go r.poll(ctx)
	
	return nil
}

func (r *logsReceiver) poll(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			r.settings.Logger.Info("Stopping logs polling")
			return
		case <-ticker.C:
			r.scrapeLogs(ctx)
		}
	}
}

func (r *logsReceiver) scrapeLogs(ctx context.Context) {
	logs, err := r.scraper.Scrape(ctx)
	if err != nil {
		r.settings.Logger.Error("Failed to scrape logs", zap.Error(err))
		return
	}
	
	if logs.LogRecordCount() == 0 {
		return
	}
	
	if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
		r.settings.Logger.Error("Failed to consume logs", zap.Error(err))
	}
}

func (r *logsReceiver) Shutdown(ctx context.Context) error {
	r.settings.Logger.Info("Shutting down Airflow logs receiver")
	
	if r.cancel != nil {
		r.cancel()
	}
	
	return r.scraper.Shutdown(ctx)
}
