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
	RESTAPIConfig   *RESTAPIConfig   `mapstructure:"rest_api"`
	DatabaseConfig  *DatabaseConfig  `mapstructure:"database"`
	StatsDConfig    *StatsDConfig    `mapstructure:"statsd"`
	LogConfig       *LogConfig       `mapstructure:"logs"`
}

type CollectionModes struct {
	RESTAPI  bool `mapstructure:"rest_api"`
	Database bool `mapstructure:"database"`
	StatsD   bool `mapstructure:"statsd"`
	Logs     bool `mapstructure:"logs"`
}

type RESTAPIConfig struct {
	confighttp.ClientConfig `mapstructure:",squash"`

	Username           string              `mapstructure:"username"`
	Password           configopaque.String `mapstructure:"password"`
	CollectionInterval time.Duration       `mapstructure:"collection_interval"`
	IncludePastRuns    bool                `mapstructure:"include_past_runs"`
	PastRunsLookback   time.Duration       `mapstructure:"past_runs_lookback"`
}

type DatabaseConfig struct {
	Host               string              `mapstructure:"host"`
	Port               int                 `mapstructure:"port"`
	Database           string              `mapstructure:"database"`
	Username           string              `mapstructure:"username"`
	Password           configopaque.String `mapstructure:"password"`
	SSLMode            string              `mapstructure:"ssl_mode"`
	CollectionInterval time.Duration       `mapstructure:"collection_interval"`
	QueryTimeout       time.Duration       `mapstructure:"query_timeout"`
}

type StatsDConfig struct {
	confignet.AddrConfig `mapstructure:",squash"`

	AggregationInterval   time.Duration             `mapstructure:"aggregation_interval"`
	EnableMetricType      bool                      `mapstructure:"enable_metric_type"`
	TimerHistogramMapping []TimerHistogramMapping   `mapstructure:"timer_histogram_mapping"`
}

type TimerHistogramMapping struct {
	StatsdType   string    `mapstructure:"statsd_type"`
	ObserverType string    `mapstructure:"observer_type"`
	Histogram    Histogram `mapstructure:"histogram"`
}

type Histogram struct {
	MaxSize int32 `mapstructure:"max_size"`
}

type LogConfig struct {
	Host               string              `mapstructure:"host"`
	Port               int                 `mapstructure:"port"`
	Database           string              `mapstructure:"database"`
	Username           string              `mapstructure:"username"`
	Password           configopaque.String `mapstructure:"password"`
	SSLMode            string              `mapstructure:"ssl_mode"`
	CollectionInterval time.Duration       `mapstructure:"collection_interval"`
}

func (cfg *Config) Validate() error {
	if !cfg.CollectionModes.RESTAPI && !cfg.CollectionModes.Database && !cfg.CollectionModes.StatsD && !cfg.CollectionModes.Logs {
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
			cfg.DatabaseConfig.CollectionInterval = 30 * time.Second
		}
		if cfg.DatabaseConfig.Port == 0 {
			cfg.DatabaseConfig.Port = 5432
		}
		if cfg.DatabaseConfig.SSLMode == "" {
			cfg.DatabaseConfig.SSLMode = "disable"
		}
		if cfg.DatabaseConfig.QueryTimeout <= 0 {
			cfg.DatabaseConfig.QueryTimeout = 15 * time.Second
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

	if cfg.CollectionModes.Logs {
		if cfg.LogConfig == nil {
			return errors.New("logs config required when logs mode enabled")
		}
		if cfg.LogConfig.Host == "" {
			return errors.New("logs database host must be specified")
		}
		if cfg.LogConfig.Database == "" {
			return errors.New("logs database name must be specified")
		}
		if cfg.LogConfig.Port == 0 {
			cfg.LogConfig.Port = 5432
		}
		if cfg.LogConfig.SSLMode == "" {
			cfg.LogConfig.SSLMode = "disable"
		}
		if cfg.LogConfig.CollectionInterval <= 0 {
			cfg.LogConfig.CollectionInterval = 30 * time.Second
		}
	}

	return nil
}
