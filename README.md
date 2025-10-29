# Apache Airflow OpenTelemetry Receiver

A production-grade OpenTelemetry receiver for comprehensive Apache Airflow observability. Collects metrics, logs, and performance data from both SaaS and self-hosted Airflow deployments.

## 🎯 Features

### Multi-Source Data Collection
- **REST API Metrics** - Works with any Airflow deployment (SaaS + On-prem)
  - DAG states, run durations, task metrics
  - Pool utilization and scheduler health
  - Connection and variable counts
  - Import errors and SLA misses
  - 70+ high-cardinality metrics

- **Database Analytics** - Deep insights from Airflow metadata DB (On-prem only)
  - Task instance statistics by operator, pool, queue
  - DAG run aggregations and trends
  - Scheduler performance metrics
  - Historical analysis over 24h windows

- **Event Logs** - Structured OpenTelemetry logs (On-prem only)
  - Real-time Airflow events from database
  - Auto-severity mapping (INFO/WARN/ERROR)
  - Rich attributes (event type, owner, host, command)

- **Self-Monitoring** - Built-in health metrics
  - Scraper success/failure rates
  - Response time tracking (last/avg/max)
  - Automatic health status (marks unhealthy after 3 failures)
  - 7 health metrics per scraper

### Production-Ready Features
- ✅ **Exponential backoff retry** - 3 attempts with 1s→2s→4s→10s backoff
- ✅ **Connection pooling** - Efficient database connection management
- ✅ **Graceful degradation** - Individual failures don't crash receiver
- ✅ **Rate limiting ready** - Configurable collection intervals
- ✅ **High cardinality preservation** - Full dimensional data
- ✅ **StatsD support** - UDP metrics listener (experimental)

## 📊 Architecture
```
┌─────────────────────────────────────────────────────────┐
│                  OTel Airflow Receiver                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  REST API    │  │  Database    │  │  Event Logs  │ │
│  │  Scraper     │  │  Scraper     │  │  Receiver    │ │
│  │              │  │              │  │              │ │
│  │ • DAGs       │  │ • Task stats │  │ • CLI events │ │
│  │ • Runs       │  │ • DAG runs   │  │ • Scheduler  │ │
│  │ • Tasks      │  │ • Scheduler  │  │ • Webserver  │ │
│  │ • Pools      │  │ • SLA misses │  │ • User ops   │ │
│  │ • Health     │  │              │  │              │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│         │                  │                  │        │
│         └──────────────────┴──────────────────┘        │
│                           │                            │
│                   ┌───────▼────────┐                   │
│                   │ Health Metrics │                   │
│                   │ Self-Monitor   │                   │
│                   └────────────────┘                   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │  OpenTelemetry Pipeline │
              │  (Processors/Exporters) │
              └─────────────────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │   Backend (Dynatrace,   │
              │   Prometheus, etc)      │
              └─────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- OpenTelemetry Collector Builder (`ocb`)
- Access to Airflow REST API
- (Optional) PostgreSQL access for database/logs features

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/npcomplete777/airflowreceiver.git
cd airflowreceiver
```

2. **Build the collector:**
```bash
ocb --config builder-config-airflow.yaml
```

3. **Configure for your environment** (see examples below)

4. **Run:**
```bash
./otelcol-airflow/otelcol-airflow --config config.yaml
```

## 📝 Configuration

### Minimal Config (SaaS - REST API Only)

Perfect for AWS MWAA, GCP Composer, Astronomer:
```yaml
receivers:
  airflow:
    collection_interval: 60s
    
    collection_modes:
      rest_api: true
      database: false
      logs: false
    
    rest_api:
      endpoint: https://your-airflow.cloud
      username: your-username
      password: your-password
      collection_interval: 60s

exporters:
  otlphttp:
    endpoint: https://your-backend.com/v1/metrics

service:
  pipelines:
    metrics:
      receivers: [airflow]
      exporters: [otlphttp]
```

### Full Config (On-Prem - All Features)
```yaml
receivers:
  airflow:
    collection_interval: 30s
    
    collection_modes:
      rest_api: true
      database: true
      logs: true
    
    rest_api:
      endpoint: http://airflow-webserver:8080
      username: admin
      password: ${AIRFLOW_PASSWORD}
      collection_interval: 30s
      include_past_runs: true
      past_runs_lookback: 24h
    
    database:
      host: postgres
      port: 5432
      database: airflow
      username: airflow
      password: ${DB_PASSWORD}
      ssl_mode: disable
      collection_interval: 30s
    
    logs:
      host: postgres
      port: 5432
      database: airflow
      username: airflow
      password: ${DB_PASSWORD}
      ssl_mode: disable
      collection_interval: 30s

exporters:
  otlphttp:
    endpoint: https://your-backend.com

service:
  pipelines:
    metrics:
      receivers: [airflow]
      exporters: [otlphttp]
    logs:
      receivers: [airflow]
      exporters: [otlphttp]
```

## 🌐 Deployment Examples

