# Multi-Datasource Support Plan

To support multiple datasources (e.g., `fundii-dev`, `fundii-uat`, `roa-uat`, `roa-prod`), the system will be refactored to be **datasource-aware** across the entire stack. A single backend instance will manage all datasources.

## Proposed Changes

### 1. Configuration Refactoring (`config.yaml`)
We will shift the data source variables from the root level into a list of `datasources`.

#### [MODIFY] `config.yaml` & `internal/config/config.go`
- **Move** `prometheus`, `signoz` connection details, `gcloud`, and `namespaces` under a `datasources` array.
- **Example structure:**
```yaml
datasources:
  - name: "fundii-dev"
    prometheus:
      url: "http://a.com"
      timeout: 30s
    signoz:
      url: "http://localhost:37357"
      user: "default"
      password: "password"
      database: "signoz_traces"
      env_tag: "dev"  # Filters traces where deployment.environment='dev'
    gcloud:
      project_id: "project-dev"
      env_version_tag: "dev"
      profiler_enabled: true
    namespaces: ["fundii"]
    
  - name: "fundii-prod"
    # ... b.com, etc.
```

---

### 2. Backend Orchestration

#### [MODIFY] `cmd/server/main.go`
- We will initialize **Maps of Clients**: `map[string]*prometheus.Client`, `map[string]*signoz.Client`, etc.
- The `Detector` will be instantiated per datasource and run in parallel as goroutines.

#### [MODIFY] `internal/api/router.go` & `handler_*.go`
- Add a `datasource` query parameter to all API routes (e.g., `/api/spikes`, `/api/deployments`, `/api/timeline`).
- The handlers will read the `datasource` string and use the appropriate clients from the maps.
- Expose a new API `/api/datasources` to list available datasources for the frontend dropdown.

---

### 3. Database Schema Updates

#### [MODIFY] `data/trace-point.db` schemas (`internal/repository/sqlite.go` & `spike_repo.go`)
- The `spike_events` table needs a new column: `datasource TEXT NOT NULL DEFAULT 'default'` to segregate spikes between different systems.
- Add SQLite `ALTER TABLE` execution during app startup if the column is missing.
- Update `Create`, `List`, and `GetRecentByDeployment` queries to filter by `datasource`.

---

### 4. Integration Client Updates

#### [MODIFY] `internal/integration/signoz/client.go` & `queries.go`
- Update ClickHouse SQL queries to dynamically filter by `deployment.environment = ?` using the `env_tag` defined in the struct.

#### [MODIFY] `internal/integration/profiler/client.go`
- Pass the correct GCP Project ID during the function call, and ensure the profile filters match the `env_version_tag`.

---

### 5. Frontend Enhancements

#### [MODIFY] `web/src/components/Layout.jsx` & `pages/Dashboard.jsx` (etc.)
- Create a `DatasourceSelector` dropdown in the global header or dashboard controls.
- Update `web/src/hooks/useData.js` to accept `datasource` and pass `?datasource=fundii-dev` in all `fetch()` calls.
- Save the selected datasource to `localStorage` so it persists between page refreshes.

## Verification Plan

### Automated/Local Tests
- Create a mock `config.yaml` with dummy `fundii-dev` and `fundii-uat` entries.
- Run `make dev` and verify API calls to `/api/datasources` return both targets.
- Ensure the frontend selector switches properly and updates the resulting `/api/timeline?datasource=fundii-dev` calls.
