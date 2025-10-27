// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package airflowreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	
	scraper_internal "github.com/npcomplete777/airflowreceiver/internal/scraper"
)

const typeStr = "airflow"

var typeVal = component.MustNewType(typeStr)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeVal,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
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
	
	// REST API scraper (using the pattern from Databricks receiver)
	if rCfg.CollectionModes.RESTAPI {
		restCfg := &scraper_internal.RESTAPIConfig{
			Endpoint:           rCfg.RESTAPIConfig.Endpoint,
			Username:           rCfg.RESTAPIConfig.Username,
			Password:           string(rCfg.RESTAPIConfig.Password),
			CollectionInterval: rCfg.RESTAPIConfig.CollectionInterval,
			IncludePastRuns:    rCfg.RESTAPIConfig.IncludePastRuns,
			PastRunsLookback:   rCfg.RESTAPIConfig.PastRunsLookback,
		}
		
		scraperInstance := scraper_internal.NewRESTAPIScraper(restCfg, settings)
		
		// Create the scraper wrapper that calls our Scrape method
		sc, err := scraper.NewMetrics(scraperInstance.Scrape)
		if err != nil {
			return nil, err
		}
		
		// Use scraperhelper to create controller with periodic scraping
		return scraperhelper.NewMetricsController(
			&rCfg.ControllerConfig,
			settings,
			consumer,
			scraperhelper.AddScraper(typeVal, sc),
		)
	}
	
	return nil, nil
}
