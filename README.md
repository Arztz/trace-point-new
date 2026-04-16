# Trace-Point v1.0.3

**Resource-to-Code Correlation Engine** — Bridges Kubernetes infrastructure metrics and code-level root cause analysis.

## Architecture

```
Prometheus (Metrics) ─→ Go Backend ─→ SQLite (Storage)
                            ↓
SigNoz/ClickHouse ──→ Correlation Logic
                            ↓
GCP Cloud Profiler ──→ Enrichment
                            ↓
Discord Webhook ────→ Alert
                            ↓
React Dashboard ────→ Visualization
```

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Access to Prometheus, SigNoz/ClickHouse (for live data)
- GCP credentials (`gcloud auth application-default login`)

### Development

```bash
# Install Go dependencies
go mod tidy

# Install frontend dependencies
cd web && npm install && cd ..

# Run backend (port 8088)
make dev-backend

# Run frontend (port 5173, proxies API to backend)
make dev-frontend
```

### Production Build

```bash
make build
# Binary: bin/trace-point
# Frontend: web/dist/
```

### Configuration

Edit `config.yaml`:

```yaml
prometheus:
  url: "http://your-prometheus:9090"

signoz:
  url: "http://your-clickhouse:8123"

gcloud:
  project_id: "your-gcp-project"
  profiler_enabled: true

namespaces:
  - "your-namespace"
```

Environment variables:
- `DISCORD_WEBHOOK_URL` — Discord webhook for alerts
- `GCP_PROJECT_ID` — GCP project for Cloud Profiler
- `TRACE_POINT_CONFIG` — Custom config file path

## Key Features

| Feature | Description |
|---------|-------------|
| **Spike Detection** | Real-time detection using moving average algorithm |
| **Deployment-Level** | All metrics aggregated at deployment level (not pod) |
| **SigNoz Correlation** | ClickHouse trace queries to find culprit routes |
| **GCP Profiler** | Cloud Profiler integration for code-level functions |
| **Discord Alerts** | Rich embeds with Route, Impact, Culprit function |
| **Gravity Scores** | Identify services needing architectural refactoring |
| **Historical Analysis** | Sliding window spike explorer with severity classification |

## Tech Stack

- **Backend:** Go + chi router
- **Frontend:** React 18 + Vite + Tailwind CSS v4 + Recharts
- **Database:** SQLite (WAL mode, 7-day retention)
- **Integrations:** Prometheus, SigNoz/ClickHouse, GCP Cloud Profiler, Discord
# trace-point-new
