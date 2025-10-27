// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package airflowreceiver

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

var (
	ErrNoEndpoint = errors.New("endpoint must be specified")
	ErrNoMode     = errors.New("at least one collection mode must be enabled")
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	
	CollectionModes CollectionModes `mapstructure:"collection_modes"`
	RESTAPIConfig *RESTAPIConfig `mapstructure:"rest_api"`
	DatabaseConfig *DatabaseConfig `mapstructure:"database"`
	StatsDConfig *StatsDConfig `mapstructure:"statsd"`
}

type CollectionModes struct {
	RESTAPI bool `mapstructure:"rest_api"`
	Database bool `mapstructure:"database"`
	StatsD  bool `mapstructure:"statsd"`
}

type RESTAPIConfig struct {
	confighttp.ClientConfig `mapstructure:",squash"`
	
	Username string `mapstructure:"username"`
	Password configopaque.String `mapstructure:"password"`
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	IncludePastRuns bool `mapstructure:"include_past_runs"`
	PastRunsLookback time.Duration `mapstructure:"past_runs_lookback"`
}

type DatabaseConfig struct {
	Host               string                     `mapstructure:"host"`
	Port               int                        `mapstructure:"port"`
	Database           string                     `mapstructure:"database"`
	Username           string                     `mapstructure:"username"`
	Password           configopaque.String        `mapstructure:"password"`
	SSLMode            string                     `mapstructure:"ssl_mode"`
	CollectionInterval time.Duration              `mapstructure:"collection_interval"`
	QueryTimeout       time.Duration              `mapstructure:"query_timeout"`
}

type StatsDConfig struct {
	confignet.AddrConfig `mapstructure:",squash"`
	
	AggregationInterval time.Duration `mapstructure:"aggregation_interval"`
	EnableMetricType bool `mapstructure:"enable_metric_type"`
	TimerHistogramMapping []TimerHistogramMapping `mapstructure:"timer_histogram_mapping"`
}

type TimerHistogramMapping struct {
	StatsdType   string  `mapstructure:"statsd_type"`
	ObserverType string  `mapstructure:"observer_type"`
	Histogram    Histogram `mapstructure:"histogram"`
}

type Histogram struct {
	MaxSize int32 `mapstructure:"max_size"`
}

func (cfg *Config) Validate() error {
	if !cfg.CollectionModes.RESTAPI && !cfg.CollectionModes.Database && !cfg.CollectionModes.StatsD {
		return ErrNoMode
	}
	
	if cfg.CollectionModes.RESTAPI {
		if cfg.RESTAPIConfig == nil {
			return errors.New("rest_api config required when rest_api mode enabled")
		}
		if cfg.RESTAPIConfig.Endpoint == "" {
			return fmt.Errorf("rest_api: %w", ErrNoEndpoint)
		}
		if cfg.RESTAPIConfig.CollectionInterval <= 0 {
			cfg.RESTAPIConfig.CollectionInterval = 30 * time.Second
		}
	}
	
	if cfg.CollectionModes.Database {
		if cfg.DatabaseConfig == nil {
			return errors.New("database config required when database mode enabled")
		}
		if cfg.DatabaseConfig.Host == "" {
			return errors.New("database host must be specified")
		}
		if cfg.DatabaseConfig.CollectionInterval <= 0 {
		}
		if cfg.DatabaseConfig.Port == 0 {
			cfg.DatabaseConfig.Port = 5432
		}
		if cfg.DatabaseConfig.SSLMode == "" {
			cfg.DatabaseConfig.SSLMode = "disable"
		}
		if cfg.DatabaseConfig.QueryTimeout <= 0 {
			cfg.DatabaseConfig.QueryTimeout = 15 * time.Second
			cfg.DatabaseConfig.CollectionInterval = 30 * time.Second
		}
	}
	
	if cfg.CollectionModes.StatsD {
		if cfg.StatsDConfig == nil {
			return errors.New("statsd config required when statsd mode enabled")
		}
		if cfg.StatsDConfig.AggregationInterval <= 0 {
			cfg.StatsDConfig.AggregationInterval = 60 * time.Second
		}
	}
	
	return nil
}
