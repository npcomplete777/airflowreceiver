// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	_ "github.com/lib/pq"
)

type LogScraper struct {
	cfg              *LogScraperConfig
	settings         receiver.Settings
	db               *sql.DB
	lb               *LogsBuilder
	lastScrapedLogID int64
}

type LogScraperConfig struct {
	Host               string
	Port               int
	Database           string
	Username           string
	Password           string
	SSLMode            string
	CollectionInterval time.Duration
}

func NewLogScraper(cfg *LogScraperConfig, settings receiver.Settings) *LogScraper {
	return &LogScraper{
		cfg:              cfg,
		settings:         settings,
		lb:               NewLogsBuilder(),
		lastScrapedLogID: 0,
	}
}

func (s *LogScraper) Start(ctx context.Context, host component.Host) error {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		s.cfg.Host,
		s.cfg.Port,
		s.cfg.Username,
		s.cfg.Password,
		s.cfg.Database,
		s.cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	s.db = db
	s.settings.Logger.Info("Log scraper database connection established",
		zap.String("host", s.cfg.Host),
		zap.Int("port", s.cfg.Port),
		zap.String("database", s.cfg.Database))

	return nil
}

func (s *LogScraper) Scrape(ctx context.Context) (plog.Logs, error) {
	// Create fresh builder for each scrape
	s.lb = NewLogsBuilder()
	
	query := `
		SELECT id, dttm, dag_id, task_id, event, execution_date, owner, extra
		FROM log
		WHERE id > $1
		ORDER BY id ASC
		LIMIT 1000
	`

	rows, err := s.db.QueryContext(ctx, query, s.lastScrapedLogID)
	if err != nil {
		return s.lb.Emit(), fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	logCount := 0
	for rows.Next() {
		var (
			id            int64
			dttm          time.Time
			dagID         sql.NullString
			taskID        sql.NullString
			event         sql.NullString
			executionDate sql.NullTime
			owner         sql.NullString
			extra         sql.NullString
		)

		if err := rows.Scan(&id, &dttm, &dagID, &taskID, &event, &executionDate, &owner, &extra); err != nil {
			s.settings.Logger.Warn("Failed to scan log row", zap.Error(err))
			continue
		}

		// Parse extra JSON if present
		extraMap := make(map[string]string)
		if extra.Valid && extra.String != "" {
			var extraData map[string]interface{}
			if err := json.Unmarshal([]byte(extra.String), &extraData); err == nil {
				for key, value := range extraData {
					if strVal, ok := value.(string); ok {
						extraMap[key] = strVal
					}
				}
			}
		}

		// Record the log event
		s.lb.RecordEventLog(
			dttm,
			dagID.String,
			taskID.String,
			event.String,
			owner.String,
			executionDate.Time,
			extraMap,
		)

		// Track highest log ID
		if id > s.lastScrapedLogID {
			s.lastScrapedLogID = id
		}

		logCount++
	}

	if err := rows.Err(); err != nil {
		return s.lb.Emit(), fmt.Errorf("error iterating log rows: %w", err)
	}

	s.settings.Logger.Debug("Scraped event logs",
		zap.Int("count", logCount),
		zap.Int64("last_log_id", s.lastScrapedLogID))

	return s.lb.Emit(), nil
}

func (s *LogScraper) Shutdown(ctx context.Context) error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
