// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"time"
	
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type MetricsBuilder struct {
	metrics pmetric.Metrics
	rm      pmetric.ResourceMetrics
	sm      pmetric.ScopeMetrics
}

func NewMetricsBuilder() *MetricsBuilder {
	m := pmetric.NewMetrics()
	rm := m.ResourceMetrics().AppendEmpty()
	
	rm.Resource().Attributes().PutStr("service.name", "airflow")
	rm.Resource().Attributes().PutStr("airflow.component", "receiver")
	
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("github.com/npcomplete777/airflowreceiver")
	sm.Scope().SetVersion("0.0.1")
	
	return &MetricsBuilder{
		metrics: m,
		rm:      rm,
		sm:      sm,
	}
}

func (mb *MetricsBuilder) RecordDAGRunDuration(value float64, dagID, runID, runType, state string, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag.run.duration")
	metric.SetUnit("s")
	metric.SetDescription("Duration of DAG run execution")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetDoubleValue(value)
	
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("run.id", runID)
	dp.Attributes().PutStr("run.type", runType)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordDAGRunCount(value int64, dagID, state string, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag.run.count")
	metric.SetUnit("{runs}")
	metric.SetDescription("Number of DAG runs by state")
	
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := sum.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetIntValue(value)
	
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordTaskInstanceDuration(value float64, dagID, taskID, runID, state string, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.task.instance.duration")
	metric.SetUnit("s")
	metric.SetDescription("Duration of task instance execution")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetDoubleValue(value)
	
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("task.id", taskID)
	dp.Attributes().PutStr("run.id", runID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordPoolSlotsOpen(value int64, poolName string, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.open")
	metric.SetUnit("{slots}")
	metric.SetDescription("Number of open slots in pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetIntValue(value)
	
	dp.Attributes().PutStr("pool.name", poolName)
}

func (mb *MetricsBuilder) RecordPoolSlotsUsed(value int64, poolName string, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.used")
	metric.SetUnit("{slots}")
	metric.SetDescription("Number of used slots in pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetIntValue(value)
	
	dp.Attributes().PutStr("pool.name", poolName)
}

func (mb *MetricsBuilder) RecordSchedulerHealth(status string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.health")
	metric.SetUnit("{status}")
	metric.SetDescription("Scheduler health status (1=healthy, 0=unhealthy)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	
	healthValue := int64(0)
	if status == "healthy" {
		healthValue = 1
	}
	dp.SetIntValue(healthValue)
	dp.Attributes().PutStr("status", status)
}

func (mb *MetricsBuilder) RecordDatabaseHealth(status string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.database.health")
	metric.SetUnit("{status}")
	metric.SetDescription("Database health status (1=healthy, 0=unhealthy)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	
	healthValue := int64(0)
	if status == "healthy" {
		healthValue = 1
	}
	dp.SetIntValue(healthValue)
	dp.Attributes().PutStr("status", status)
}

func (mb *MetricsBuilder) RecordSchedulerHeartbeatAge(age float64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.heartbeat.age")
	metric.SetUnit("s")
	metric.SetDescription("Age of scheduler heartbeat in seconds")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetDoubleValue(age)
}

func (mb *MetricsBuilder) RecordConnectionCount(count int64, connType string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.connections.count")
	metric.SetUnit("{connections}")
	metric.SetDescription("Number of connections by type")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("connection.type", connType)
}

func (mb *MetricsBuilder) RecordVariableCount(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.variables.count")
	metric.SetUnit("{variables}")
	metric.SetDescription("Total number of Airflow variables")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordImportErrorCount(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.import_errors.count")
	metric.SetUnit("{errors}")
	metric.SetDescription("Number of DAG import errors")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordDAGCount(count int64, status string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dags.count")
	metric.SetUnit("{dags}")
	metric.SetDescription("Total number of DAGs by status")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("status", status)
}

func (mb *MetricsBuilder) RecordTaskInstancesByState(count int64, dagID, state string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.task_instances.by_state")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of task instances by state")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordDAGRunsByState(count int64, dagID, state string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag_runs.by_state")
	metric.SetUnit("{runs}")
	metric.SetDescription("Number of DAG runs by state")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordPoolQueuedSlots(value int64, poolName string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.queued")
	metric.SetUnit("{slots}")
	metric.SetDescription("Number of queued slots in pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(value)
	dp.Attributes().PutStr("pool.name", poolName)
}

func (mb *MetricsBuilder) RecordPoolRunningSlots(value int64, poolName string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.running")
	metric.SetUnit("{slots}")
	metric.SetDescription("Number of running slots in pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(value)
	dp.Attributes().PutStr("pool.name", poolName)
}


// Additional dimensional metrics

func (mb *MetricsBuilder) RecordTaskInstanceDurationWithDimensions(value float64, dagID, taskID, dagRunID, state, operator, pool, queue string, tryNumber int, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.task.instance.duration")
	metric.SetUnit("s")
	metric.SetDescription("Duration of task instance execution with full dimensions")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetDoubleValue(value)
	
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("task.id", taskID)
	dp.Attributes().PutStr("dag_run.id", dagRunID)
	dp.Attributes().PutStr("state", state)
	dp.Attributes().PutStr("operator", operator)
	dp.Attributes().PutStr("pool", pool)
	dp.Attributes().PutStr("queue", queue)
	dp.Attributes().PutInt("try_number", int64(tryNumber))
}

func (mb *MetricsBuilder) RecordDAGRunDurationWithDimensions(value float64, dagID, dagRunID, runType, state string, externalTrigger bool, ts pcommon.Timestamp) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag.run.duration")
	metric.SetUnit("s")
	metric.SetDescription("Duration of DAG run execution with full dimensions")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(ts)
	dp.SetDoubleValue(value)
	
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("dag_run.id", dagRunID)
	dp.Attributes().PutStr("run.type", runType)
	dp.Attributes().PutStr("state", state)
	dp.Attributes().PutBool("external_trigger", externalTrigger)
}

func (mb *MetricsBuilder) RecordPoolTotalSlots(value int64, poolName, description string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.total")
	metric.SetUnit("{slots}")
	metric.SetDescription("Total capacity of pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(value)
	dp.Attributes().PutStr("pool.name", poolName)
	if description != "" {
		dp.Attributes().PutStr("pool.description", description)
	}
}

func (mb *MetricsBuilder) RecordPoolDeferredSlots(value int64, poolName string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.deferred")
	metric.SetUnit("{slots}")
	metric.SetDescription("Number of deferred slots in pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(value)
	dp.Attributes().PutStr("pool.name", poolName)
}

func (mb *MetricsBuilder) RecordPoolScheduledSlots(value int64, poolName string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.pool.slots.scheduled")
	metric.SetUnit("{slots}")
	metric.SetDescription("Number of scheduled slots in pool")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(value)
	dp.Attributes().PutStr("pool.name", poolName)
}

func (mb *MetricsBuilder) RecordDAGWithTags(count int64, dagID string, tags []string, isPaused bool, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag.info")
	metric.SetUnit("{dag}")
	metric.SetDescription("DAG information with tags")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutBool("is_paused", isPaused)
	
	// Add tags as a comma-separated string
	if len(tags) > 0 {
		tagStr := ""
		for i, tag := range tags {
			if i > 0 {
				tagStr += ","
			}
			tagStr += tag
		}
		dp.Attributes().PutStr("tags", tagStr)
	}
}

// Database-sourced metrics

func (mb *MetricsBuilder) RecordTaskInstanceCountDB(count int64, dagID, taskID, state, operator, pool string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.task.instance.count.db")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Task instance count from database aggregation (24h)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("task.id", taskID)
	dp.Attributes().PutStr("state", state)
	dp.Attributes().PutStr("operator", operator)
	dp.Attributes().PutStr("pool", pool)
}

func (mb *MetricsBuilder) RecordTaskInstanceAvgDuration(avg float64, dagID, taskID, state string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.task.instance.duration.avg")
	metric.SetUnit("s")
	metric.SetDescription("Average task instance duration (24h)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetDoubleValue(avg)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("task.id", taskID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordTaskInstanceMaxDuration(max float64, dagID, taskID, state string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.task.instance.duration.max")
	metric.SetUnit("s")
	metric.SetDescription("Maximum task instance duration (24h)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetDoubleValue(max)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("task.id", taskID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordDAGRunCountDB(count int64, dagID, state string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag.run.count.db")
	metric.SetUnit("{runs}")
	metric.SetDescription("DAG run count from database (24h)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordDAGRunAvgDuration(avg float64, dagID, state string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.dag.run.duration.avg")
	metric.SetUnit("s")
	metric.SetDescription("Average DAG run duration (24h)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetDoubleValue(avg)
	dp.Attributes().PutStr("dag.id", dagID)
	dp.Attributes().PutStr("state", state)
}

func (mb *MetricsBuilder) RecordSchedulerTasksScheduled(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.tasks.scheduled")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of scheduled tasks")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordSchedulerTasksQueued(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.tasks.queued")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of queued tasks")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordSchedulerTasksRunning(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.tasks.running")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of running tasks")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordSchedulerTasksSuccess24h(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.tasks.success.24h")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of successful tasks in last 24 hours")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordSchedulerTasksFailed24h(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.tasks.failed.24h")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of failed tasks in last 24 hours")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordSchedulerTasksOrphaned(count int64, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.scheduler.tasks.orphaned")
	metric.SetUnit("{tasks}")
	metric.SetDescription("Number of orphaned tasks (running >1 hour)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
}

func (mb *MetricsBuilder) RecordSLAMissCount(count int64, dagID string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName("airflow.sla.miss.count")
	metric.SetUnit("{misses}")
	metric.SetDescription("Number of SLA misses (24h)")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(count)
	dp.Attributes().PutStr("dag.id", dagID)
}

// Generic metrics for StatsD (dynamic metric names)

func (mb *MetricsBuilder) RecordGenericCounter(value int64, metricName string, tags map[string]string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName(metricName)
	metric.SetUnit("{count}")
	metric.SetDescription("StatsD counter metric")
	
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := sum.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetIntValue(value)
	
	for k, v := range tags {
		dp.Attributes().PutStr(k, v)
	}
}

func (mb *MetricsBuilder) RecordGenericGauge(value float64, metricName string, tags map[string]string, ts time.Time) {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName(metricName)
	metric.SetUnit("{value}")
	metric.SetDescription("StatsD gauge metric")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetDoubleValue(value)
	
	for k, v := range tags {
		dp.Attributes().PutStr(k, v)
	}
}

func (mb *MetricsBuilder) RecordGenericTimer(avg, min, max float64, metricName string, tags map[string]string, ts time.Time) {
	// Average
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName(metricName + ".avg")
	metric.SetUnit("ms")
	metric.SetDescription("StatsD timer average")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dp.SetDoubleValue(avg)
	
	for k, v := range tags {
		dp.Attributes().PutStr(k, v)
	}
	
	// Min
	metricMin := mb.sm.Metrics().AppendEmpty()
	metricMin.SetName(metricName + ".min")
	metricMin.SetUnit("ms")
	
	gaugeMin := metricMin.SetEmptyGauge()
	dpMin := gaugeMin.DataPoints().AppendEmpty()
	dpMin.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dpMin.SetDoubleValue(min)
	
	for k, v := range tags {
		dpMin.Attributes().PutStr(k, v)
	}
	
	// Max
	metricMax := mb.sm.Metrics().AppendEmpty()
	metricMax.SetName(metricName + ".max")
	metricMax.SetUnit("ms")
	
	gaugeMax := metricMax.SetEmptyGauge()
	dpMax := gaugeMax.DataPoints().AppendEmpty()
	dpMax.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	dpMax.SetDoubleValue(max)
	
	for k, v := range tags {
		dpMax.Attributes().PutStr(k, v)
	}
}


// Emit returns the accumulated metrics
func (mb *MetricsBuilder) Emit() pmetric.Metrics {
	return mb.metrics
}
