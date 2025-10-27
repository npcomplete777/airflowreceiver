# Apache Airflow OpenTelemetry Receiver

An OpenTelemetry collector receiver for Apache Airflow that collects metrics from multiple data sources.

**Status:** Alpha  
**OTEL Version:** 0.136.0  
**Airflow Compatibility:** 2.0+

## Overview

This receiver implements three independent collection mechanisms:

1. **REST API Scraper** - Polls Airflow's REST API endpoints
2. **Database Scraper** - Queries the Airflow metadata database directly
3. **StatsD Receiver** - Listens for StatsD metrics emitted by Airflow

Each mode can be enabled independently or in combination depending on deployment requirements and access constraints.

## Architecture
```
┌─────────────────────────────────────┐
│         Apache Airflow              │
│  REST API │ PostgreSQL │ StatsD     │
└─────┬──────────┬─────────┬──────────┘
      │          │         │
      ▼          ▼         ▼
┌─────────────────────────────────────┐
│    Airflow OTEL Receiver            │
│  ┌─────────┐ ┌─────────┐ ┌───────┐ │
│  │  REST   │ │   DB    │ │StatsD │ │
│  │ Scraper │ │ Scraper │ │Scraper│ │
│  └─────────┘ └─────────┘ └───────┘ │
│         Metrics Builder             │
└─────────────┬───────────────────────┘
              │ OTLP
              ▼
      [OTLP Exporter]
```

## Metrics

All metrics follow OpenTelemetry semantic conventions where applicable. Custom conventions are documented below.

### Metric Naming Convention

Format: `airflow.<component>.<metric>.<unit>`

Example: `airflow.dag.run.duration` (seconds)

### REST API Metrics

#### DAG Metrics

**`airflow.dag.run.duration`** (Gauge, seconds)
- Description: Duration of a DAG run execution
- Attributes:
  - `dag.id` (string): DAG identifier
  - `dag_run.id` (string): Unique DAG run identifier
  - `run.type` (string): Type of run (scheduled, manual, backfill, dataset_triggered)
  - `state` (string): Final state (success, failed, running)
  - `external_trigger` (boolean): Whether externally triggered

**`airflow.dag.run.count`** (Sum, monotonic)
- Description: Number of DAG runs by state
- Attributes:
  - `dag.id` (string)
  - `state` (string)

**`airflow.dag_runs.by_state`** (Gauge)
- Description: Current count of DAG runs grouped by state
- Attributes:
  - `dag.id` (string)
  - `state` (string): queued, running, success, failed

**`airflow.dags.count`** (Gauge)
- Description: Total number of DAGs by operational status
- Attributes:
  - `status` (string): paused, active

**`airflow.dag.info`** (Gauge, value=1)
- Description: DAG metadata and configuration
- Attributes:
  - `dag.id` (string)
  - `is_paused` (boolean)
  - `tags` (string): Comma-separated list of tags

#### Task Metrics

**`airflow.task.instance.duration`** (Gauge, seconds)
- Description: Task instance execution duration
- Attributes:
  - `dag.id` (string)
  - `task.id` (string)
  - `dag_run.id` (string)
  - `state` (string)
  - `operator` (string): Task operator type (BashOperator, PythonOperator, etc)
  - `pool` (string): Execution pool name
  - `queue` (string): Celery queue name
  - `try_number` (int): Execution attempt number

**`airflow.task_instances.by_state`** (Gauge)
- Description: Current count of task instances by state
- Attributes:
  - `dag.id` (string)
  - `state` (string)

#### Pool Metrics

**`airflow.pool.slots.open`** (Gauge)
- Description: Available slots in the pool
- Attributes:
  - `pool.name` (string)

**`airflow.pool.slots.used`** (Gauge)
- Description: Currently occupied slots
- Attributes:
  - `pool.name` (string)

**`airflow.pool.slots.queued`** (Gauge)
- Description: Tasks queued waiting for slots
- Attributes:
  - `pool.name` (string)

**`airflow.pool.slots.running`** (Gauge)
- Description: Tasks currently executing
- Attributes:
  - `pool.name` (string)

