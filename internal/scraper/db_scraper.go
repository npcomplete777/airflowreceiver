// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	_ "github.com/lib/pq"
)

type DatabaseScraper struct {
	cfg         *DatabaseConfig
	settings    receiver.Settings
	db          *sql.DB
	mb          *MetricsBuilder
	retryConfig RetryConfig
}

type DatabaseConfig struct {
	Host               string
	Port               int
	Database           string
	Username           string
	Password           string
	SSLMode            string
	CollectionInterval time.Duration
}

// Database query result types
type TaskInstanceStats struct {
	DAGID         string
	TaskID        string
	State         string
	Operator      string
	Pool          string
	Queue         string
	Count         int64
	AvgDuration   float64
	MaxDuration   float64
	MinDuration   float64
	TotalDuration float64
}

type DAGRunStats struct {
	DAGID       string
	State       string
	Count       int64
	AvgDuration float64
	MaxDuration float64
}

type SchedulerMetrics struct {
	ScheduledTasks  int64
	QueuedTasks     int64
	RunningTasks    int64
	SuccessTasks24h int64
	FailedTasks24h  int64
	OrphanedTasks   int64
}

func NewDatabaseScraper(cfg *DatabaseConfig, settings receiver.Settings) *DatabaseScraper {
	return &DatabaseScraper{
		cfg:         cfg,
		settings:    settings,
		mb:          NewMetricsBuilder(),
		retryConfig: DefaultRetryConfig(),
	}
}

