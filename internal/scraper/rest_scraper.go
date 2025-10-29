// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type RESTAPIScraper struct {
	cfg         *RESTAPIConfig
	settings    receiver.Settings
	client      *http.Client
	mb          *MetricsBuilder
	retryConfig RetryConfig
	health      *ScraperHealth
}

type RESTAPIConfig struct {
	Endpoint           string
	Username           string
	Password           string
	CollectionInterval time.Duration
	IncludePastRuns    bool
	PastRunsLookback   time.Duration
}

func NewRESTAPIScraper(cfg *RESTAPIConfig, settings receiver.Settings) *RESTAPIScraper {
	return &RESTAPIScraper{
		cfg:         cfg,
		settings:    settings,
		client:      &http.Client{Timeout: 30 * time.Second},
		mb:          NewMetricsBuilder(),
		retryConfig: DefaultRetryConfig(),
		health:      NewScraperHealth("rest_api", settings.Logger),
	}
}

func (s *RESTAPIScraper) Start(ctx context.Context, host component.Host) error {
	s.settings.Logger.Info("Starting REST API scraper", zap.String("endpoint", s.cfg.Endpoint))
	return nil
}

func (s *RESTAPIScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// Use health tracking wrapper
	metrics, err := s.health.WithScrapeTracking(ctx, func(ctx context.Context) (pmetric.Metrics, error) {
		now := time.Now()
		s.scrapeComprehensive(ctx, now)
		return s.mb.Emit(), nil
	})
	
	// Add health metrics to output
	s.health.EmitMetrics(s.mb, time.Now())
	
	return metrics, err
}

func (s *RESTAPIScraper) Shutdown(ctx context.Context) error {
	s.settings.Logger.Info("Shutting down REST API scraper")
	return nil
}

func (s *RESTAPIScraper) doRequest(ctx context.Context, path string) ([]byte, error) {
	url := s.cfg.Endpoint + path
	
	var body []byte
	err := RetryWithBackoff(ctx, s.retryConfig, s.settings.Logger, fmt.Sprintf("GET %s", path), func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
		req.Header.Set("Accept", "application/json")
		
		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			// Don't retry authentication failures
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				body = nil
				return fmt.Errorf("authentication failed: status code %d", resp.StatusCode)
			}
			// Retry server errors
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		
		body, err = io.ReadAll(resp.Body)
		return err
	})
	
	return body, err
}

func (s *RESTAPIScraper) getDags(ctx context.Context) ([]DAG, error) {
	body, err := s.doRequest(ctx, "/api/v1/dags")
	if err != nil {
		return nil, err
	}
	
	var response DAGResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.DAGs, nil
}

func (s *RESTAPIScraper) getDAGRuns(ctx context.Context, dagID string) ([]DAGRun, error) {
	path := fmt.Sprintf("/api/v1/dags/%s/dagRuns?limit=100", dagID)
	if s.cfg.IncludePastRuns {
		startDate := time.Now().Add(-s.cfg.PastRunsLookback)
		path += fmt.Sprintf("&start_date_gte=%s", startDate.Format(time.RFC3339))
	}
	
	body, err := s.doRequest(ctx, path)
	if err != nil {
		return nil, err
	}
	
	var response DAGRunsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.DAGRuns, nil
}

func (s *RESTAPIScraper) getTaskInstances(ctx context.Context, dagID, dagRunID string) ([]TaskInstance, error) {
	path := fmt.Sprintf("/api/v1/dags/%s/dagRuns/%s/taskInstances", dagID, dagRunID)
	
	body, err := s.doRequest(ctx, path)
	if err != nil {
		return nil, err
	}
	
	var response TaskInstancesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.TaskInstances, nil
}

func (s *RESTAPIScraper) getPools(ctx context.Context) ([]Pool, error) {
	body, err := s.doRequest(ctx, "/api/v1/pools")
	if err != nil {
		return nil, err
	}
	
	var response PoolsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.Pools, nil
}

func (s *RESTAPIScraper) getHealth(ctx context.Context) (*HealthResponse, error) {
	body, err := s.doRequest(ctx, "/api/v1/health")
	if err != nil {
		return nil, err
	}
	
	var response HealthResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

func (s *RESTAPIScraper) getConnections(ctx context.Context) ([]Connection, error) {
	body, err := s.doRequest(ctx, "/api/v1/connections?limit=100")
	if err != nil {
		return nil, err
	}
	
	var response ConnectionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.Connections, nil
}

func (s *RESTAPIScraper) getVariables(ctx context.Context) ([]Variable, error) {
	body, err := s.doRequest(ctx, "/api/v1/variables?limit=100")
	if err != nil {
		return nil, err
	}
	
	var response VariablesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.Variables, nil
}

func (s *RESTAPIScraper) getImportErrors(ctx context.Context) ([]ImportError, error) {
	body, err := s.doRequest(ctx, "/api/v1/importErrors?limit=100")
	if err != nil {
		return nil, err
	}
	
	var response ImportErrorsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.ImportErrors, nil
}