**`airflow.pool.slots.total`** (Gauge)
- Description: Total pool capacity
- Attributes:
  - `pool.name` (string)
  - `pool.description` (string, optional)

**`airflow.pool.slots.deferred`** (Gauge)
- Description: Deferred tasks (async)
- Attributes:
  - `pool.name` (string)

**`airflow.pool.slots.scheduled`** (Gauge)
- Description: Scheduled but not yet running tasks
- Attributes:
  - `pool.name` (string)

#### System Health Metrics

**`airflow.scheduler.health`** (Gauge)
- Description: Scheduler health status (1=healthy, 0=unhealthy)
- Attributes:
  - `status` (string): healthy, unhealthy

**`airflow.database.health`** (Gauge)
- Description: Metadata database health (1=healthy, 0=unhealthy)
- Attributes:
  - `status` (string): healthy, unhealthy

**`airflow.scheduler.heartbeat.age`** (Gauge, seconds)
- Description: Time since last scheduler heartbeat
- No attributes

#### Configuration Metrics

**`airflow.connections.count`** (Gauge)
- Description: Number of configured connections by type
- Attributes:
  - `connection.type` (string): postgres, http, aws, gcp, etc.

**`airflow.variables.count`** (Gauge)
- Description: Total number of Airflow variables
- No attributes

**`airflow.import_errors.count`** (Gauge)
- Description: Number of DAG import errors
- No attributes

### Database Metrics

These metrics require direct database access and provide aggregated statistics.

**`airflow.task.instance.count.db`** (Gauge)
- Description: Task instance count from 24-hour database aggregation
- Attributes:
  - `dag.id` (string)
  - `task.id` (string)
  - `state` (string)
  - `operator` (string)
  - `pool` (string)

**`airflow.task.instance.duration.avg`** (Gauge, seconds)
- Description: Average task duration over 24 hours
- Attributes:
  - `dag.id` (string)
  - `task.id` (string)
  - `state` (string)

**`airflow.task.instance.duration.max`** (Gauge, seconds)
- Description: Maximum task duration over 24 hours
- Attributes:
  - `dag.id` (string)
  - `task.id` (string)
  - `state` (string)

**`airflow.dag.run.count.db`** (Gauge)
- Description: DAG run count from 24-hour database aggregation
- Attributes:
  - `dag.id` (string)
  - `state` (string)

**`airflow.dag.run.duration.avg`** (Gauge, seconds)
- Description: Average DAG run duration over 24 hours
- Attributes:
  - `dag.id` (string)
  - `state` (string)

**`airflow.scheduler.tasks.scheduled`** (Gauge)
- Description: Count of tasks in scheduled state
- No attributes

**`airflow.scheduler.tasks.queued`** (Gauge)
- Description: Count of tasks in queued state
- No attributes

**`airflow.scheduler.tasks.running`** (Gauge)
- Description: Count of tasks in running state
- No attributes

**`airflow.scheduler.tasks.success.24h`** (Gauge)
- Description: Successful tasks in last 24 hours
- No attributes

**`airflow.scheduler.tasks.failed.24h`** (Gauge)
- Description: Failed tasks in last 24 hours
- No attributes

**`airflow.scheduler.tasks.orphaned`** (Gauge)
- Description: Tasks running for more than 1 hour
- No attributes

**`airflow.sla.miss.count`** (Gauge)
- Description: SLA misses in last 24 hours
- Attributes:
  - `dag.id` (string)

### StatsD Metrics

StatsD metrics are received from Airflow's built-in StatsD emitter. Metric names and attributes depend on Airflow's configuration. Common patterns:

- `airflow.dag_processing.*` - DAG parsing metrics
- `airflow.executor.*` - Executor queue metrics
- `airflow.scheduler.*` - Scheduler loop metrics
- `airflow.task.*` - Task-level metrics

## Configuration

### Basic Configuration
```yaml
receivers:
  airflow:
    collection_interval: 30s
    initial_delay: 5s
    
    collection_modes:
      rest_api: true
      database: false
      statsd: false
```

