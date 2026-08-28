# Monitoring

IdpForge exposes Prometheus metrics at `/metrics` (no auth; put it behind
your network perimeter, not the public internet).

## Prometheus

Use [deploy/monitoring/prometheus.yml](../deploy/monitoring/prometheus.yml)
as a starting scrape config; change the target to your instance's
host:port.

## Grafana

Import [deploy/monitoring/grafana-dashboard.json](../deploy/monitoring/grafana-dashboard.json)
directly (Dashboards -> New -> Import), pointing it at your Prometheus data
source. Panels: HTTP request rate by status, login attempts by outcome,
p95 request latency by route, rate limit rejections by route, audit queue
depth, goroutines, and memory in use.

## Built-in usage graphs

The admin console's **Usage** page (`/usage`) doesn't need Prometheus at
all: IdpForge samples its own cumulative counters into a
`metrics_snapshots` table every 10 minutes and plots deltas over the last
7/30/90 days directly from the database. This is the quicker option if you
don't already run a Prometheus + Grafana stack; use the Prometheus/Grafana
setup above for longer retention, alerting, or cross-service dashboards.
