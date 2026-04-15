# Trace-Point: Resource-to-Code Correlation Engine — Implementation Plan

## Goal

Build the **Trace-Point** application from scratch — a Resource-to-Code Correlation Engine that bridges Kubernetes infrastructure metrics and code-level root cause analysis. The application correlates **Prometheus** metrics with **SigNoz/ClickHouse** trace data and **GCP Cloud Profiler** profiling data to automatically identify culprit functions during resource spikes.

---

## User Review Required

> [!IMPORTANT]
> **Key Corrections from User Feedback (must be applied throughout):**
> 1. **Deployment-level only** — All metrics, queries, timeline, and dashboards operate at the **deployment/replicaset** level. No pod-level views. The FINAL-REQUIREMENTS.md has pod-level PromQL queries and pod filtering in the UI — these are **removed** in favor of deployment-level aggregation using `label_replace`.
> 2. **GCP Cloud Profiler** — The profiler integration is **Google Cloud Profiler** (not a generic profiler or SigNoz profiler). Uses `cloud.google.com/go/cloudprofiler/apiv2` to fetch profiles via the `ListProfiles` API.
> 3. **SigNoz integration must be functional** — Not a stub; the ClickHouse SQL queries for trace correlation must work against a real SigNoz/ClickHouse backend.

> [!WARNING]
> **Technology Stack (per FINAL-REQUIREMENTS.md):**
> - Backend: **Go** with **chi router**
> - Frontend: **React 18 + Vite + Tailwind CSS + Recharts**
> - Database: **SQLite** (WAL mode, 7-day retention)
> - Config: **config.yaml** (YAML)
> - Auth: Local `gcloud auth application-default login`

---

## Proposed Changes

### Phase 1: Project Scaffolding & Configuration

#### [NEW] Go module & project structure

```
trace-point-renew/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point
├── internal/
│   ├── config/
│   │   └── config.go                  # YAML config loader
│   ├── api/
│   │   ├── router.go                  # Chi router setup & middleware
│   │   ├── handler_health.go          # /health, /metrics, /api/v1
│   │   ├── handler_spikes.go          # /api/v1/spikes, /api/v1/spikes/:id
│   │   ├── handler_timeline.go        # /api/v1/timeline
│   │   ├── handler_analyze.go         # /api/v1/spikes/analyze
│   │   ├── handler_export.go          # /api/v1/export, /api/v1/export/refactoring
│   │   ├── handler_config.go          # /api/v1/config
│   │   └── handler_gravity.go         # /api/v1/gravity-scores
│   ├── domain/
│   │   ├── spike.go                   # SpikeEvent, HistoricalSpike entities
│   │   ├── timeline.go                # TimelineMetric, AvailableDeployment
│   │   ├── gravity.go                 # GravityScore entity
│   │   └── correlation.go             # RouteActivity, CorrelationResult
│   ├── service/
│   │   ├── detector.go                # Spike detection engine (polling loop)
│   │   ├── correlator.go              # Orchestrates Prometheus → SigNoz → GCP Profiler
│   │   ├── alerter.go                 # Discord webhook alerting
│   │   ├── gravity.go                 # Resource Gravity Score calculator
│   │   └── analyzer.go                # Historical spike analysis
│   ├── repository/
│   │   ├── sqlite.go                  # SQLite connection, migrations, WAL
│   │   ├── spike_repo.go              # Spike CRUD + 7-day purge
│   │   └── metrics_cache_repo.go      # Metrics cache table
│   ├── integration/
│   │   ├── prometheus/
│   │   │   ├── client.go              # Prometheus HTTP client
│   │   │   └── queries.go             # PromQL builders (DEPLOYMENT-LEVEL ONLY)
│   │   ├── signoz/
│   │   │   ├── client.go              # ClickHouse HTTP client
│   │   │   └── queries.go             # Trace SQL builders
│   │   ├── profiler/
│   │   │   └── client.go              # GCP Cloud Profiler client (apiv2)
│   │   └── discord/
│   │       └── client.go              # Discord webhook client
│   └── middleware/
│       ├── cors.go                    # CORS middleware
│       └── logging.go                 # Request logging
├── web/                               # React frontend (Vite project)
│   ├── src/
│   │   ├── App.jsx
│   │   ├── main.jsx
│   │   ├── index.css                  # Tailwind + custom CSS
│   │   ├── pages/
│   │   │   ├── Dashboard.jsx          # Main timeline view
│   │   │   ├── SpikeList.jsx          # Spike events list
│   │   │   ├── SpikeExplorer.jsx      # Historical analysis
│   │   │   └── GravityScores.jsx      # Refactoring intelligence
│   │   ├── components/
│   │   │   ├── Layout.jsx             # App shell (sidebar + main)
│   │   │   ├── TimelineChart.jsx      # Recharts CPU/RAM timeline
│   │   │   ├── DeploymentSelector.jsx # Deployment filter dropdown
│   │   │   ├── DeploymentLegend.jsx   # Clickable legend
│   │   │   ├── TimeRangeSelector.jsx  # 1h/6h/24h/7d
│   │   │   ├── SpikeAnalysisTable.jsx # Sortable historical spike table
│   │   │   ├── AnalyzeControls.jsx    # Date range + window size
│   │   │   ├── SpikeDetailModal.jsx   # Drill-down: Trace + Profiler
│   │   │   ├── FilterBar.jsx          # Namespace filter
│   │   │   └── GravityTable.jsx       # Gravity score table
│   │   ├── hooks/
│   │   │   ├── useTimeline.js         # Timeline data fetching
│   │   │   ├── useSpikes.js           # Spike data fetching
│   │   │   └── useGravity.js          # Gravity score fetching
│   │   └── utils/
│   │       ├── api.js                 # Axios/fetch wrapper
│   │       └── formatters.js          # Date/number formatting
│   ├── package.json
│   ├── vite.config.js
│   ├── tailwind.config.js
│   └── index.html
├── config.yaml                        # Application configuration
├── Makefile                           # Build, run, dev commands
├── go.mod
├── go.sum
└── README.md
```