func (s *DatabaseScraper) Start(ctx context.Context, host component.Host) error {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		s.cfg.Host,
		s.cfg.Port,
		s.cfg.Username,
		s.cfg.Password,
		s.cfg.Database,
		s.cfg.SSLMode,
	)
	
	var db *sql.DB
	err := RetryWithBackoff(ctx, s.retryConfig, s.settings.Logger, "database connection", func() error {
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		
		// Configure connection pool
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(1 * time.Minute)
		
		// Test connection
		if err := db.PingContext(ctx); err != nil {
			db.Close()
			return fmt.Errorf("failed to ping database: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}
	
	s.db = db
	s.settings.Logger.Info("Connected to Airflow database",
		zap.String("host", s.cfg.Host),
		zap.String("database", s.cfg.Database))
	
	return nil
}

func (s *DatabaseScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	now := pcommon.NewTimestampFromTime(time.Now())
	
	// Query 1: Task instance statistics
	if err := s.scrapeTaskInstanceStats(ctx, now); err != nil {
		s.settings.Logger.Warn("Failed to scrape task instance stats", zap.Error(err))
	}
	
	// Query 2: DAG run statistics
	if err := s.scrapeDAGRunStats(ctx, now); err != nil {
		s.settings.Logger.Warn("Failed to scrape DAG run stats", zap.Error(err))
	}
	
	// Query 3: Scheduler metrics
	if err := s.scrapeSchedulerMetrics(ctx, now); err != nil {
		s.settings.Logger.Warn("Failed to scrape scheduler metrics", zap.Error(err))
	}
	
	// Query 4: SLA misses
	if err := s.scrapeSLAMisses(ctx, now); err != nil {
		s.settings.Logger.Warn("Failed to scrape SLA misses", zap.Error(err))
	}
	
	return s.mb.Emit(), nil
}

func (s *DatabaseScraper) scrapeTaskInstanceStats(ctx context.Context, ts pcommon.Timestamp) error {
	query := `
		SELECT 
			dag_id,
			task_id,
			state,
			operator,
			pool,
			queue,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (end_date - start_date))) as avg_duration,
			MAX(EXTRACT(EPOCH FROM (end_date - start_date))) as max_duration,
			MIN(EXTRACT(EPOCH FROM (end_date - start_date))) as min_duration
		FROM task_instance
		WHERE start_date >= NOW() - INTERVAL '24 hours'
			AND end_date IS NOT NULL
		GROUP BY dag_id, task_id, state, operator, pool, queue
		ORDER BY count DESC
		LIMIT 1000
	`
	
	var rows *sql.Rows
	err := RetryWithBackoff(ctx, s.retryConfig, s.settings.Logger, "query task instances", func() error {
		var err error
		rows, err = s.db.QueryContext(ctx, query)
		return err
	})
	
	if err != nil {
		return err
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		var stats TaskInstanceStats
		if err := rows.Scan(
			&stats.DAGID,
			&stats.TaskID,
			&stats.State,
			&stats.Operator,
			&stats.Pool,
			&stats.Queue,
			&stats.Count,
			&stats.AvgDuration,
			&stats.MaxDuration,
			&stats.MinDuration,
		); err != nil {
			continue
		}
		
		// Record aggregated metrics
		s.mb.RecordTaskInstanceCountDB(stats.Count, stats.DAGID, stats.TaskID, stats.State, stats.Operator, stats.Pool, time.Now())
		
		if stats.AvgDuration > 0 {
			s.mb.RecordTaskInstanceAvgDuration(stats.AvgDuration, stats.DAGID, stats.TaskID, stats.State, time.Now())
			s.mb.RecordTaskInstanceMaxDuration(stats.MaxDuration, stats.DAGID, stats.TaskID, stats.State, time.Now())
		}
		count++
	}
	
	s.settings.Logger.Info("Scraped task instance stats from DB", zap.Int("records", count))
	return rows.Err()
}

func (s *DatabaseScraper) scrapeDAGRunStats(ctx context.Context, ts pcommon.Timestamp) error {
	query := `
		SELECT 
			dag_id,
			state,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (end_date - start_date))) as avg_duration,
			MAX(EXTRACT(EPOCH FROM (end_date - start_date))) as max_duration
		FROM dag_run
		WHERE start_date >= NOW() - INTERVAL '24 hours'
			AND end_date IS NOT NULL
		GROUP BY dag_id, state
		ORDER BY count DESC
	`
	
	var rows *sql.Rows
	err := RetryWithBackoff(ctx, s.retryConfig, s.settings.Logger, "query dag runs", func() error {
		var err error
		rows, err = s.db.QueryContext(ctx, query)
		return err
	})
	
	if err != nil {
		return err
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		var stats DAGRunStats
		if err := rows.Scan(
			&stats.DAGID,
			&stats.State,
			&stats.Count,
			&stats.AvgDuration,
			&stats.MaxDuration,
		); err != nil {
			continue
		}
		
		s.mb.RecordDAGRunCountDB(stats.Count, stats.DAGID, stats.State, time.Now())
		
		if stats.AvgDuration > 0 {
			s.mb.RecordDAGRunAvgDuration(stats.AvgDuration, stats.DAGID, stats.State, time.Now())
		}
		count++
	}
	
	s.settings.Logger.Info("Scraped DAG run stats from DB", zap.Int("records", count))
	return rows.Err()
}

func (s *DatabaseScraper) scrapeSchedulerMetrics(ctx context.Context, ts pcommon.Timestamp) error {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE state = 'scheduled') as scheduled,
			COUNT(*) FILTER (WHERE state = 'queued') as queued,
			COUNT(*) FILTER (WHERE state = 'running') as running,
			COUNT(*) FILTER (WHERE state = 'success' AND start_date >= NOW() - INTERVAL '24 hours') as success_24h,
			COUNT(*) FILTER (WHERE state = 'failed' AND start_date >= NOW() - INTERVAL '24 hours') as failed_24h,
			COUNT(*) FILTER (WHERE state = 'running' AND start_date < NOW() - INTERVAL '1 hour') as orphaned
		FROM task_instance
	`
	
	var metrics SchedulerMetrics
	err := RetryWithBackoff(ctx, s.retryConfig, s.settings.Logger, "query scheduler metrics", func() error {
		return s.db.QueryRowContext(ctx, query).Scan(
			&metrics.ScheduledTasks,
			&metrics.QueuedTasks,
			&metrics.RunningTasks,
			&metrics.SuccessTasks24h,
			&metrics.FailedTasks24h,
			&metrics.OrphanedTasks,
		)
	})
	
	if err != nil {
		return err
	}
	
	s.mb.RecordSchedulerTasksScheduled(metrics.ScheduledTasks, time.Now())
	s.mb.RecordSchedulerTasksQueued(metrics.QueuedTasks, time.Now())
	s.mb.RecordSchedulerTasksRunning(metrics.RunningTasks, time.Now())
	s.mb.RecordSchedulerTasksSuccess24h(metrics.SuccessTasks24h, time.Now())
	s.mb.RecordSchedulerTasksFailed24h(metrics.FailedTasks24h, time.Now())
	s.mb.RecordSchedulerTasksOrphaned(metrics.OrphanedTasks, time.Now())
	
	s.settings.Logger.Info("Scraped scheduler metrics from DB",
		zap.Int64("queued", metrics.QueuedTasks),
		zap.Int64("running", metrics.RunningTasks))
	
	return nil
}

func (s *DatabaseScraper) scrapeSLAMisses(ctx context.Context, ts pcommon.Timestamp) error {
	query := `
		SELECT 
			dag_id,
			COUNT(*) as count
		FROM sla_miss
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		GROUP BY dag_id
	`
	
	var rows *sql.Rows
	err := RetryWithBackoff(ctx, s.retryConfig, s.settings.Logger, "query SLA misses", func() error {
		var err error
		rows, err = s.db.QueryContext(ctx, query)
		return err
	})
	
	if err != nil {
		return err
	}
	defer rows.Close()
	
	totalMisses := int64(0)
	for rows.Next() {
		var dagID string
		var count int64
		if err := rows.Scan(&dagID, &count); err != nil {
			continue
		}
		
		s.mb.RecordSLAMissCount(count, dagID, time.Now())
		totalMisses += count
	}
	
	if totalMisses > 0 {
		s.settings.Logger.Warn("SLA misses detected", zap.Int64("total", totalMisses))
	}
	
	return rows.Err()
}

func (s *DatabaseScraper) Shutdown(ctx context.Context) error {
	if s.db != nil {
		s.settings.Logger.Info("Closing database connections")
		return s.db.Close()
	}
	return nil
}