### REST API Configuration
```yaml
rest_api:
  endpoint: http://airflow-webserver:8080
  username: metrics_user
  password: ${env:AIRFLOW_PASSWORD}
  collection_interval: 30s
  include_past_runs: false
  past_runs_lookback: 24h
  timeout: 30s
```

**Parameters:**
- `endpoint` (string, required): Airflow webserver URL
- `username` (string, required): Basic auth username
- `password` (string, required): Basic auth password
- `collection_interval` (duration, default: 30s): Scrape frequency
- `include_past_runs` (boolean, default: false): Collect historical DAG runs
- `past_runs_lookback` (duration, default: 24h): Historical range when include_past_runs=true
- `timeout` (duration, default: 30s): HTTP request timeout

### Database Configuration
```yaml
database:
  host: postgres.example.com
  port: 5432
  database: airflow
  username: metrics_readonly
  password: ${env:DB_PASSWORD}
  ssl_mode: require
  collection_interval: 60s
  query_timeout: 15s
```

**Parameters:**
- `host` (string, required): PostgreSQL host
- `port` (int, default: 5432): PostgreSQL port
- `database` (string, required): Database name
- `username` (string, required): Database user
- `password` (string, required): Database password
- `ssl_mode` (string, default: disable): SSL mode (disable, require, verify-ca, verify-full)
- `collection_interval` (duration, default: 60s): Query frequency
- `query_timeout` (duration, default: 15s): SQL query timeout

**Database Permissions Required:**
```sql
GRANT CONNECT ON DATABASE airflow TO metrics_readonly;
GRANT USAGE ON SCHEMA public TO metrics_readonly;
GRANT SELECT ON task_instance TO metrics_readonly;
GRANT SELECT ON dag_run TO metrics_readonly;
GRANT SELECT ON sla_miss TO metrics_readonly;
GRANT SELECT ON job TO metrics_readonly;
```

### StatsD Configuration
```yaml
statsd:
  endpoint: 0.0.0.0:8125
  transport: udp
  aggregation_interval: 60s
```

**Parameters:**
- `endpoint` (string, default: 0.0.0.0:8125): UDP listen address
- `aggregation_interval` (duration, default: 60s): Metric aggregation window

**Airflow Configuration:**
```ini
# airflow.cfg
[metrics]
statsd_on = True
statsd_host = <collector-ip>
statsd_port = 8125
statsd_prefix = airflow
```

## Deployment

### Prerequisites

- Go 1.21+
- OpenTelemetry Collector Builder (OCB) v0.135.0+
- Airflow 2.0+ (for testing)

### Build
```bash
ocb --config builder-config-airflow.yaml
```

Output: `./otelcol-airflow/otelcol-airflow` (29MB binary)

### Docker
```dockerfile
FROM alpine:latest
COPY otelcol-airflow /otelcol-airflow
COPY config.yaml /etc/otel/config.yaml
ENTRYPOINT ["/otelcol-airflow"]
CMD ["--config=/etc/otel/config.yaml"]
```

### Kubernetes
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: airflow-otel-config
data:
  config.yaml: |
    receivers:
      airflow:
        collection_modes:
          rest_api: true
        rest_api:
          endpoint: http://airflow-webserver.airflow:8080
          username: ${AIRFLOW_USER}
          password: ${AIRFLOW_PASSWORD}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: airflow-otel-collector
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: collector
        image: your-registry/otelcol-airflow:latest
        volumeMounts:
        - name: config
          mountPath: /etc/otel
        envFrom:
        - secretRef:
            name: airflow-credentials
      volumes:
      - name: config
        configMap:
          name: airflow-otel-config