#### [NEW] [config.yaml](file:///home/rut/Project/trace-point-renew/config.yaml)

Application configuration file with all tunable parameters:
- `app`: host, port
- `prometheus`: URL, timeout, `use_deployment_aggregation: true` (always true — no pod-level)
- `signoz`: URL, timeout, database name
- `gcloud`: project_id, profiler_enabled
- `detection`: cpu_threshold, memory_threshold, polling_interval, moving_average_window, baseline_learning_period, reconciliation_buffer, cooldown
- `timeline`: classification thresholds
- `discord`: enabled, webhook_url (from env var)
- `database`: type (sqlite), path
- `namespaces`: list of monitored namespaces
- `deploy_exclude_patterns`: regex patterns to exclude deployments

---

### Phase 2: Backend Core — Domain & Config

#### [NEW] internal/config/config.go

- YAML config loading using `gopkg.in/yaml.v3`
- Environment variable expansion for secrets (`${DISCORD_WEBHOOK_URL}`)
- Defaults for all detection parameters
- Validation

#### [NEW] internal/domain/*.go

All domain entities as defined in §4 of FINAL-REQUIREMENTS, **corrected**:
- `SpikeEvent` — uses `DeploymentName` instead of `PodName`
- `TimelineMetric` — `DeploymentName` instead of `PodName`
- `HistoricalSpike` — `DeploymentName` (was `ReplicasetName`)
- `AvailableDeployment` (renamed from `AvailablePod`) — `Name`, `Namespace`, `CurrentCPU`, `CurrentRAM`
- `GravityScore`, `RouteActivity`, `CorrelationResult`, `ContainerMetrics` — unchanged

---

### Phase 3: Backend — Database Layer

#### [NEW] internal/repository/sqlite.go

- SQLite connection with WAL mode
- Schema migrations (version 2)
- Tables: `spike_events`, `config`, `schema_migrations`, `metrics_cache`
- Auto-purge goroutine (delete records > 7 days, runs every hour)

#### [NEW] internal/repository/spike_repo.go

- `CreateSpike(event)` — insert spike event
- `GetSpike(id)` — get by UUID
- `ListSpikes(filters)` — with namespace, deployment, limit, offset, sort, order
- `GetSpikeDetails(id)` — full details with trace & profiler data
- `PurgeOldSpikes()` — delete entries older than 7 days
- `GetSpikesForGravity(days)` — aggregated data for gravity score calculation

---

