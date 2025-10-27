# Apache Airflow Receiver

The Airflow receiver collects metrics, logs, and traces from Apache Airflow deployments.

**Status: Alpha**

## Supported Pipeline Types

| Pipeline Type | Stability |
|--------------|-----------|
| metrics      | [alpha]   |
| logs         | [development] |
| traces       | [development] |

## Overview

This receiver supports three collection modes that can be used independently or together:

1. **REST API** - Queries Airflow's REST API for DAG runs, task instances, and pool metrics
2. **Database** - Directly queries the Airflow metadata database for detailed telemetry
3. **StatsD** - Receives StatsD metrics emitted by Airflow (backward compatibility)

### When to Use Each Mode

| Mode | Use Case | Pros | Cons |
|------|----------|------|------|
| REST API | Modern Airflow (2.0+) deployments | No DB access needed, rich context | API rate limits, polling delay |
| Database | Direct DB access available | Real-time data, no API overhead | Requires read-only DB access |
| StatsD | Legacy compatibility, real-time metrics | Low latency, push-based | Pre-aggregated, limited context |

## Configuration

### REST API Mode (Recommended)
```yaml
receivers:
  airflow:
    collection_modes:
      rest_api: true
    rest_api:
      endpoint: http://airflow-webserver:8080
      username: admin
      password: ${env:AIRFLOW_PASSWORD}
      collection_interval: 30s
      include_past_runs: false
      past_runs_lookback: 24h
```

### Database Mode
```yaml
receivers:
  airflow:
    collection_modes:
      database: true
    database:
      endpoint: postgres:5432
      transport: tcp
      database: airflow
      username: metrics_readonly
      password: ${env:DB_PASSWORD}
      collection_interval: 30s
      query_timeout: 15s
      tls:
        insecure: false
```

### StatsD Mode
```yaml
receivers:
  airflow:
    collection_modes:
      statsd: true
    statsd:
      endpoint: 0.0.0.0:8125
      transport: udp
      aggregation_interval: 60s
```

### Multi-Mode (Enterprise)
```yaml
receivers:
  airflow:
    collection_modes:
      rest_api: true
      database: true
      statsd: true
    rest_api:
      endpoint: http://airflow:8080
      username: admin
      password: ${env:AIRFLOW_PASSWORD}
      collection_interval: 60s
    database:
      endpoint: postgres:5432
      database: airflow
      username: readonly
      password: ${env:DB_PASSWORD}
      collection_interval: 30s
    statsd:
      endpoint: 0.0.0.0:8125
      aggregation_interval: 60s
```

## Metrics

| Metric | Description | Attributes | Mode |
|--------|-------------|------------|------|
| `airflow.dag.run.duration` | DAG run execution time | dag.id, run.id, run.type, state | REST/DB |
| `airflow.dag.run.count` | Number of DAG runs by state | dag.id, state | REST/DB |
| `airflow.task.instance.duration` | Task execution time | dag.id, task.id, run.id, state | REST/DB |
| `airflow.scheduler.heartbeat` | Scheduler heartbeat timestamp | - | REST/DB/StatsD |
| `airflow.pool.slots.open` | Available pool slots | pool.name | REST/DB |
| `airflow.pool.slots.used` | Used pool slots | pool.name | REST/DB |
| `airflow.executor.queued.tasks` | Tasks queued in executor | executor.type | StatsD |

## Resource Attributes

| Attribute | Description |
|-----------|-------------|
| `airflow.deployment.environment` | Deployment environment (prod, staging) |
| `airflow.version` | Airflow version |
| `airflow.component` | Component type (scheduler, webserver, worker) |

## Airflow Configuration

### REST API Setup

Ensure Airflow REST API is enabled and accessible:
```python
# airflow.cfg
[api]
auth_backends = airflow.api.auth.backend.basic_auth
```

Create a read-only user for metrics collection:
```bash
airflow users create \
  --username metrics \
  --password <password> \
  --role Viewer \
  --email metrics@example.com \
  --firstname Metrics \
  --lastname User
```

### Database Setup

Create a read-only database user:
```sql
-- PostgreSQL
CREATE USER metrics_readonly WITH PASSWORD 'password';
GRANT CONNECT ON DATABASE airflow TO metrics_readonly;
GRANT USAGE ON SCHEMA public TO metrics_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO metrics_readonly;
```

### StatsD Setup

Enable StatsD in Airflow:
```ini
# airflow.cfg
[metrics]
statsd_on = True
statsd_host = localhost
statsd_port = 8125
statsd_prefix = airflow
```

## Example Deployment

### Docker Compose
```yaml
version: '3'
services:
  otel-collector:
    image: otelcontribcol:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "8125:8125/udp"  # StatsD
    environment:
      - AIRFLOW_PASSWORD=admin
      - AIRFLOW_DB_PASSWORD=airflow
```

### Kubernetes
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
data:
  config.yaml: |
    receivers:
      airflow:
        collection_modes:
          rest_api: true
        rest_api:
          endpoint: http://airflow-webserver.airflow.svc.cluster.local:8080
          username: admin
          password: ${AIRFLOW_PASSWORD}
---
apiVersion: v1
kind: Secret
metadata:
  name: airflow-receiver-secrets
type: Opaque
stringData:
  airflow-password: "admin"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: collector
        image: otelcontribcol:latest
        envFrom:
        - secretRef:
            name: airflow-receiver-secrets
```

## Performance Considerations

### Collection Intervals

- **REST API**: 30-60s for production (avoid API rate limits)
- **Database**: 15-30s (faster response, but queries metadata DB)
- **StatsD**: Real-time (no polling)

### Resource Usage

- Memory: ~50MB per receiver instance
- CPU: <5% during collection
- Network: ~100KB per collection cycle (REST API)

### High-Cardinality Data

The receiver captures high-cardinality dimensions (dag.id, task.id, run.id). When using backends like ClickHouse:
```yaml
exporters:
  clickhouse:
    endpoint: tcp://clickhouse:9000
    database: observability
    ttl: 30d  # Retention for high-cardinality data
```

## Troubleshooting

### No Metrics Appearing

1. Check Airflow API accessibility:
```bash
   curl -u admin:password http://airflow:8080/api/v1/health
```

2. Verify receiver logs:
```bash
   docker logs otel-collector | grep airflow
```

3. Check collection mode is enabled:
```yaml
   collection_modes:
     rest_api: true  # Must be true!
```

### Database Connection Failures

- Verify connectivity: `psql -h postgres -U airflow -d airflow`
- Check TLS settings if using SSL
- Ensure user has SELECT permissions

### StatsD Not Receiving Metrics

- Verify Airflow StatSD config points to receiver
- Check firewall/network policies (UDP port 8125)
- Look for "Failed to send statsd metric" in Airflow logs

## Contributing

This receiver is under active development. Contributions welcome!

**Roadmap:**
- [ ] Logs collection from task execution logs
- [ ] Traces collection via OpenLineage integration
- [ ] Connection health metrics
- [ ] SLA miss tracking
- [ ] Custom XCom metric extraction

## License

Apache 2.0
