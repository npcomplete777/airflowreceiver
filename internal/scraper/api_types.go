// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import "time"

// API Response types with ALL available fields for complete dimensions

type DAGResponse struct {
	DAGs []DAG `json:"dags"`
}

type DAG struct {
	DAGID                    string   `json:"dag_id"`
	Description              string   `json:"description"`
	Owners                   []string `json:"owners"`
	IsPaused                 bool     `json:"is_paused"`
	IsActive                 bool     `json:"is_active"`
	Tags                     []Tag    `json:"tags"`
	ScheduleInterval         interface{} `json:"schedule_interval"`
	Fileloc                  string   `json:"fileloc"`
	MaxActiveRuns            int      `json:"max_active_runs"`
	MaxActiveTasks           int      `json:"max_active_tasks"`
	HasImportErrors          bool     `json:"has_import_errors"`
	HasTaskConcurrencyLimits bool     `json:"has_task_concurrency_limits"`
}

type Tag struct {
	Name string `json:"name"`
}

type DAGRunsResponse struct {
	DAGRuns      []DAGRun `json:"dag_runs"`
	TotalEntries int      `json:"total_entries"`
}

type DAGRun struct {
	DAGID                 string                 `json:"dag_id"`
	DAGRunID              string                 `json:"dag_run_id"` // This is the key field!
	State                 string                 `json:"state"`
	StartDate             time.Time              `json:"start_date"`
	EndDate               time.Time              `json:"end_date"`
	ExecutionDate         time.Time              `json:"execution_date"`
	LogicalDate           time.Time              `json:"logical_date"`
	DataIntervalStart     time.Time              `json:"data_interval_start"`
	DataIntervalEnd       time.Time              `json:"data_interval_end"`
	RunType               string                 `json:"run_type"`
	ExternalTrigger       bool                   `json:"external_trigger"`
	Conf                  map[string]interface{} `json:"conf"`
	LastSchedulingDecision time.Time             `json:"last_scheduling_decision"`
}

type TaskInstancesResponse struct {
	TaskInstances []TaskInstance `json:"task_instances"`
	TotalEntries  int            `json:"total_entries"`
}

type TaskInstance struct {
	TaskID         string    `json:"task_id"`
	DAGID          string    `json:"dag_id"`
	DAGRunID       string    `json:"dag_run_id"` // Also using dag_run_id!
	State          string    `json:"state"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	Duration       float64   `json:"duration"`
	PoolSlots      int       `json:"pool_slots"`
	Pool           string    `json:"pool"`
	Queue          string    `json:"queue"`
	Operator       string    `json:"operator"` // Very useful dimension!
	Executor       string    `json:"executor_config"`
	TryNumber      int       `json:"try_number"`
	MaxTries       int       `json:"max_tries"`
	PriorityWeight int       `json:"priority_weight"`
	MapIndex       int       `json:"map_index"`
	Hostname       string    `json:"hostname"`
	Unixname       string    `json:"unixname"`
}

type PoolsResponse struct {
	Pools []Pool `json:"pools"`
}

type Pool struct {
	Name           string `json:"name"`
	Slots          int    `json:"slots"`
	OccupiedSlots  int    `json:"occupied_slots"`
	RunningSlots   int    `json:"running_slots"`
	QueuedSlots    int    `json:"queued_slots"`
	OpenSlots      int    `json:"open_slots"`
	DeferredSlots  int    `json:"deferred_slots"`
	ScheduledSlots int    `json:"scheduled_slots"`
	Description    string `json:"description"`
}

type HealthResponse struct {
	Metadatabase struct {
		Status string `json:"status"`
	} `json:"metadatabase"`
	Scheduler struct {
		Status                   string    `json:"status"`
		LatestSchedulerHeartbeat time.Time `json:"latest_scheduler_heartbeat"`
	} `json:"scheduler"`
	DAGProcessor struct {
		Status                       string    `json:"status"`
		LatestDAGProcessorHeartbeat time.Time `json:"latest_dag_processor_heartbeat"`
	} `json:"dag_processor"`
	Triggerer struct {
		Status                    string    `json:"status"`
		LatestTriggererHeartbeat time.Time `json:"latest_triggerer_heartbeat"`
	} `json:"triggerer"`
}

type ConnectionsResponse struct {
	Connections  []Connection `json:"connections"`
	TotalEntries int          `json:"total_entries"`
}

type Connection struct {
	ConnectionID string `json:"connection_id"`
	ConnType     string `json:"conn_type"`
	Description  string `json:"description"`
	Host         string `json:"host"`
	Port         *int   `json:"port"`
}

type VariablesResponse struct {
	Variables    []Variable `json:"variables"`
	TotalEntries int        `json:"total_entries"`
}

type Variable struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type ImportErrorsResponse struct {
	ImportErrors []ImportError `json:"import_errors"`
	TotalEntries int           `json:"total_entries"`
}

type ImportError struct {
	ImportErrorID int       `json:"import_error_id"`
	Filename      string    `json:"filename"`
	StackTrace    string    `json:"stack_trace"`
	Timestamp     time.Time `json:"timestamp"`
}