### Phase 4: Backend — Integration Clients

#### [NEW] internal/integration/prometheus/client.go & queries.go

**All queries use DEPLOYMENT-LEVEL aggregation** via `label_replace`:

```promql
-- CPU (deployment-level, instant)
(sum by (deployment, namespace) (
    label_replace(
        rate(container_cpu_usage_seconds_total{namespace=~"fundii", container!=""}[5m]),
        "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
    )
)) /
(sum by (deployment, namespace) (
    label_replace(
        kube_pod_container_resource_requests{namespace=~"fundii", container!="", resource="cpu"},
        "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
    )
)) * 100
```

Functions:
- `QueryInstantCPU(namespaces)` — current CPU % per deployment
- `QueryInstantRAM(namespaces)` — current RAM % per deployment
- `QueryTimelineMetrics(start, end, step, namespaces)` — range query for timeline
- `QueryDeploymentCPUHistory(deployment, namespace, start, end, step)` — specific deployment
- `QueryDeploymentRAMHistory(deployment, namespace, start, end, step)` — specific deployment
- `GetAvailableDeployments(namespaces)` — list deployments with current metrics

> [!IMPORTANT]
> **No pod-level queries exist.** The `label_replace` regex `"(.*)-[a-z0-9]+-[a-z0-9]+"` strips the ReplicaSet hash and pod hash to aggregate at deployment level.

---

#### [NEW] internal/integration/signoz/client.go & queries.go

**Functional ClickHouse integration** (not a stub):

```sql
SELECT
    traceID, spanName, serviceName,
    toUnixTimestamp(startTime/1000000000) as startTime,
    durationNano as duration,
    JSONExtractString(attributes, 'http.url') as httpUrl,
    JSONExtractString(attributes, 'http.method') as httpMethod,
    JSONExtractInt(attributes, 'http.status_code') as statusCode,
    JSONExtractString(attributes, 'k8s.deployment.name') as deploymentName,
    JSONExtractString(attributes, 'k8s.namespace.name') as namespace
FROM signoz_traces.distributed_signoz_index_v2
WHERE startTime >= {startTs} AND startTime <= {endTs}
  AND JSONExtractString(attributes, 'k8s.namespace.name') = '{namespace}'
  AND JSONExtractString(attributes, 'k8s.deployment.name') = '{deployment}'
ORDER BY startTime DESC
LIMIT {limit}
```

Functions:
- `QueryTraces(namespace, deployment, start, end)` — get traces during spike window
- `CorrelateRoutes(traces)` — aggregate by route, calculate resource weights
- `FindCulpritRoute(activities)` — highest CPU consumer is primary culprit

> [!IMPORTANT]
> **Changed from pod-level filter** (`k8s.pod.name`) **to deployment-level** (`k8s.deployment.name`) to match the deployment-only architecture.

---

#### [NEW] internal/integration/profiler/client.go

**GCP Cloud Profiler integration** using `cloud.google.com/go/cloudprofiler/apiv2`:

```go
import (
    cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
    "cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
)

func (c *Client) FetchProfiles(ctx context.Context, serviceName string, start, end time.Time) ([]*ProfileResult, error) {
    client, err := cloudprofiler.NewProfilerClient(ctx)
    // ...
    req := &cloudprofilerpb.ListProfilesRequest{
        Parent: fmt.Sprintf("projects/%s", c.projectID),
    }
    it := client.ListProfiles(ctx, req)
    // Filter by time range and service name
    // Parse profile data to extract top functions
}
```

Functions:
- `FetchProfiles(serviceName, start, end)` — list profiles for a service in time range
- `ExtractTopFunctions(profile)` — parse profile to find top CPU/memory consumers
- `GetCulpritFunction(serviceName, start, end)` — combined: fetch + extract top function name and file path

> [!IMPORTANT]
> **This is GCP Cloud Profiler** — NOT a generic profiler or SigNoz profiler. Authentication uses Application Default Credentials (`gcloud auth application-default login`). The profiler data has a 5-10 minute lag (reconciliation buffer accounts for this).

---

#### [NEW] internal/integration/discord/client.go

Discord webhook client:
- `SendSpikeAlert(event)` — format as Discord embed with Route, Impact, Culprit fields
- Rate limiting with exponential backoff
- Alert format per Appendix C of requirements