See the `/examples` directory for complete configurations:
- `config-aws-mwaa.yaml` - AWS Managed Workflows for Apache Airflow
- `config-gcp-composer.yaml` - Google Cloud Composer
- `config-astronomer.yaml` - Astronomer Cloud
- `config-kubernetes.yaml` - Self-hosted on Kubernetes
- `config-docker.yaml` - Docker Compose deployment

## 📈 Metrics Reference

### REST API Metrics
- `airflow.scheduler.health` - Scheduler health status (1=healthy, 0=unhealthy)
- `airflow.database.health` - Database health status
- `airflow.scheduler.heartbeat.age` - Age of last scheduler heartbeat (seconds)
- `airflow.dag.info` - DAG information with tags (per DAG)
- `airflow.dags.count` - Total DAGs by status (paused/active)
- `airflow.dag.run.duration` - DAG run execution time with dimensions
- `airflow.dag_runs.by_state` - DAG run counts by state
- `airflow.pool.slots.*` - Pool utilization (open/used/queued/running/total)
- `airflow.variables.count` - Total Airflow variables
- `airflow.import_errors.count` - Number of DAG import errors

### Database Metrics  
- `airflow.scheduler.tasks.*` - Task counts (scheduled/queued/running/success/failed/orphaned)
- `airflow.task.instance.count` - Task instance counts by DAG/task/state/operator/pool
- `airflow.task.instance.duration.*` - Task duration statistics (avg/max)
- `airflow.dag.run.count` - DAG run counts from database
- `airflow.dag.run.duration.*` - DAG run duration from database
- `airflow.sla.miss.count` - SLA misses by DAG

### Health Metrics (Per Scraper)
- `airflow.scraper.scrapes.total` - Total scrape attempts
- `airflow.scraper.scrapes.successful` - Successful scrapes
- `airflow.scraper.scrapes.failed` - Failed scrapes  
- `airflow.scraper.health` - Scraper health (1=healthy, 0=unhealthy)
- `airflow.scraper.duration.last` - Last scrape duration
- `airflow.scraper.duration.avg` - Average scrape duration
- `airflow.scraper.errors.consecutive` - Consecutive error count

### Event Logs
Structured OpenTelemetry logs with attributes:
- `airflow.log.source` - Always "database"
- `airflow.event` - Event type (cli_scheduler, dag_run, task_instance, etc)
- `owner` - Airflow user/system
- `extra.host_name` - Host that generated event
- `extra.full_command` - Full command executed

## 🔧 Advanced Configuration

### Retry Logic
```yaml
# Built-in exponential backoff (not configurable yet)
# - Max attempts: 3
# - Backoff: 1s → 2s → 4s → 10s max
# - Skips retry on 401/403
```

### Rate Limiting
```yaml
receivers:
  airflow:
    # Adjust intervals to control API load
    collection_interval: 60s  # Global default
    
    rest_api:
      collection_interval: 30s  # Per-scraper override
```

### Connection Pooling
```yaml
# Automatic database connection pooling:
# - Max open connections: 10
# - Max idle connections: 5
# - Connection max lifetime: 5 minutes
# - Connection max idle time: 1 minute
```

## 🐛 Troubleshooting

### 401 Authentication Errors
**Problem:** REST API returns 401 even with correct credentials  
**Solution:** 
- Verify Airflow version supports basic auth (2.0+)
- Check credentials are correct
- For AWS MWAA: Use IAM authentication (see examples)
- Health/database metrics will still work

### Database Connection Refused
**Problem:** Cannot connect to PostgreSQL  
**Solution:**
- Verify PostgreSQL is accessible from collector
- Check `pg_hba.conf` allows connection from collector IP
- Ensure credentials are correct
- REST API metrics will still work

### High Memory Usage
**Problem:** Collector using excessive memory  
**Solution:**
- Increase `collection_interval` to reduce scrape frequency
- Disable unused collection modes
- Limit `past_runs_lookback` duration

### Missing Metrics
**Problem:** Some metrics not appearing  
**Solution:**
- Check scraper health metrics for failures
- Verify Airflow version compatibility (2.0+ recommended)
- Enable debug logging: `service.telemetry.logs.level: debug`
- Review consecutive error counts

## 🤝 Contributing

Contributions welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup
```bash
# Clone and build
git clone https://github.com/npcomplete777/airflowreceiver.git
cd airflowreceiver
ocb --config builder-config-airflow.yaml

# Run tests (when available)
go test ./...

# Run against local Airflow
docker-compose -f examples/docker-compose.yaml up
./otelcol-airflow/otelcol-airflow --config examples/config-docker.yaml
```

## 📜 License

Apache License 2.0

## 🙏 Acknowledgments

Built with the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/).

## 📞 Support

- Issues: https://github.com/npcomplete777/airflowreceiver/issues
- Discussions: https://github.com/npcomplete777/airflowreceiver/discussions

---

**Status:** Production-ready for SaaS deployments (REST API). Database and logs features tested on PostgreSQL-backed Airflow 2.7+. StatsD support experimental.
