package signoz

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
)

// Client is the SigNoz/ClickHouse HTTP client for trace correlation.
type Client struct {
	baseURL    string
	database   string
	user       string
	password   string
	envTag     string
	httpClient *http.Client
}

// NewClient creates a new SigNoz/ClickHouse client.
func NewClient(baseURL string, database string, user string, password string, envTag string, timeout time.Duration) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		database: database,
		user:     user,
		password: password,
		envTag:   envTag,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// TraceRow represents a single trace span from ClickHouse.
type TraceRow struct {
	TraceID        string `json:"traceID"`
	SpanName       string `json:"spanName"`
	ServiceName    string `json:"serviceName"`
	StartTime      int64  `json:"startTime"`
	Duration       int64  `json:"duration"`
	HTTPUrl        string `json:"httpUrl"`
	HTTPMethod     string `json:"httpMethod"`
	StatusCode     int    `json:"statusCode"`
	DeploymentName string `json:"deploymentName"`
	Namespace      string `json:"namespace"`
	Kind           string `json:"kind"`
}

// ClickHouseResponse represents the response from the ClickHouse HTTP interface.
type ClickHouseResponse struct {
	Data []map[string]interface{} `json:"data"`
}

// QueryTraces fetches traces from ClickHouse for a deployment during a time window.
func (c *Client) QueryTraces(namespace, deployment string, start, end time.Time, limit int) ([]TraceRow, error) {
	if limit <= 0 {
		limit = 100
	}

	query := BuildTraceQuery(c.envTag, namespace, deployment, start, end, limit)

	traces, err := c.executeQuery(query)
	if err != nil {
		return nil, fmt.Errorf("trace query failed: %w", err)
	}

	return traces, nil
}

// CorrelateSpike correlates a spike with trace data to find the culprit route.
func (c *Client) CorrelateSpike(namespace, deployment string, spikeTime time.Time, windowSeconds int) (*domain.CorrelationResult, error) {
	if windowSeconds <= 0 {
		windowSeconds = 5
	}

	maxWindowSeconds := 60
	stepSeconds := 5

	var traces []TraceRow
	var err error

	// Progressively widen the time window until we find traces or hit 60s max
	for currentWindow := windowSeconds; currentWindow <= maxWindowSeconds; currentWindow += stepSeconds {
		start := spikeTime.Add(-time.Duration(currentWindow) * time.Second)
		end := spikeTime.Add(time.Duration(currentWindow) * time.Second)

		traces, err = c.QueryTraces(namespace, deployment, start, end, 500)
		if err != nil {
			return nil, err
		}

		if len(traces) > 0 {
			log.Printf("[SigNoz] Found %d traces with ±%ds window", len(traces), currentWindow)
			break
		}

		log.Printf("[SigNoz] No traces found with ±%ds window, expanding...", currentWindow)
	}

	if len(traces) == 0 {
		log.Printf("[SigNoz] No traces found after expanding to ±%ds, giving up", maxWindowSeconds)
		return &domain.CorrelationResult{
			FoundTraces: false,
			TraceCount:  0,
		}, nil
	}

	// Aggregate routes
	routeMap := make(map[string]*domain.RouteActivity)
	for _, t := range traces {
		route := extractRoute(t)
		if route == "" {
			continue
		}

		if _, ok := routeMap[route]; !ok {
			routeMap[route] = &domain.RouteActivity{Route: route}
		}

		ra := routeMap[route]
		ra.TraceCount++
		ra.TotalDuration += t.Duration / 1000000 // nano to ms
		if t.StatusCode >= 400 {
			ra.ErrorCount++
		}
		// Resource weight = count * avg_duration (proxy for resource consumption)
		ra.ResourceWeight = float64(ra.TraceCount) * float64(ra.TotalDuration) / float64(ra.TraceCount)
	}

	// Convert to slice and sort by resource weight
	activities := make([]domain.RouteActivity, 0, len(routeMap))
	for _, ra := range routeMap {
		activities = append(activities, *ra)
	}
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].ResourceWeight > activities[j].ResourceWeight
	})

	result := &domain.CorrelationResult{
		FoundTraces:     true,
		TraceCount:      len(traces),
		RouteActivities: activities,
	}

	// Set culprit as the highest resource weight route
	if len(activities) > 0 {
		result.CulpritRoute = activities[0].Route
		result.CulpritScore = activities[0].ResourceWeight
	}
	// log.Printf("[SigNoz] Culprit Route: %v", result)
	// Pick a representative trace ID from the culprit route
	for _, t := range traces {
		if extractRoute(t) == result.CulpritRoute {
			result.TraceID = t.SpanName
			break
		}
	}

	return result, nil
}

func (c *Client) executeQuery(query string) ([]TraceRow, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("default_format", "JSON")

	reqURL := fmt.Sprintf("%s/?%s", c.baseURL, params.Encode())

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ClickHouse failed to build request: %w", err)
	}

	if c.user != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ClickHouse query failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ClickHouse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ClickHouse returned status %d: %s", resp.StatusCode, string(body))
	}

	var chResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &chResp); err != nil {
		return nil, fmt.Errorf("failed to parse ClickHouse response: %w", err)
	}

	traces := make([]TraceRow, 0, len(chResp.Data))
	for _, row := range chResp.Data {
		t := TraceRow{}
		if v, ok := row["traceID"].(string); ok {
			t.TraceID = v
		}
		if v, ok := row["spanName"].(string); ok {
			t.SpanName = v
		}
		if v, ok := row["serviceName"].(string); ok {
			t.ServiceName = v
		}
		if v, ok := row["startTime"].(float64); ok {
			t.StartTime = int64(v)
		}
		if v, ok := row["duration"].(float64); ok {
			t.Duration = int64(v)
		}
		if v, ok := row["httpUrl"].(string); ok {
			t.HTTPUrl = v
		}
		if v, ok := row["httpMethod"].(string); ok {
			t.HTTPMethod = v
		}
		if v, ok := row["statusCode"].(float64); ok {
			t.StatusCode = int(v)
		}
		if v, ok := row["deploymentName"].(string); ok {
			t.DeploymentName = v
		}
		if v, ok := row["namespace"].(string); ok {
			t.Namespace = v
		}
		if v, ok := row["kind"].(string); ok {
			t.Kind = v
		}
		traces = append(traces, t)
	}

	log.Printf("[SigNoz] Fetched %d traces", len(traces))
	return traces, nil
}

func extractRoute(t TraceRow) string {
	if t.HTTPUrl != "" {
		// Extract path from URL
		parsed, err := url.Parse(t.HTTPUrl)
		if err == nil {
			return parsed.Path
		}
		return t.HTTPUrl
	}
	if t.SpanName != "" {
		return t.SpanName
	}
	return ""
}
