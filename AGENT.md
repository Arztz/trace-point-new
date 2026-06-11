# Trace-Point Agent System

**Project:** Trace-Point v1.0.3 — Resource-to-Code Correlation Engine
**Tech Stack:** Go + chi, React 18 + Vite + Tailwind CSS v4, SQLite, Prometheus, SigNoz/ClickHouse, GCP Cloud Profiler

---

## Entry Point (CRITICAL)

**ALL user input goes to orchestrator FIRST. No other agent should respond directly to user.**

```
USER → orchestrator → [appropriate agent]
```

---

## Agent Hierarchy

```
orchestrator (Level 0)
└── tech-lead (Level 1) — Coordinator, task creator, reports to orchestrator
    ├── developer-lead (Level 2) — Technical execution lead
    │   ├── backend-lead → backend-senior → [backend-junior-a, backend-junior-b]
    │   ├── frontend-lead → frontend-senior → [frontend-junior-a, frontend-junior-b]
    │   └── ux-ui-lead → [ux-senior, ui-senior]
    ├── qa-lead (Level 2) — QA lead
    │   └── qa-senior → [qa-tester-a, qa-tester-b]
    └── infra-lead (Level 2) — Infrastructure lead
        ├── devops-lead → devops-senior
        └── sre-lead → sre-senior
```

---

## Agent Definitions

### tech-lead

**Responsibilities:**
- Coordinate all development work
- Create and assign tasks to team leads
- Manage priorities and dependencies
- Report status to orchestrator

**Domain:** Full stack (Go backend, React frontend, infrastructure)

**Key Files:**
- `cmd/server/main.go` — Entry point
- `config.yaml` — Configuration
- `internal/` — Business logic

---

### backend-lead → backend-senior

**Responsibilities:**
- Go backend architecture and implementation
- API design (chi router)
- Service layer (detector, analyzer, correlator, alerter, gravity)
- Database schema (SQLite)
- External integrations (Prometheus, SigNoz, GCP)

**Key Directories:**
- `internal/api/` — HTTP handlers and router
- `internal/service/` — Business logic services
- `internal/repository/` — Data access
- `internal/domain/` — Domain models
- `internal/integration/` — External service clients
- `internal/integration/prometheus/` — Prometheus client (client.go, queries.go)
- `internal/config/` — Configuration loading
- `internal/middleware/` — HTTP middleware

**Key API Endpoints:**
- `GET /api/v1/timeline?time_range=1h|3h|5h|5d` — Metrics + summaries
- `GET /api/v1/spikes` — Spike detection alerts
- `GET /api/v1/datasources` — Available datasources

**Code Patterns:**
- Handlers call services, services call repositories
- Domain models in `internal/domain/`
- Config loaded from `config.yaml` via `internal/config/`
- Discord alerts via webhook in `internal/service/alerter.go`
- Prometheus client uses `QueryTimelineMetrics()` for timeline data

---

### frontend-lead → frontend-senior

**Responsibilities:**
- React 18 + Vite frontend
- Tailwind CSS v4 styling
- Recharts visualizations
- UI/UX implementation per DESIGN.md

**Key Directories:**
- `web/src/` — React source
- `web/src/pages/` — Page components (Dashboard.jsx)
- `web/src/components/` — Reusable components (TimelineChart, NamespaceSelector, TimeRangeSelector, DeploymentSelector)
- `web/src/hooks/` — Custom React hooks (useData.js)
- `web/src/utils/` — Utility functions (formatters.js, api.js)

**Key Frontend Components:**

**Dashboard.jsx** — Main dashboard page:
- State: `selectedNamespace`, `highlighted`, `cpuFilter`, `ramFilter`, `timeRange`
- `filteredSummary` — derived from `data.summary`, filtered by all filters
- `deploymentNames` — derived from `filteredSummary` for consistent colors
- `deploymentColorMap` — maps deployment to color using `getDeploymentColor(index)`
- Namespace filtering is frontend-only (not sent to backend)

**TimelineChart.jsx** — Graph component:
- Uses `deployments` prop for consistent color ordering (not from metrics)
- Filter by `highlighted` (single deployment) or `deployments` list
- Tooltip uses `formatShortDateTime()` for full date+time

**NamespaceSelector.jsx** — Namespace dropdown:
- Extracts unique namespaces from `available_deployments`
- Default option: "All Namespace"
- Filters `filteredSummary` by `s.namespace === selectedNamespace`

