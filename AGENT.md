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
- `internal/config/` — Configuration loading
- `internal/middleware/` — HTTP middleware

**Code Patterns:**
- Handlers call services, services call repositories
- Domain models in `internal/domain/`
- Config loaded from `config.yaml` via `internal/config/`
- Discord alerts via webhook in `internal/service/alerter.go`

---

### frontend-lead → frontend-senior

**Responsibilities:**
- React 18 + Vite frontend
- Tailwind CSS v4 styling
- Recharts visualizations
- UI/UX implementation per DESIGN.md

**Key Directories:**
- `web/src/` — React source
- `web/src/pages/` — Page components
- `web/src/components/` — Reusable components
- `web/src/hooks/` — Custom React hooks
- `web/src/utils/` — Utility functions

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

## Key Features for Development Context

| Feature | Backend | Frontend |
|---------|---------|----------|
| **Spike Detection** | `internal/service/detector.go` | Spike timeline chart |
| **Deployment-Level Metrics** | Aggregated in services | Dashboard cards |
| **SigNoz Correlation** | `internal/service/correlator.go` | Correlation results view |
| **GCP Profiler** | `internal/integration/` | Profiler data display |
| **Discord Alerts** | `internal/service/alerter.go` | Alert configuration UI |
| **Gravity Scores** | `internal/service/gravity.go` | Gravity score dashboard |
| **Historical Analysis** | Sliding window in analyzer | Timeline explorer |

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