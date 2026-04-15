# Trace-Point Application Requirements Document (ARD)

## Consolidated Final Specification

**Version:** 1.0.3  
**Date:** 2026-04-15  
**Status:** Final  
**Backend:** Go (chi router)  
**Frontend:** React 18 + Vite + Tailwind + Recharts

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Functional Requirements](#2-functional-requirements)
3. [API Endpoints](#3-api-endpoints)
4. [Data Models](#4-data-models)
5. [Business Logic](#5-business-logic)
6. [Frontend Components](#6-frontend-components)
7. [Configuration](#7-configuration)
8. [Non-Functional Requirements](#8-non-functional-requirements)

---

## 1. Executive Summary

Trace-Point is a Resource-to-Code Correlation Engine that bridges the gap between Kubernetes infrastructure metrics and code-level root cause analysis. It correlates Prometheus metrics with trace data to identify culprit functions during resource spikes.

**Key Features Implemented (v1.0.3):**
- Real-time spike detection using moving average algorithm
- Historical spike analysis (Spike Explorer)
- Timeline visualization with CPU/RAM metrics
- Resource gravity scoring for refactoring recommendations
- Discord webhook alerting
- JSON export for spike history and refactoring intelligence

---

## 2. Functional Requirements

### 2.1 Spike Detection

| Requirement | Description | Status |
|-------------|-------------|--------|
| FR-010 | Poll Prometheus every 30 seconds | ✅ Implemented |
| FR-011 | Spike = Current > Moving Average + Threshold (50%) | ✅ Implemented |
| FR-011a | 30-minute moving average window | ✅ Implemented |
| FR-011b | 5-minute baseline learning period | ✅ Implemented |
| FR-012 | Configurable threshold (default 50%) | ✅ Implemented |
| FR-013 | 8-minute reconciliation buffer | ✅ Implemented |
| FR-015 | Discord alert with Route/Impact/Culprit | ✅ Implemented |
| FR-016 | 15-minute cooldown between alerts | ✅ Implemented |
| FR-018 | SQLite storage with 7-day retention | ✅ Implemented |

### 2.2 Dashboard & Visualization

| Requirement | Description | Status |
|-------------|-------------|--------|
| FR-001 | Unified timeline with CPU line + RAM area chart | ✅ Implemented |
| FR-002 | Spike markers overlay on timeline | ✅ Implemented |
| FR-003 | Time range selection (1h, 6h, 24h, 7d) | ✅ Implemented |
| FR-004 | Click spike to see Trace ID and profiler data | ✅ Implemented |
| FR-008 | Filter by namespace and pod | ✅ Implemented |

### 2.3 Historical Analysis (v1.0.3)

| Requirement | Description | Status |
|-------------|-------------|--------|
| HA-001 | Analyze spikes over custom time range | ✅ Implemented |
| HA-002 | Window size options: 5m, 15m, 30m, 1h | ✅ Implemented |
| HA-003 | Filter by namespace and replicaset | ✅ Implemented |
| HA-004 | Severity classification (critical/medium/low) | ✅ Implemented |
| HA-005 | Pagination support | ✅ Implemented |

### 2.4 Refactoring Intelligence

| Requirement | Description | Status |
|-------------|-------------|--------|
| FR-020 | Resource Gravity Score calculation | ✅ Implemented |
| FR-021 | Identify job-like routes (/tasks/*, /batch/*, /jobs/*) | ✅ Implemented |
| FR-023 | JSON export with recommendations | ✅ Implemented |

---

## 3. API Endpoints

### 3.1 System Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check with timestamp |
| `/metrics` | GET | Prometheus metrics |
| `/api/v1` | GET | API version info |

### 3.2 Spike Events API

| Endpoint | Method | Query Parameters |
|----------|--------|-----------------|
| `/api/v1/spikes` | GET | namespace, pod, limit, offset, sort, order |
| `/api/v1/spikes/:id` | GET | Path param: id |
| `/api/v1/spikes/:id/details` | GET | Path param: id |

### 3.3 Timeline API

| Endpoint | Method | Query Parameters |
|----------|--------|-----------------|
| `/api/v1/timeline` | GET | time_range (1h/6h/24h/7d), pod_name |

**Response:**
```json
{
  "generated_at": "2026-04-14T16:34:44+07:00",
  "start_date": "2026-04-14T15:34:44+07:00",
  "end_date": "2026-04-14T16:34:44+07:00",
  "metrics": [...],
  "spike_markers": [...],
  "availablePods": [...],
  "summary": [...]
}
```

### 3.4 Spike Analysis API (v1.0.3)

| Endpoint | Method | Query Parameters |
|----------|--------|-----------------|
| `/api/v1/spikes/analyze` | GET | start, end, window, namespace, replicaset, threshold, limit, offset |

**Response:**
```json
{
  "request": { "start": "...", "end": "...", "window": "30m", ... },
  "summary": { "total_spikes": 457, "analyzed_replicasets": 32, ... },
  "spikes": [...],
  "pagination": { "limit": 1000, "offset": 0, "has_more": false }
}
```

### 3.5 Export API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/export` | GET | Export spike history (7 days) |
| `/api/v1/export/refactoring` | GET | Export refactoring recommendations |

### 3.6 Configuration & Scores

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/config` | GET | Get application config |
| `/api/v1/gravity-scores` | GET | Get resource gravity scores |

---

## 4. Data Models

### 4.1 Prometheus Query Types

Trace-Point uses two types of Prometheus API queries:

#### Instant Query (`/api/v1/query`)
Returns a single value at a specific point in time.
- **Use case:** Current metrics, spike detection, available pods
- **Response format:** `Value` field contains `[timestamp, value]`

#### Range Query (`/api/v1/query_range`)
Returns a series of values over a time range.
- **Use case:** Timeline visualization, historical analysis
- **Response format:** `Values` field contains `[[timestamp, value], ...]`

### 4.2 Prometheus Query Catalog

#### Q-001: CPU Utilization (Instant)
**Builder:** `BuildCPUUtilizationQuery()` / `BuildCPUUtilizationQueryWithAggregation()`

**PromQL (Pod-level):**
```promql
(sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{namespace="fundii", container!=""}[5m])) / 
  sum by (pod, namespace, container) (kube_pod_container_resource_requests{namespace="fundii", container!="", resource="cpu"})) * 100
```

**PromQL (Deployment-level):**
```promql
(sum by (deployment, namespace, container) (
    label_replace(
        rate(container_cpu_usage_seconds_total{namespace="fundii", container!=""}[5m]),
        "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
    )
)) / 
(sum by (deployment, namespace, container) (
    label_replace(
        kube_pod_container_resource_requests{namespace="fundii", container!="", resource="cpu"},
        "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
    )
)) * 100
```

**Returns:** CPU utilization as percentage (0-100+) of CPU request
**Labels:** `pod`, `namespace`, `container`

---

#### Q-002: RAM Utilization (Instant)
**Builder:** `BuildRAMUtilizationQuery()` / `BuildRAMUtilizationQueryWithAggregation()`

**PromQL (Pod-level):**
```promql
(sum by (pod, namespace, container) (rate(container_memory_working_set_bytes{namespace="fundii", container!=""}[5m])) / 
  sum by (pod, namespace, container) (kube_pod_container_resource_requests{namespace="fundii", container!="", resource="memory"})) * 100
```

**PromQL (Deployment-level):**
```promql
(sum by (deployment, namespace, container) (
    label_replace(
        rate(container_memory_working_set_bytes{namespace="fundii", container!=""}[5m]),
        "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
    )
)) / 
(sum by (deployment, namespace, container) (
    label_replace(
        kube_pod_container_resource_requests{namespace="fundii", container!="", resource="memory"},
        "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
    )
)) * 100
```

**Returns:** RAM utilization as percentage (0-100+) of memory request
**Labels:** `pod`, `namespace`, `container`

---

#### Q-003: CPU Utilization (Range)
**Builder:** `QueryTimelineMetrics()` with CPU query

**Usage:** Timeline visualization with configurable step
- 7d: 5-minute step (2016 points)
- 3d: 2-minute step (2160 points)
- 24h: 1-minute step (1440 points)
- 1h/6h: 30-second step (default)

**Returns:** Time series of CPU percentages per pod/container
**Used by:** `/api/v1/timeline`

---

#### Q-004: RAM Utilization (Range)
**Builder:** `QueryTimelineMetrics()` with RAM query

**Usage:** Timeline visualization with configurable step
**Returns:** Time series of RAM percentages per pod/container
**Used by:** `/api/v1/timeline`

---

#### Q-005: Container CPU History
**Builder:** `BuildContainerCPUQuery(podName, namespace, containerName)`

**PromQL:**
```promql
(sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{pod="payment-service-abcde", namespace="fundii", container="payment-service"}[5m])) / 
  sum by (pod, namespace, container) (kube_pod_container_resource_requests{pod="payment-service-abcde", namespace="fundii", container="payment-service", resource="cpu"})) * 100
```

**Returns:** Historical CPU percentage for a specific container
**Used by:** Spike detection historical analysis

---

#### Q-006: Container RAM History
**Builder:** `BuildContainerRAMQuery(podName, namespace, containerName)`

**PromQL:**
```promql
(sum by (pod, namespace, container) (rate(container_memory_working_set_bytes{pod="payment-service-abcde", namespace="fundii", container="payment-service"}[5m])) / 
  sum by (pod, namespace, container) (kube_pod_container_resource_requests{pod="payment-service-abcde", namespace="fundii", container="payment-service", resource="memory"})) * 100
```

**Returns:** Historical RAM percentage for a specific container
**Used by:** Spike detection historical analysis

---

### 4.4 Prometheus API Response Structures

#### MetricQueryResult (Instant Query)
```go
type MetricQueryResult struct {
    Status string `json:"status"`           // "success" or "error"
    Data   Data   `json:"data"`
}

type Data struct {
    ResultType string   `json:"resultType"`  // "vector"
    Result     []Result `json:"result"`
}

type Result struct {
    Metric map[string]string `json:"metric"`  // Labels: pod, namespace, container, etc.
    Value  interface{}       `json:"value"`   // [timestamp (float64), value (float64)]
}
```

**Example Response:**
```json
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "pod": "payment-service-abcde",
          "namespace": "fundii",
          "container": "payment-service"
        },
        "value": [1713115200.000, 45.23]
      }
    ]
  }
}
```

---

#### RangeQueryResult (Range Query)
```go
type RangeQueryResult struct {
    Status string    `json:"status"`
    Data   RangeData `json:"data"`
}

type RangeData struct {
    ResultType string        `json:"resultType"`  // "matrix"
    Result     []RangeResult `json:"result"`
}

type RangeResult struct {
    Metric map[string]string `json:"metric"`  // Labels
    Values [][]interface{}   `json:"values"`  // [[timestamp, value], [timestamp, value], ...]
}
```

**Example Response:**
```json
{
  "status": "success",
  "data": {
    "resultType": "matrix",
    "result": [
      {
        "metric": {
          "pod": "payment-service-abcde",
          "namespace": "fundii",
          "container": "payment-service"
        },
        "values": [
          [1713115200.000, "12.45"],
          [1713115230.000, "15.32"],
          [1713115260.000, "45.23"]
        ]
      }
    ]
  }
}
```

---

### 4.5 Signoz/ClickHouse Query Structures

#### Trace Query
**Builder:** `BuildTraceQuery()`

**SQL:**
```sql
SELECT 
    traceID,
    spanName,
    serviceName,
    toUnixTimestamp(startTime/1000000000) as startTime,
    durationNano as duration,
    JSONExtractString(attributes, 'http.url') as httpUrl,
    JSONExtractString(attributes, 'http.method') as httpMethod,
    JSONExtractInt(attributes, 'http.status_code') as statusCode,
    JSONExtractString(attributes, 'k8s.pod.name') as podName,
    JSONExtractString(attributes, 'k8s.namespace.name') as namespace,
    JSONExtractString(attributes, 'span.kind') as kind
FROM signoz_traces.distributed_signoz_index_v2
WHERE startTime >= {startTs} AND startTime <= {endTs}
  AND JSONExtractString(attributes, 'k8s.namespace.name') = '{namespace}'
ORDER BY startTime DESC
LIMIT {limit}
```

---

### 4.6 SpikeEvent

```go
type SpikeEvent struct {
    ID                   string    // UUID
    Timestamp            time.Time // When spike occurred
    PodName              string    // Affected pod
    Namespace            string    // K8s namespace
    CPUUsagePercent      float64   // Current CPU %
    CPULimitPercent      float64   // CPU limit %
    RAMUsagePercent      float64   // Current RAM %
    RAMLimitPercent      float64   // RAM limit %
    ThresholdPercent     float64   // Configured threshold
    MovingAveragePercent float64   // Calculated average
    RouteName            *string   // Primary route during spike
    TraceID              *string   // Associated trace
    CulpritFunction      *string   // Function from profiler
    CulpritFilePath      *string   // File path
    AlertSent            bool      // Discord alert sent
    CooldownEnd          *time.Time // Cooldown end time
    CreatedAt            time.Time // Record creation time
}
```

### 4.7 TimelineMetric

```go
type TimelineMetric struct {
    Timestamp      time.Time // Data point timestamp
    ReplicasetName string    // Deployment name
    PodName        string    // Individual pod
    Namespace      string    // K8s namespace
    CPUPercent     float64   // CPU utilization %
    RAMPercent     float64   // RAM utilization %
}
```

### 4.8 HistoricalSpike (v1.0.3)

```go
type HistoricalSpike struct {
    ID               string    // Unique ID
    Timestamp        time.Time // Spike timestamp
    ReplicasetName   string    // Service/deployment name
    PodName          string    // Individual pod
    Namespace        string    // K8s namespace
    ContainerName    string    // Container name
    Type             string    // "cpu", "ram", "both"
    CPUPercent       float64   // Current CPU %
    RAMPercent       float64   // Current RAM %
    MovingAverageCPU float64   // Moving average CPU
    MovingAverageRAM float64   // Moving average RAM
    ThresholdPercent float64   // Threshold used
    DeviationPercent float64   // % above average
    Severity         string    // "critical", "medium", "low"
}
```

### 4.9 GravityScore

```go
type GravityScore struct {
    ServiceName           string  // namespace/service
    RouteName             string  // API route
    SpikeCount            int     // Total spikes in period
    MaxCPUPercent         float64 // Peak CPU
    MaxRAMPercent         float64 // Peak RAM
    AverageCPUPercent     float64 // Average CPU
    AverageRAMPercent     float64 // Average RAM
    ResourceGravityScore   float64 // Calculated score
}
```

### 4.10 AvailablePod (Frontend Dropdown)

```go
type AvailablePod struct {
    Name       string  `json:"name"`        // Replicaset name (grouped)
    Namespace  string  `json:"namespace"`    // Namespace
    PodCount   int     `json:"pod_count"`   // Number of pods in this replicaset
    CurrentCPU float64 `json:"current_cpu"` // Aggregated CPU (average across all pods)
    CurrentRAM float64 `json:"current_ram"` // Aggregated RAM (average across all pods)
}
```

### 4.11 ContainerMetrics (Internal)

```go
type ContainerMetrics struct {
    PodName       string
    Namespace     string
    ContainerName string
    CPUPercent    float64
    RAMPercent    float64
    Timestamp     time.Time
}
```

### 4.12 RouteActivity (Signoz Correlation)

```go
type RouteActivity struct {
    Route          string  `json:"route"`
    TraceCount     int     `json:"trace_count"`
    TotalDuration  int64   `json:"total_duration_ms"`
    ErrorCount     int     `json:"error_count"`
    ResourceWeight float64 `json:"resource_weight"`
}
```

### 4.13 CorrelationResult (Signoz Correlation)

```go
type CorrelationResult struct {
    FoundTraces     bool            `json:"found_traces"`
    TraceCount      int             `json:"trace_count"`
    RouteActivities []RouteActivity `json:"route_activities"`
    CulpritRoute    string          `json:"culprit_route"`
    CulpritScore    float64         `json:"culprit_score"`
    TraceID         string          `json:"trace_id"`
}
```

---

## 5. Business Logic

### 5.1 Spike Detection Algorithm

```
Formula: Spike = Current Usage > Moving Average + (Threshold %)

Example:
- Moving Average: 50%
- Threshold: 50%
- Trigger: > 75% (50% + 50%)
```

**Configuration:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| PollingIntervalSeconds | 30 | Prometheus poll frequency |
| ThresholdPercent | 50.0 | Deviation threshold % |
| MovingAverageWindowMinutes | 30 | Moving average window |
| ReconciliationBufferMinutes | 8 | Buffer before alerting |
| CooldownMinutes | 15 | Alert cooldown |

### 5.2 Historical Spike Analysis (v1.0.3)

**Algorithm:** Sliding Window Spike Detection

For each point Pi in time range:
1. Window = points from (Pi - window_size) to (Pi-1)
2. Moving Average = sum(previous_points) / count(previous_points)
3. Deviation % = (Pi - MovingAverage) / MovingAverage × 100
4. If Deviation > Threshold → SPIKE!

**Severity Classification:**
| Deviation | Severity |
|-----------|-----------|
| > 200% | critical |
| 100-200% | medium |
| 50-100% | low |
| < 50% | normal |

### 5.3 Resource Gravity Score

```
Resource Gravity Score = Resource Peak × (1 / Request Frequency)

Where:
- Resource Peak = max(CPU peak, RAM peak) over 7 days
- Request Frequency = requests per day

Score Range:
- 0-3: Low impact (monitor)
- 3-6: Medium impact (consider optimization)
- 6-10+: High impact (priority refactoring)
```

### 5.4 Route Selection (Culprit Detection)

When multiple routes active during spike:
- **Primary:** Highest CPU consumer
- **Secondary:** Highest RAM consumer

### 5.5 Timeline Summary Classification

```go
if avgCPU >= 85 || maxCPU > 100 {
    classification = "needs_more_cpu"
} else if avgCPU <= 50 && maxCPU <= 100 {
    classification = "overprovisioned"
} else {
    classification = "balanced"
}
```

---

## 6. Frontend Components

### 6.1 Pages

| Page | Description |
|------|-------------|
| Dashboard | Main timeline view with CPU/RAM charts |
| SpikeList | List of spike events with sorting |
| SpikeExplorer | Historical spike analysis (v1.0.3) |
| GravityScores | Resource gravity scoring table |

### 6.2 Components

| Component | Description |
|-----------|-------------|
| TimelineChart | Recharts-based timeline with CPU line + RAM area |
| PodSelector | Multi-select dropdown for pod filtering |
| PodLegend | Clickable legend for highlighting |
| SpikeAnalysisTable | Sortable table for historical analysis |
| AnalyzeControls | Date range picker + window selector |
| TimeRangeSelector | 1h/6h/24h/7d selection |
| FilterBar | Namespace and pod filters |

### 6.3 Features Implemented

- [x] Multi-line chart (solid=CPU, dashed=RAM per pod)
- [x] 20-color palette for pod differentiation
- [x] PodLegend click-to-highlight
- [x] Sort dropdown (Time/CPU/RAM/Pod, asc/desc)
- [x] Baseline comparison tooltip (green/yellow/red)
- [x] Spike Explorer tab with date picker
- [x] Window size selector (5m/15m/30m/1h)

---

## 7. Configuration

### 7.1 config.yaml

```yaml
app:
  host: "0.0.0.0"
  port: 8088

prometheus:
  url: "http://prod.prometheus.fundii-prod.internal"
  timeout: 30s
  use_deployment_aggregation: true

detection:
  cpu_threshold: 80
  memory_threshold: 85

timeline:
  cpu_close_to_100_threshold: 85
  cpu_far_below_100_threshold: 50

discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/..."

database:
  type: "sqlite"
  path: "./data/trace-point.db"

namespaces:
  - "fundii"

pod_exclude_patterns:
  - "^kube-"
  - "^system-"
```

---

## 8. Non-Functional Requirements

### 8.1 Performance

| Metric | Target | Status |
|--------|--------|--------|
| Prometheus query | ≤5s | ✅ |
| Dashboard load | ≤3s | ✅ |
| Alert generation | ≤2min | ✅ |
| Timeline 7d query | ≤30s | ✅ (dynamic step) |

### 8.2 Database

- SQLite with WAL mode
- Auto-purge after 7 days
- Schema version 2
- Tables: spike_events, config, schema_migrations, metrics_cache

### 8.3 Integrations

| Service | Status | Notes |
|---------|--------|-------|
| Prometheus | ✅ | Real data |
| Signoz/Clickhouse | ✅ Stub | Full impl available |
| GCloud Profiler | ✅ Stub | Full impl available |
| Discord | ✅ | Real webhooks |

---

## Appendix A: API Response Formats

### A.1 Timeline Response

```json
{
  "generated_at": "2026-04-14T16:34:44+07:00",
  "start_date": "2026-04-14T15:34:44+07:00",
  "end_date": "2026-04-14T16:34:44+07:00",
  "metrics": [
    {
      "timestamp": "2026-04-14T15:34:44+07:00",
      "replicaset_name": "payment-service",
      "pod_name": "payment-service-abcde",
      "namespace": "fundii",
      "cpu_percent": 45.2,
      "ram_percent": 62.8
    }
  ],
  "spike_markers": [...],
  "availablePods": [...],
  "summary": [...]
}
```

### A.2 Spike Analysis Response

```json
{
  "request": {
    "start": "2026-04-13T16:34:45+07:00",
    "end": "2026-04-14T16:34:45+07:00",
    "window": "30m",
    "namespace": "",
    "replicaset": "",
    "threshold": 50,
    "limit": 1000,
    "offset": 0
  },
  "summary": {
    "total_spikes": 457,
    "time_range_hours": 24.0,
    "analyzed_replicasets": 32,
    "spikes_by_type": { "cpu": 457 },
    "top_replicasets": [...]
  },
  "spikes": [...],
  "pagination": {
    "limit": 1000,
    "offset": 0,
    "has_more": false
  }
}
```

---

## Appendix B: Calculation Formulas

### B.1 Spike Detection
```
Spike Detected = Current Usage > Moving Average + (Threshold %)

Example:
- Moving Average: 50%
- Threshold: 50%
- Trigger: > 75%
```

### B.2 Resource Gravity Score
```
Resource Gravity Score = Resource Peak × (1 / Request Frequency)

Where:
- Resource Peak = max(CPU peak, RAM peak) over 7 days
- Request Frequency = requests per day
```

### B.3 Baseline Comparison
```
vs Baseline % = ((Current - Baseline) / Baseline) × 100

Indicators:
- Green (▼): Below baseline
- Yellow (±): Near baseline (within 20%)
- Red (▲): Above baseline (spike)
```

### B.4 Timeline 7d Dynamic Step
```
- 7d: 5-minute step (2016 points)
- 3d: 2-minute step (2160 points)  
- 24h: 1-minute step (1440 points)
- 1h/6h: 30-second step (default)
```

---

*End of Consolidated ARD v1.0.3*