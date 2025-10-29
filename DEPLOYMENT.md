# Deployment Guide

## Quick Deployment Options

### 1. AWS MWAA (Managed Airflow)

**Prerequisites:**
- AWS MWAA environment running
- Network access to MWAA API endpoint
- Valid MWAA credentials

**Steps:**
```bash
# 1. Set environment variables
export MWAA_USERNAME="your-username"
export MWAA_PASSWORD="your-password"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://your-backend.com"
export OTEL_API_KEY="your-api-key"

# 2. Update endpoint in config
sed -i 's/YOUR-ENVIRONMENT-ID/your-actual-id/g' examples/config-aws-mwaa.yaml

# 3. Run collector
./otelcol-airflow/otelcol-airflow --config examples/config-aws-mwaa.yaml
```

**Expected Results:**
- 70+ REST API metrics collected every 60s
- No database/logs (not available in MWAA)

---

### 2. GCP Composer (Managed Airflow)

**Prerequisites:**
- GCP Composer environment running
- Network access to Composer endpoint
- Valid Composer credentials

**Steps:**
```bash
# 1. Set environment variables
export COMPOSER_USERNAME="your-username"
export COMPOSER_PASSWORD="your-password"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://your-backend.com"
export OTEL_API_KEY="your-api-key"

# 2. Update endpoint in config
sed -i 's/YOUR-COMPOSER-ENVIRONMENT/your-actual-env/g' examples/config-gcp-composer.yaml

# 3. Run collector
./otelcol-airflow/otelcol-airflow --config examples/config-gcp-composer.yaml
```

---

### 3. Kubernetes (Self-Hosted)

**Prerequisites:**
- Airflow deployed on Kubernetes (Helm chart recommended)
- PostgreSQL accessible
- Network policies allow collector → Airflow + PostgreSQL

**Option A: Sidecar Container (Recommended)**

Deploy collector as sidecar in Airflow namespace:
```yaml
# Add to your Airflow deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: airflow-otel-collector
  namespace: airflow
spec:
  replicas: 1
  selector:
    matchLabels:
      app: airflow-otel
  template:
    metadata:
      labels:
        app: airflow-otel
    spec:
      containers:
      - name: otelcol
        image: your-registry/otelcol-airflow:latest
        args: ["--config=/etc/otel/config.yaml"]
        env:
        - name: AIRFLOW_USERNAME
          valueFrom:
            secretKeyRef:
              name: airflow-secrets
              key: username
        - name: AIRFLOW_PASSWORD
          valueFrom:
            secretKeyRef:
              name: airflow-secrets
              key: password
        - name: DB_USERNAME
          valueFrom:
            secretKeyRef:
              name: postgres-secrets
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secrets
              key: password
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "https://your-backend.com"
        - name: OTEL_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: otel-secrets
              key: token
        - name: K8S_CLUSTER_NAME
          value: "production-cluster"
        volumeMounts:
        - name: config
          mountPath: /etc/otel
      volumes:
      - name: config
        configMap:
          name: otel-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-config
  namespace: airflow
data:
  config.yaml: |
    # Paste contents of examples/config-kubernetes.yaml here
```

**Option B: Standalone Deployment**
```bash
# 1. Create namespace
kubectl create namespace observability

# 2. Create secrets
kubectl create secret generic airflow-creds \
  --from-literal=username=$AIRFLOW_USERNAME \
  --from-literal=password=$AIRFLOW_PASSWORD \
  -n observability

kubectl create secret generic postgres-creds \
  --from-literal=username=$DB_USERNAME \
  --from-literal=password=$DB_PASSWORD \
  -n observability

# 3. Apply collector deployment
kubectl apply -f k8s/otel-collector-deployment.yaml
```

---

### 4. Docker Compose (Local Testing)

**Prerequisites:**
- Docker and Docker Compose installed
- Airflow running in Docker

**Steps:**
```bash
# 1. Start Airflow (if not running)
cd examples
docker-compose up -d

# 2. Wait for Airflow to be ready
sleep 30

# 3. Run collector
cd ..
./otelcol-airflow/otelcol-airflow --config examples/config-docker.yaml
```

**Expected Results:**
- REST API metrics (70+)
- Database metrics (15+)
- Event logs (continuous stream)
- Health metrics (14 metrics)

---

## Verification

### Check Collector Health
```bash
# Look for these log messages:
✓ "Starting REST API scraper"
✓ "Connected to Airflow database"
✓ "Log scraper database connection established"
✓ "Scrape succeeded"
```

### View Metrics
```bash
# If using debug exporter:
# Watch the terminal output for:
- airflow.scheduler.health
- airflow.dag.info
- airflow.scraper.health
```

### Common Issues

**401 Authentication Errors on REST API:**
- Expected if Airflow session expired
- Health and database metrics still work
- Check credentials and endpoint

**Connection Refused (Database):**
- Verify PostgreSQL host/port
- Check network policies
- Ensure pg_hba.conf allows connection
- REST API metrics still work

**High Memory Usage:**
- Increase `collection_interval` to 60s or 120s
- Disable unused collection modes
- Reduce `past_runs_lookback` duration

---

## Production Recommendations

### Resource Limits
```yaml
# Kubernetes resource requests/limits
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Collection Intervals
- **SaaS (MWAA/Composer):** 60s (avoid rate limits)
- **On-prem low load:** 30s
- **On-prem high load:** 60s or 120s

### High Availability
- Run 2+ collector replicas
- Use StatefulSet for stable network identity
- Monitor collector health metrics

### Security
- Store credentials in secrets management (Vault, AWS Secrets Manager)
- Use IAM roles where possible (AWS MWAA)
- Enable TLS for database connections (`ssl_mode: require`)
- Rotate credentials regularly

---

## Monitoring the Collector

### Key Health Metrics to Alert On
```promql
# Scraper marked unhealthy
airflow_scraper_health{scraper_type="rest_api"} == 0

# High failure rate
rate(airflow_scraper_scrapes_failed[5m]) > 0.5

# Consecutive errors climbing
airflow_scraper_errors_consecutive > 3

# Slow scrapes
airflow_scraper_duration_avg > 30
```

### Dashboard Panels
1. Scraper health status (gauge)
2. Success vs failure rate (time series)
3. Scrape duration trends (time series)
4. Consecutive errors (gauge)

---

## Scaling Considerations

### Single Collector Capacity
- **REST API only:** 100+ Airflow DAGs
- **With database:** 50+ DAGs (depends on query complexity)
- **With logs:** 1000+ events/minute

### When to Scale Out
- Scrape duration > 20s consistently
- Memory usage > 400MB
- CPU usage > 50%

### Multi-Collector Setup
If monitoring multiple Airflow instances, deploy one collector per instance rather than one collector scraping all instances.
```yaml
# Collector 1 → Airflow Production
# Collector 2 → Airflow Staging
# Collector 3 → Airflow Dev
```