---

### Phase 5: Backend — Business Logic Services

#### [NEW] internal/service/detector.go

The core spike detection engine:
- Runs as a goroutine, polls Prometheus every `polling_interval_seconds` (default 30s)
- Maintains in-memory moving average per deployment (30-minute rolling window)
- **Baseline learning period**: First 5 minutes — collects data silently, no spike detection
- **Spike formula**: `Current Usage > Moving Average + (Threshold%)`
  - Example: MA=50%, Threshold=50% → trigger at >75%
- On spike detection → logs event → waits reconciliation buffer (8 min) → triggers correlation

#### [NEW] internal/service/correlator.go

Orchestrates the full correlation chain:
1. **Spike detected** by detector
2. **Wait** reconciliation buffer (8 minutes) for GCP Profiler data availability
3. **Query SigNoz/ClickHouse** for traces active during spike ±5 minutes
4. **Identify culprit route** (highest CPU consumer)
5. **Fetch GCP Profiler** data for the service during spike period
6. **Extract culprit function** name and file path
7. **Store** complete spike event in SQLite
8. **Trigger** Discord alert (if cooldown allows)

Output chain: `Deployment Spike → Route → Trace ID → Function Name`

#### [NEW] internal/service/alerter.go

Discord alerting service:
- Takes correlated spike event
- Checks cooldown (15 min per deployment)
- Formats Discord embed
- Sends via webhook
- Records `AlertSent` and `CooldownEnd` in DB

#### [NEW] internal/service/gravity.go

Resource Gravity Score calculator:
- Formula: `Score = Resource Peak × (1 / Request Frequency)`
- Scans 7 days of spike data
- Identifies job-like routes via regex (`/tasks/*`, `/batch/*`, `/jobs/*`)
- Auto-tags suspected jobs
- Returns ranked list of services needing refactoring

#### [NEW] internal/service/analyzer.go

Historical spike analysis (Spike Explorer):
- Sliding window algorithm over Prometheus range data
- For each point: calculate moving average from preceding window → check deviation
- Severity: >200% = critical, 100-200% = medium, 50-100% = low
- Supports configurable window sizes: 5m, 15m, 30m, 1h
- Pagination support

---

### Phase 6: Backend — API Handlers

#### [NEW] internal/api/router.go

Chi router with:
- CORS middleware (allow frontend on different port during dev)
- Request logging middleware
- Recovery middleware
- Route grouping under `/api/v1`

#### [NEW] internal/api/handler_*.go

| File | Endpoints | Description |
|------|-----------|-------------|
| `handler_health.go` | `GET /health`, `GET /metrics`, `GET /api/v1` | System endpoints |
| `handler_spikes.go` | `GET /api/v1/spikes`, `GET /api/v1/spikes/:id`, `GET /api/v1/spikes/:id/details` | Spike CRUD |
| `handler_timeline.go` | `GET /api/v1/timeline` | Timeline with CPU/RAM metrics, spike markers, available deployments, summary |
| `handler_analyze.go` | `GET /api/v1/spikes/analyze` | Historical spike analysis with sliding window |
| `handler_export.go` | `GET /api/v1/export`, `GET /api/v1/export/refactoring` | JSON exports |
| `handler_config.go` | `GET /api/v1/config` | Current config (sanitized, no secrets) |
| `handler_gravity.go` | `GET /api/v1/gravity-scores` | Resource gravity scores |

---

### Phase 7: Frontend — React Dashboard

#### [NEW] web/ (Vite project)

Initialize with:
```bash
npm create vite@latest web -- --template react
cd web && npm install
npm install recharts react-router-dom @tanstack/react-query tailwindcss @tailwindcss/vite
```

#### Frontend Pages

| Page | Route | Description |
|------|-------|-------------|
| **Dashboard** | `/` | Main timeline with CPU line + RAM area chart per deployment. Time range selector (1h/6h/24h/7d). Deployment selector dropdown. Click spike marker to drill-down. |
| **SpikeList** | `/spikes` | Sortable table of all spike events. Sort by time/CPU/RAM/deployment. Click to view details modal. |
| **SpikeExplorer** | `/explorer` | Historical analysis. Date range picker + window size selector (5m/15m/30m/1h). Namespace/deployment filter. Severity badges (critical/medium/low). |
| **GravityScores** | `/gravity` | Resource Gravity Score table. Sorted by score desc. Job-like route tagging. Export button. |