**Dashboard Card Layout (3 columns):**
```
| Avg CPU | Max CPU | Limit CPU |
| Avg RAM | Max RAM | Limit RAM |
```
- `max_cpu` / `max_ram` — of request (no recommend badge)
- `max_cpu_of_limit` / `max_ram_of_limit` — of limit (with recommend badge)
- Color based on classification: `high`=red, `medium`=yellow, `ok/low`=green

**Design System (see DESIGN.md):**
- White-dominant layout with colorful product card accents
- Multi-font: DM Sans (UI), Outfit (display), Poppins (mid-tier), Roboto (data)
- Pill buttons (9999px radius) for nav/tabs, standard (8px) for CTAs
- Purple-tinted shadows for featured elements
- Border radius: 20–24px for cards, 8–13px for UI elements

---

### ux-ui-lead → ux-senior, ui-senior

**Responsibilities:**
- UX research and wireframes
- UI design and component styling
- Design system maintenance (DESIGN.md)

**Key Files:**
- `DESIGN.md` — Complete design system documentation

---

### qa-lead → qa-senior → qa-tester-a, qa-tester-b

**Responsibilities:**
- Test strategy and planning
- API testing (spike detection, correlation, gravity scores)
- E2E testing of React dashboard
- Integration testing (Prometheus, SigNoz, Discord)

---

### infra-lead → devops-lead, devops-senior

**Responsibilities:**
- Docker containerization
- CI/CD pipeline
- Deployment automation

**Key Files:**
- `Dockerfile` — Backend container
- `Dockerfile.frontend` — Frontend container
- `docker-compose.yml` — Local development
- `Makefile` — Build targets

---

### sre-lead → sre-senior

**Responsibilities:**
- Production reliability
- Monitoring and alerting
- Incident response

**Key Areas:**
- Discord webhook integration
- Prometheus metrics scraping
- GCP Cloud Profiler health

---

## Domain Models

### DeploymentSummary (from `/api/v1/timeline`)

```json
{
  "deployment_name": "iic-dashboard",
  "namespace": "fundii",
  "avg_cpu": 13.0,
  "max_cpu": 207.0,
  "avg_ram": 88.5,
  "max_ram": 395.0,
  "avg_cpu_of_limit": 6.7,
  "max_cpu_of_limit": 106.4,
  "avg_ram_of_limit": 22.1,
  "max_ram_of_limit": 98.9,
  "classification": "balanced",
  "cpu_classification": "low",
  "ram_classification": "ok"
}
```

### Classification Logic (based on LIMIT)

Defined in `internal/domain/timeline.go`:
```go
func ClassifyResourceOfLimit(avgOfLimit float64) string {
    if avgOfLimit >= 50 { return "high" }
    if avgOfLimit <= 10 { return "low" }
    return "ok"
}
```

- `high`: avgOfLimit >= 50%
- `low`: avgOfLimit <= 10%
- `ok`: 10% < avgOfLimit < 50%

### Prometheus Queries

**CPU Utilization of Request:**
```promql
sum by (pod, deployment, namespace) (label_replace(rate(container_cpu_usage_seconds_total{...}[5m]), "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+")) 
/ sum by (pod, deployment, namespace) (label_replace(kube_pod_container_resource_requests{...}, "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"))
```

**CPU Utilization of Limit:**
```promql
sum by (pod, deployment, namespace) (label_replace(rate(container_cpu_usage_seconds_total{...}[5m]), "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+")) 
/ sum by (pod, deployment, namespace) (label_replace(kube_pod_container_resource_limits{...}, "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"))
```

---

## Key Features for Development Context

| Feature | Backend | Frontend |
|---------|---------|----------|
| **Timeline Metrics** | `internal/api/handler_timeline.go`, `internal/integration/prometheus/` | Dashboard cards + TimelineChart |
| **Spike Detection** | `internal/service/detector.go` | Spike timeline chart |
| **Deployment-Level Metrics** | Aggregated in services | Dashboard cards (avg/max/limit) |
| **Namespace Filtering** | Frontend-only via `filteredSummary` | NamespaceSelector dropdown |
| **SigNoz Correlation** | `internal/service/correlator.go` | Correlation results view |
| **GCP Profiler** | `internal/integration/` | Profiler data display |
| **Discord Alerts** | `internal/service/alerter.go` | Alert configuration UI |
| **Gravity Scores** | `internal/service/gravity.go` | Gravity score dashboard |

---

## Communication Protocol

All agents follow the delegation protocol in `.pi/DELEGATION-PROTOCOL.md`:
- Task creation via `todo` tool
- Status updates and blockers
- Sync points for multi-agent coordination

---

## Project Memory

Use `memory_search` to recall:
- Previous design decisions
- Bug fixes and workarounds
- Integration configurations
- User preferences