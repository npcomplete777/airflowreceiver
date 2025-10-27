// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func (s *RESTAPIScraper) scrapeComprehensive(ctx context.Context, now time.Time) {
	ts := pcommon.NewTimestampFromTime(now)
	
	s.scrapeHealthMetrics(ctx, ts)
	s.scrapeDAGMetrics(ctx, ts)
	
	pools, err := s.getPools(ctx)
	if err == nil {
		s.recordEnhancedPoolMetrics(pools, ts)
	}
	
	s.scrapeConnectionMetrics(ctx, ts)
	s.scrapeConfigMetrics(ctx, ts)
}

func (s *RESTAPIScraper) scrapeHealthMetrics(ctx context.Context, ts pcommon.Timestamp) {
	health, err := s.getHealth(ctx)
	if err != nil {
		s.settings.Logger.Warn("Failed to get health", zap.Error(err))
		return
	}
	
	s.mb.RecordSchedulerHealth(health.Scheduler.Status, time.Now())
	s.mb.RecordDatabaseHealth(health.Metadatabase.Status, time.Now())
	
	if !health.Scheduler.LatestSchedulerHeartbeat.IsZero() {
		heartbeatAge := time.Since(health.Scheduler.LatestSchedulerHeartbeat).Seconds()
		s.mb.RecordSchedulerHeartbeatAge(heartbeatAge, time.Now())
	}
}

func (s *RESTAPIScraper) scrapeDAGMetrics(ctx context.Context, ts pcommon.Timestamp) {
	dags, err := s.getDags(ctx)
	if err != nil {
		s.settings.Logger.Error("Failed to get DAGs", zap.Error(err))
		return
	}
	
	s.settings.Logger.Info("Scraping comprehensive DAG metrics", zap.Int("dag_count", len(dags)))
	
	// Count DAGs by status and record with tags
	pausedCount := int64(0)
	activeCount := int64(0)
	for _, dag := range dags {
		// Extract tag names
		tagNames := make([]string, len(dag.Tags))
		for i, tag := range dag.Tags {
			tagNames[i] = tag.Name
		}
		
		// Record DAG info with tags
		s.mb.RecordDAGWithTags(1, dag.DAGID, tagNames, dag.IsPaused, time.Now())
		
		if dag.IsPaused {
			pausedCount++
		} else {
			activeCount++
		}
	}
	s.mb.RecordDAGCount(pausedCount, "paused", time.Now())
	s.mb.RecordDAGCount(activeCount, "active", time.Now())
	
	// For each DAG, get runs
	for _, dag := range dags {
		dagRuns, err := s.getDAGRuns(ctx, dag.DAGID)
		if err != nil {
			continue
		}
		
		runsByState := make(map[string]int64)
		for _, run := range dagRuns {
			// Use DAGRunID not RunID!
			if run.DAGRunID == "" {
				s.settings.Logger.Warn("Empty dag_run_id, skipping",
					zap.String("dag_id", run.DAGID),
					zap.String("state", run.State))
				continue
			}
			
			runsByState[run.State]++
			
			// Record duration with full dimensions
			if (run.State == "success" || run.State == "failed") && !run.EndDate.IsZero() && !run.StartDate.IsZero() {
				duration := run.EndDate.Sub(run.StartDate).Seconds()
				if duration > 0 {
					s.mb.RecordDAGRunDurationWithDimensions(
						duration,
						run.DAGID,
						run.DAGRunID,
						run.RunType,
						run.State,
						run.ExternalTrigger,
						ts,
					)
				}
			}
		}
		
		for state, count := range runsByState {
			s.mb.RecordDAGRunsByState(count, dag.DAGID, state, time.Now())
		}
		
		// Get task instances for recent/running runs
		for _, run := range dagRuns {
			if run.DAGRunID == "" {
				continue
			}
			
			if run.State == "running" || time.Since(run.StartDate) < 5*time.Minute {
				tasks, err := s.getTaskInstances(ctx, dag.DAGID, run.DAGRunID)
				if err != nil {
					continue
				}
				
				tasksByState := make(map[string]int64)
				for _, task := range tasks {
					tasksByState[task.State]++
					
					// Record with ALL dimensions
					if task.Duration > 0 && task.TaskID != "" && task.DAGRunID != "" {
						s.mb.RecordTaskInstanceDurationWithDimensions(
							task.Duration,
							task.DAGID,
							task.TaskID,
							task.DAGRunID,
							task.State,
							task.Operator,
							task.Pool,
							task.Queue,
							task.TryNumber,
							ts,
						)
					}
				}
				
				for state, count := range tasksByState {
					s.mb.RecordTaskInstancesByState(count, dag.DAGID, state, time.Now())
				}
			}
		}
	}
}

func (s *RESTAPIScraper) recordEnhancedPoolMetrics(pools []Pool, ts pcommon.Timestamp) {
	for _, pool := range pools {
		if pool.Name == "" {
			continue
		}
		
		s.mb.RecordPoolSlotsOpen(int64(pool.OpenSlots), pool.Name, ts)
		s.mb.RecordPoolSlotsUsed(int64(pool.OccupiedSlots), pool.Name, ts)
		s.mb.RecordPoolQueuedSlots(int64(pool.QueuedSlots), pool.Name, time.Now())
		s.mb.RecordPoolRunningSlots(int64(pool.RunningSlots), pool.Name, time.Now())
		s.mb.RecordPoolTotalSlots(int64(pool.Slots), pool.Name, pool.Description, time.Now())
		s.mb.RecordPoolDeferredSlots(int64(pool.DeferredSlots), pool.Name, time.Now())
		s.mb.RecordPoolScheduledSlots(int64(pool.ScheduledSlots), pool.Name, time.Now())
	}
}

func (s *RESTAPIScraper) scrapeConnectionMetrics(ctx context.Context, ts pcommon.Timestamp) {
	connections, err := s.getConnections(ctx)
	if err != nil {
		s.settings.Logger.Warn("Failed to get connections", zap.Error(err))
		return
	}
	
	connByType := make(map[string]int64)
	for _, conn := range connections {
		if conn.ConnType != "" {
			connByType[conn.ConnType]++
		}
	}
	
	for connType, count := range connByType {
		s.mb.RecordConnectionCount(count, connType, time.Now())
	}
}

func (s *RESTAPIScraper) scrapeConfigMetrics(ctx context.Context, ts pcommon.Timestamp) {
	variables, err := s.getVariables(ctx)
	if err == nil {
		s.mb.RecordVariableCount(int64(len(variables)), time.Now())
	}
	
	importErrors, err := s.getImportErrors(ctx)
	if err == nil {
		s.mb.RecordImportErrorCount(int64(len(importErrors)), time.Now())
	}
}