```

## Resource Requirements

**Memory:**
- REST API mode: ~50MB
- Database mode: ~50MB
- StatsD mode: ~30MB + aggregation buffer
- Combined: ~100MB

**CPU:**
- Idle: <1%
- Active scraping: 2-5%

**Network (REST API mode):**
- Per scrape cycle: ~100KB
- 30s interval: ~200KB/min

## Performance Considerations

### Collection Intervals

- **REST API:** 30-60s recommended (avoid API rate limits)
- **Database:** 30-60s recommended (balance freshness vs DB load)
- **StatsD:** Real-time (no polling interval)

### Cardinality

High-cardinality dimensions (`dag_run.id`, `task.id`) can produce significant metric volume:

- 100 DAGs × 10 runs/day × 20 tasks = 20,000 unique timeseries
- With attributes: ~100,000+ timeseries possible

**Recommendation:** Use OTLP backends designed for high-cardinality data (ClickHouse, Honeycomb, Prometheus with appropriate retention).

### Database Load

Direct database queries add load to the Airflow metadata database. Recommendations:

- Use read replicas when possible
- Set appropriate `query_timeout` values
- Monitor query performance
- Consider disabling database mode in high-load environments

## Troubleshooting

### No Metrics Appearing

1. Check collector logs:
```bash
   ./otelcol-airflow --config config.yaml 2>&1 | grep airflow
```

2. Verify Airflow API access:
```bash
   curl -u user:pass http://airflow:8080/api/v1/health
```

3. Test database connectivity:
```bash
   psql -h postgres -U airflow -d airflow -c "SELECT COUNT(*) FROM dag;"
```

### High Memory Usage

- Reduce `collection_interval` frequency
- Limit `past_runs_lookback` duration
- Disable unused collection modes
- For StatsD: reduce `aggregation_interval`

### Missing Dimensions

Some dimensions may be empty or null in Airflow's data model. The receiver skips metrics with critical empty dimensions (like `dag_run_id`) to avoid incomplete data.

## Development

### Project Structure
```
airflowreceiver/
├── config.go                    # Configuration structs
├── factory.go                   # Receiver factory (multi-scraper support)
├── doc.go                       # Package documentation
├── metadata.yaml                # Metric definitions
├── internal/scraper/
│   ├── rest_scraper.go          # REST API client
│   ├── rest_scraper_enhanced.go # Comprehensive metric collection
│   ├── db_scraper.go            # PostgreSQL queries
│   ├── statsd_scraper.go        # UDP listener + aggregation
│   ├── api_types.go             # API response structures
│   └── metrics_builder.go       # Metric construction
└── builder-config-airflow.yaml  # OCB configuration
```

### Adding New Metrics

1. Define metric in `metadata.yaml`
2. Add method to `metrics_builder.go`
3. Call from appropriate scraper
4. Update this README with metric documentation

### Testing
```bash
# Unit tests (TODO)
go test ./...

# Integration test
./otelcol-airflow --config test-config.yaml
```

## Implementation Notes

### Design Decisions

1. **Multiple scrapers:** Independent implementation allows flexible deployment
2. **No caching:** Fresh data on every scrape to avoid stale metrics
3. **Fail-open:** Individual scraper failures don't stop the collector
4. **High-cardinality first:** Preserve all dimensions, let backend handle aggregation

### Known Limitations

1. **No log collection:** Metrics only (logs planned for future)
2. **No trace collection:** Would require OpenLineage integration
3. **PostgreSQL only:** Database scraper assumes PostgreSQL metadata backend
4. **Basic auth only:** REST API uses HTTP basic authentication

### Future Enhancements

- Log collection from task instances
- Trace collection via OpenLineage
- MySQL metadata database support
- OAuth2/JWT authentication for REST API
- Custom XCom metric extraction
- Kubernetes executor metrics
- Plugin system for extensibility

## References

- [Airflow REST API Documentation](https://airflow.apache.org/docs/apache-airflow/stable/stable-rest-api-ref.html)
- [Airflow Metrics Documentation](https://airflow.apache.org/docs/apache-airflow/stable/administration-and-deployment/logging-monitoring/metrics.html)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)

## License

Apache License 2.0 - See LICENSE file

## Contributing

Contributions welcome. Please ensure:
- Code follows Go conventions
- New metrics documented in this README
- Configuration changes include examples
- Semantic versioning for releases
