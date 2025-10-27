// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// StatsDConfig for scraper
type StatsDConfig struct {
	Endpoint            string
	AggregationInterval time.Duration
}

// StatsDMetric represents an aggregated StatsD metric
type StatsDMetric struct {
	Name       string
	Value      float64
	Type       string // counter, gauge, timer
	SampleRate float64
	Tags       map[string]string
	Count      int64
	Sum        float64
	Min        float64
	Max        float64
}

type StatsDScraper struct {
	cfg      *StatsDConfig
	settings receiver.Settings
	conn     *net.UDPConn
	mb       *MetricsBuilder
	
	mu      sync.RWMutex
	metrics map[string]*StatsDMetric
	
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewStatsDScraper(cfg *StatsDConfig, settings receiver.Settings) *StatsDScraper {
	return &StatsDScraper{
		cfg:      cfg,
		settings: settings,
		mb:       NewMetricsBuilder(),
		metrics:  make(map[string]*StatsDMetric),
		stopChan: make(chan struct{}),
	}
}

func (s *StatsDScraper) Start(ctx context.Context, host component.Host) error {
	s.settings.Logger.Info("Starting StatsD scraper", 
		zap.String("endpoint", s.cfg.Endpoint),
		zap.Duration("aggregation_interval", s.cfg.AggregationInterval))
	
	addr, err := net.ResolveUDPAddr("udp", s.cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}
	
	s.conn = conn
	s.wg.Add(1)
	go s.listen()
	
	s.settings.Logger.Info("StatsD receiver started successfully")
	return nil
}

func (s *StatsDScraper) listen() {
	defer s.wg.Done()
	buf := make([]byte, 65535)
	
	for {
		select {
		case <-s.stopChan:
			return
		default:
			s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, _, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				s.settings.Logger.Error("Error reading from UDP", zap.Error(err))
				continue
			}
			s.parseAndAggregate(string(buf[:n]))
		}
	}
}

func (s *StatsDScraper) parseAndAggregate(data string) {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		metric := s.parseStatsDLine(line)
		if metric != nil {
			s.aggregate(metric)
		}
	}
}

func (s *StatsDScraper) parseStatsDLine(line string) *StatsDMetric {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil
	}
	
	nameValue := strings.SplitN(parts[0], ":", 2)
	if len(nameValue) != 2 {
		return nil
	}
	
	value, err := strconv.ParseFloat(nameValue[1], 64)
	if err != nil {
		return nil
	}
	
	metric := &StatsDMetric{
		Name:       nameValue[0],
		Value:      value,
		Type:       parts[1],
		SampleRate: 1.0,
		Tags:       make(map[string]string),
	}
	
	for i := 2; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "@") {
			if rate, err := strconv.ParseFloat(parts[i][1:], 64); err == nil {
				metric.SampleRate = rate
			}
		} else if strings.HasPrefix(parts[i], "#") {
			tagPairs := strings.Split(parts[i][1:], ",")
			for _, pair := range tagPairs {
				kv := strings.SplitN(pair, ":", 2)
				if len(kv) == 2 {
					metric.Tags[kv[0]] = kv[1]
				}
			}
		}
	}
	
	return metric
}

func (s *StatsDScraper) aggregate(metric *StatsDMetric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := metric.Name
	for k, v := range metric.Tags {
		key += fmt.Sprintf(",%s=%s", k, v)
	}
	
	existing, exists := s.metrics[key]
	if !exists {
		s.metrics[key] = &StatsDMetric{
			Name:  metric.Name,
			Type:  metric.Type,
			Tags:  metric.Tags,
			Value: metric.Value,
			Count: 1,
			Sum:   metric.Value,
			Min:   metric.Value,
			Max:   metric.Value,
		}
		return
	}
	
	switch metric.Type {
	case "c":
		existing.Value += metric.Value / metric.SampleRate
	case "g":
		existing.Value = metric.Value
	case "ms", "h":
		existing.Count++
		existing.Sum += metric.Value
		if metric.Value < existing.Min {
			existing.Min = metric.Value
		}
		if metric.Value > existing.Max {
			existing.Max = metric.Value
		}
	}
}

func (s *StatsDScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, metric := range s.metrics {
		switch metric.Type {
		case "c":
			s.mb.RecordGenericCounter(int64(metric.Value), metric.Name, metric.Tags, time.Now())
		case "g":
			s.mb.RecordGenericGauge(metric.Value, metric.Name, metric.Tags, time.Now())
		case "ms", "h":
			if metric.Count > 0 {
				avg := metric.Sum / float64(metric.Count)
				s.mb.RecordGenericTimer(avg, metric.Min, metric.Max, metric.Name, metric.Tags, time.Now())
			}
		}
	}
	
	s.settings.Logger.Debug("Scraped StatsD metrics", zap.Int("metric_count", len(s.metrics)))
	return s.mb.Emit(), nil
}

func (s *StatsDScraper) Shutdown(ctx context.Context) error {
	s.settings.Logger.Info("Shutting down StatsD scraper")
	close(s.stopChan)
	if s.conn != nil {
		s.conn.Close()
	}
	s.wg.Wait()
	return nil
}
