// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package airflowreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.uber.org/zap"
	
	scraper_internal "github.com/npcomplete777/airflowreceiver/internal/scraper"
)

const typeStr = "airflow"

var typeVal = component.MustNewType(typeStr)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeVal,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig: scraperhelper.NewDefaultControllerConfig(),
		CollectionModes: CollectionModes{
			RESTAPI: true,
		},
	}
}

func createMetricsReceiver(
	ctx context.Context,
	settings receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	rCfg := cfg.(*Config)
	
	opts := make([]scraperhelper.ControllerOption, 0, 3)
	
	// REST API scraper
	if rCfg.CollectionModes.RESTAPI {
		settings.Logger.Info("Enabling REST API scraper")
		
		restCfg := &scraper_internal.RESTAPIConfig{
			Endpoint:           rCfg.RESTAPIConfig.Endpoint,
			Username:           rCfg.RESTAPIConfig.Username,
			Password:           string(rCfg.RESTAPIConfig.Password),
			CollectionInterval: rCfg.RESTAPIConfig.CollectionInterval,
			IncludePastRuns:    rCfg.RESTAPIConfig.IncludePastRuns,
			PastRunsLookback:   rCfg.RESTAPIConfig.PastRunsLookback,
		}
		
		scraperInstance := scraper_internal.NewRESTAPIScraper(restCfg, settings)
		sc, err := scraper.NewMetrics(scraperInstance.Scrape)
		if err != nil {
			return nil, fmt.Errorf("failed to create REST API scraper: %w", err)
		}
		
		opts = append(opts, scraperhelper.AddScraper(component.MustNewType("airflow_rest"), sc))
	}
	
	// Database scraper with wrapper
	if rCfg.CollectionModes.Database {
		settings.Logger.Info("Enabling Database scraper")
		
		dbCfg := &scraper_internal.DatabaseConfig{
			Host:               rCfg.DatabaseConfig.Host,
			Port:               rCfg.DatabaseConfig.Port,
			Database:           rCfg.DatabaseConfig.Database,
			Username:           rCfg.DatabaseConfig.Username,
			Password:           string(rCfg.DatabaseConfig.Password),
			SSLMode:            rCfg.DatabaseConfig.SSLMode,
			CollectionInterval: rCfg.DatabaseConfig.CollectionInterval,
		}
		
		dbScraper := scraper_internal.NewDatabaseScraper(dbCfg, settings)
		wrapper := scraper_internal.NewDatabaseScraperWrapper(dbScraper)
		sc, err := scraper.NewMetrics(wrapper.Scrape)
		if err != nil {
			return nil, fmt.Errorf("failed to create database scraper: %w", err)
		}
		
		opts = append(opts, scraperhelper.AddScraper(component.MustNewType("airflow_db"), sc))
	}
	
	// StatsD scraper
	if rCfg.CollectionModes.StatsD {
		settings.Logger.Info("Enabling StatsD scraper")
		
		statsdCfg := &scraper_internal.StatsDConfig{
			Endpoint:            rCfg.StatsDConfig.Endpoint,
			AggregationInterval: rCfg.StatsDConfig.AggregationInterval,
		}
		
		scraperInstance := scraper_internal.NewStatsDScraper(statsdCfg, settings)
		sc, err := scraper.NewMetrics(scraperInstance.Scrape)
		if err != nil {
			return nil, fmt.Errorf("failed to create StatsD scraper: %w", err)
		}
		
		opts = append(opts, scraperhelper.AddScraper(component.MustNewType("airflow_statsd"), sc))
	}
	
	if len(opts) == 0 {
		return nil, fmt.Errorf("no data collection modes enabled")
	}
	
	settings.Logger.Info("Creating Airflow receiver", zap.Int("scraper_count", len(opts)))
	
	return scraperhelper.NewMetricsController(
		&rCfg.ControllerConfig,
		settings,
		consumer,
		opts...,
	)
}

func createLogsReceiver(
	ctx context.Context,
	settings receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	rCfg := cfg.(*Config)
	
	if !rCfg.CollectionModes.Logs {
		return nil, fmt.Errorf("logs collection mode not enabled")
	}
	
	if rCfg.LogConfig == nil {
		return nil, fmt.Errorf("logs config is required when logs mode is enabled")
	}
	
	settings.Logger.Info("Creating Airflow logs receiver")
	
	return newLogsReceiver(settings, rCfg.LogConfig, consumer)
}