#### Key Components

- **TimelineChart**: Recharts `ComposedChart` with `Line` (CPU, solid) + `Area` (RAM, dashed) per deployment. 20-color palette. Spike markers as reference lines.
- **DeploymentSelector**: Multi-select dropdown showing deployments (not pods) with current CPU/RAM.
- **DeploymentLegend**: Clickable legend to highlight/isolate deployments.
- **SpikeDetailModal**: Shows Trace ID, route, GCP Profiler function/file path on drill-down.
- **TimeRangeSelector**: 1h/6h/24h/7d button group.
- **AnalyzeControls**: Date range pickers + window size dropdown.
- **FilterBar**: Namespace dropdown filter.

#### Design Aesthetics

- **Dark theme** with glassmorphism cards
- **Vibrant accent colors** for CPU (cyan/teal gradient) and RAM (purple/magenta gradient)
- **Micro-animations**: hover effects on cards, smooth chart transitions, loading skeletons
- **Google Fonts**: Inter for UI, JetBrains Mono for metrics/code
- **Responsive grid layout** using Tailwind

---

### Phase 8: Entry Point & Wiring

#### [NEW] cmd/server/main.go

- Load `config.yaml`
- Initialize SQLite database + run migrations
- Create integration clients (Prometheus, SigNoz, GCP Profiler, Discord)
- Create services (detector, correlator, alerter, gravity, analyzer)
- Create API handlers
- Start spike detection goroutine
- Start HTTP server
- Graceful shutdown on SIGINT/SIGTERM

#### [NEW] Makefile

```makefile
dev-backend:    # Run Go backend with hot reload
dev-frontend:   # Run Vite dev server
dev:            # Run both concurrently
build:          # Build Go binary + frontend assets
test:           # Run Go tests
lint:           # Run golangci-lint
```

---

## Open Questions

> [!IMPORTANT]
> 1. **SigNoz/ClickHouse connection details**: The FINAL-REQUIREMENTS mentions internal DNS (`signoz-otel-collector.monitoring.svc.cluster.local:4318`). Should we connect to ClickHouse directly (port 8123 for HTTP, 9000 for native) for trace queries, or go through SigNoz's query API? The SQL queries in the doc target ClickHouse directly.
> 2. **GCP Project ID**: Is there a specific GCP project to configure, or should this be entirely config-driven?
> 3. **Frontend serving**: Should the Go backend serve the built frontend assets (single binary deployment), or should they run as separate services? I'll default to **Go serving static files from `web/dist/`** for simplicity while also supporting separate dev servers.
> 4. **Tailwind CSS version**: The FINAL-REQUIREMENTS specifies Tailwind. Should I use **Tailwind v4** (latest, CSS-first config) or **v3** (utility-first with JS config)?  I'll default to **v4** unless you prefer v3.

---

## Verification Plan

### Automated Tests

1. **Go unit tests**: 
   - Domain entity validation
   - Spike detection algorithm (moving average, threshold, cooldown)
   - Gravity score calculation
   - Historical analysis sliding window
   - PromQL query builder output verification
   - ClickHouse SQL query builder output verification
   
2. **Integration tests** (with mocked external services):
   - Prometheus client with mock HTTP server
   - SigNoz client with mock ClickHouse responses
   - GCP Profiler client with mock gRPC
   - Discord webhook with mock server
   - SQLite repository operations

3. **Build verification**:
   ```bash
   go build ./cmd/server/         # Backend compiles
   cd web && npm run build        # Frontend builds
   go test ./...                  # All tests pass
   ```

### Manual Verification

1. **Dashboard visual inspection** via browser:
   - Timeline chart renders with deployment-level data
   - Deployment selector works (not pod selector)
   - Spike markers appear on timeline
   - Drill-down shows Trace ID + GCP Profiler data
   - Time range switching works
   
2. **Integration smoke tests** (with real services):
   - Prometheus queries return deployment-level metrics
   - SigNoz queries return traces correlated by deployment
   - GCP Profiler returns profile data for service
   - Discord webhook delivers formatted alerts

---

*End of Implementation Plan*
