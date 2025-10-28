// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type LogsBuilder struct {
	logs plog.Logs
	rl   plog.ResourceLogs
	sl   plog.ScopeLogs
}

func NewLogsBuilder() *LogsBuilder {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	
	rl.Resource().Attributes().PutStr("service.name", "airflow")
	rl.Resource().Attributes().PutStr("airflow.component", "event_logs")
	
	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("github.com/npcomplete777/airflowreceiver/log_scraper")
	sl.Scope().SetVersion("0.0.1")
	
	return &LogsBuilder{
		logs: logs,
		rl:   rl,
		sl:   sl,
	}
}

func (lb *LogsBuilder) RecordEventLog(
	timestamp time.Time,
	dagID, taskID, event, owner string,
	executionDate time.Time,
	extra map[string]string,
) {
	lr := lb.sl.LogRecords().AppendEmpty()
	
	lr.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	// Set severity based on event type
	severity := getSeverityFromEvent(event)
	lr.SetSeverityNumber(severity)
	lr.SetSeverityText(getSeverityText(severity))
	
	// Body contains the event description
	if event != "" {
		lr.Body().SetStr(fmt.Sprintf("Airflow event: %s", event))
	}
	
	// Add structured attributes
	attrs := lr.Attributes()
	attrs.PutStr("airflow.log.source", "database")
	
	if dagID != "" {
		attrs.PutStr("dag.id", dagID)
	}
	if taskID != "" {
		attrs.PutStr("task.id", taskID)
	}
	if event != "" {
		attrs.PutStr("airflow.event", event)
	}
	if !executionDate.IsZero() {
		attrs.PutStr("execution.date", executionDate.Format(time.RFC3339))
	}
	if owner != "" {
		attrs.PutStr("owner", owner)
	}
	
	// Add extra fields
	for key, value := range extra {
		attrs.PutStr(fmt.Sprintf("extra.%s", key), value)
	}
}

func getSeverityFromEvent(event string) plog.SeverityNumber {
	switch event {
	case "failed", "failed_task":
		return plog.SeverityNumberError
	case "success", "success_task":
		return plog.SeverityNumberInfo
	case "running":
		return plog.SeverityNumberInfo
	case "skipped":
		return plog.SeverityNumberWarn
	case "up_for_retry", "retry":
		return plog.SeverityNumberWarn
	case "up_for_reschedule":
		return plog.SeverityNumberInfo
	default:
		return plog.SeverityNumberInfo
	}
}

func getSeverityText(severity plog.SeverityNumber) string {
	switch severity {
	case plog.SeverityNumberError:
		return "ERROR"
	case plog.SeverityNumberWarn:
		return "WARN"
	case plog.SeverityNumberInfo:
		return "INFO"
	default:
		return "INFO"
	}
}

func (lb *LogsBuilder) Emit() plog.Logs {
	return lb.logs
}
