package prometheus

import (
	"os"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
)

// Client is the Prometheus HTTP API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	namespaces []string
}

// NewClient creates a new Prometheus client.
func NewClient(baseURL string, timeout time.Duration, namespaces []string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		namespaces: namespaces,
	}
}

// MetricQueryResult represents a Prometheus instant query response.
type MetricQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// RangeQueryResult represents a Prometheus range query response.
type RangeQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// QueryInstantCPU fetches current CPU utilization per deployment.
func (c *Client) QueryInstantCPU() ([]domain.ContainerMetrics, error) {
	query := BuildCPUUtilizationQuery(c.namespacesRegex())
	return c.queryInstant(query)
}

// QueryInstantRAM fetches current RAM utilization per deployment.
func (c *Client) QueryInstantRAM() ([]domain.ContainerMetrics, error) {
	query := BuildRAMUtilizationQuery(c.namespacesRegex())
	return c.queryInstant(query)
}

// QueryInstantMetrics fetches both CPU and RAM metrics and merges them.
func (c *Client) QueryInstantMetrics() ([]domain.ContainerMetrics, error) {
	cpuMetrics, err := c.QueryInstantCPU()
	if err != nil {
		return nil, fmt.Errorf("failed to query CPU: %w", err)
	}

	ramMetrics, err := c.QueryInstantRAM()
	if err != nil {
		return nil, fmt.Errorf("failed to query RAM: %w", err)
	}

	// Merge CPU and RAM into a single slice keyed by deployment+namespace
	merged := make(map[string]*domain.ContainerMetrics)
	for i := range cpuMetrics {
		key := cpuMetrics[i].DeploymentName + "/" + cpuMetrics[i].Namespace
		merged[key] = &cpuMetrics[i]
	}
	for _, ram := range ramMetrics {
		key := ram.DeploymentName + "/" + ram.Namespace
		if m, ok := merged[key]; ok {
			m.RAMPercent = ram.CPUPercent
		} else {
			r := ram
			merged[key] = &r
		}
	}

	result := make([]domain.ContainerMetrics, 0, len(merged))
	for _, m := range merged {
		result = append(result, *m)
	}
	return result, nil
}

// QueryTimelineMetrics fetches CPU and RAM time series for the timeline chart,
// including both request-based and limit-based utilization.
func (c *Client) QueryTimelineMetrics(start, end time.Time, deploymentFilter string) ([]domain.TimelineMetric, error) {
	step := calculateStep(start, end)

	// Query request-based CPU and RAM
	cpuQuery := BuildCPUUtilizationQuery(c.namespacesRegex())
	ramQuery := BuildRAMUtilizationQuery(c.namespacesRegex())

	cpuResults, err := c.queryRange(cpuQuery, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("failed to query CPU timeline: %w", err)
	}

	ramResults, err := c.queryRange(ramQuery, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("failed to query RAM timeline: %w", err)
	}

	// Query limit-based CPU and RAM
	cpuLimitQuery := BuildCPUUtilizationOfLimitQuery(c.namespacesRegex())
	ramLimitQuery := BuildRAMUtilizationOfLimitQuery(c.namespacesRegex())

	// DEBUG: Log limit queries
	log.Printf("[Timeline] CPU Limit Query: %s", cpuLimitQuery)
	log.Printf("[Timeline] RAM Limit Query: %s", ramLimitQuery)

	cpuLimitResults, err := c.queryRange(cpuLimitQuery, start, end, step)
	if err != nil {
		log.Printf("[Prometheus] Warning: failed to query CPU limit timeline: %v", err)
		// Continue without limit data
		cpuLimitResults = &RangeQueryResult{}
	}

	ramLimitResults, err := c.queryRange(ramLimitQuery, start, end, step)
	if err != nil {
		log.Printf("[Prometheus] Warning: failed to query RAM limit timeline: %v", err)
		// Continue without limit data
		ramLimitResults = &RangeQueryResult{}
	}

	// DEBUG: Log limit results count
	log.Printf("[Timeline] CPU limit results count: %d", len(cpuLimitResults.Data.Result))
	log.Printf("[Timeline] RAM limit results count: %d", len(ramLimitResults.Data.Result))

	// DEBUG: Write all results to files
	cpuF, _ := os.Create("/tmp/cpu_request_results.json")
	defer cpuF.Close()
	json.NewEncoder(cpuF).Encode(cpuResults)
	ramF, _ := os.Create("/tmp/ram_request_results.json")
	defer ramF.Close()
	json.NewEncoder(ramF).Encode(ramResults)
	cpuLimitF, _ := os.Create("/tmp/cpu_limit_results.json")
	defer cpuLimitF.Close()
	json.NewEncoder(cpuLimitF).Encode(cpuLimitResults)
	ramLimitF, _ := os.Create("/tmp/ram_limit_results.json")
	defer ramLimitF.Close()
	json.NewEncoder(ramLimitF).Encode(ramLimitResults)
	log.Printf("[DEBUG] Wrote all results to files, CPU request: %d, RAM request: %d, CPU limit: %d, RAM limit: %d", 
		len(cpuResults.Data.Result), 
		len(ramResults.Data.Result), 
		len(cpuLimitResults.Data.Result), 
		len(ramLimitResults.Data.Result))
	ramLookup := make(map[string]float64)
	for _, r := range ramResults.Data.Result {
		deployment := r.Metric["deployment"]
		namespace := r.Metric["namespace"]

		if deploymentFilter != "" && deployment != deploymentFilter {
			continue
		}

		for _, v := range r.Values {
			ts, val := parseRangeValue(v)
			key := fmt.Sprintf("%s/%s/%d", deployment, namespace, ts.Unix())
			ramLookup[key] = val
		}
	}

	// Build CPU limit lookup: deployment+namespace+timestamp -> CPU% of limit
	cpuLimitLookup := make(map[string]float64)
	for _, r := range cpuLimitResults.Data.Result {
		deployment := r.Metric["deployment"]
		namespace := r.Metric["namespace"]

		if deploymentFilter != "" && deployment != deploymentFilter {
			continue
		}

		for _, v := range r.Values {
			ts, val := parseRangeValue(v)
			key := fmt.Sprintf("%s/%s/%d", deployment, namespace, ts.Unix())
			cpuLimitLookup[key] = val
		}
	}

	// Build RAM limit lookup: deployment+namespace+timestamp -> RAM% of limit
	ramLimitLookup := make(map[string]float64)
	for _, r := range ramLimitResults.Data.Result {
		deployment := r.Metric["deployment"]
		namespace := r.Metric["namespace"]

		if deploymentFilter != "" && deployment != deploymentFilter {
			continue
		}

		for _, v := range r.Values {
			ts, val := parseRangeValue(v)
			key := fmt.Sprintf("%s/%s/%d", deployment, namespace, ts.Unix())
			ramLimitLookup[key] = val
		}
	}

	// DEBUG: Log limit lookups
	log.Printf("[Timeline] CPU limit lookup size: %d", len(cpuLimitLookup))
	log.Printf("[Timeline] RAM limit lookup size: %d", len(ramLimitLookup))
	// for k, v := range cpuLimitLookup {
	// 	if strings.Contains(k, "iic-dashboard") {
	// 		log.Printf("[Timeline] iic-dashboard CPU limit key: %s, value: %f", k, v)
	// 	}
	// }

	// Build timeline metrics with both request-based and limit-based values
	var metrics []domain.TimelineMetric
	for _, r := range cpuResults.Data.Result {
		deployment := r.Metric["deployment"]
		namespace := r.Metric["namespace"]

		if deploymentFilter != "" && deployment != deploymentFilter {
			continue
		}

		for _, v := range r.Values {
			ts, cpuVal := parseRangeValue(v)
			key := fmt.Sprintf("%s/%s/%d", deployment, namespace, ts.Unix())
			ramVal := ramLookup[key]
			cpuLimitVal := cpuLimitLookup[key]
			ramLimitVal := ramLimitLookup[key]

			metrics = append(metrics, domain.TimelineMetric{
				Timestamp:         ts,
				DeploymentName:    deployment,
				Namespace:         namespace,
				CPUPercent:        cpuVal,
				RAMPercent:        ramVal,
				CPUPercentOfLimit: cpuLimitVal,
				RAMPercentOfLimit: ramLimitVal,
			})
		}
	}

	return metrics, nil
}

// GetAvailableDeployments returns a list of deployments with current metrics.
func (c *Client) GetAvailableDeployments() ([]domain.AvailableDeployment, error) {
	metrics, err := c.QueryInstantMetrics()
	if err != nil {
		return nil, err
	}

	deployments := make([]domain.AvailableDeployment, 0, len(metrics))
	for _, m := range metrics {
		deployments = append(deployments, domain.AvailableDeployment{
			Name:       m.DeploymentName,
			Namespace:  m.Namespace,
			CurrentCPU: m.CPUPercent,
			CurrentRAM: m.RAMPercent,
		})
	}

	return deployments, nil
}

// QueryDeploymentHistory fetches historical CPU and RAM for a specific deployment.
func (c *Client) QueryDeploymentHistory(deployment, namespace string, start, end time.Time) ([]domain.ContainerMetrics, error) {
	step := calculateStep(start, end)

	cpuQuery := BuildDeploymentCPUQuery(deployment, namespace)
	ramQuery := BuildDeploymentRAMQuery(deployment, namespace)

	cpuResults, err := c.queryRange(cpuQuery, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("CPU history query failed: %w", err)
	}

	ramResults, err := c.queryRange(ramQuery, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("RAM history query failed: %w", err)
	}

	// Build RAM lookup
	ramLookup := make(map[int64]float64)
	for _, r := range ramResults.Data.Result {
		for _, v := range r.Values {
			ts, val := parseRangeValue(v)
			ramLookup[ts.Unix()] = val
		}
	}

	// Merge
	var metrics []domain.ContainerMetrics
	for _, r := range cpuResults.Data.Result {
		for _, v := range r.Values {
			ts, cpuVal := parseRangeValue(v)
			metrics = append(metrics, domain.ContainerMetrics{
				DeploymentName: deployment,
				Namespace:      namespace,
				CPUPercent:     cpuVal,
				RAMPercent:     ramLookup[ts.Unix()],
				Timestamp:      ts,
			})
		}
	}

	return metrics, nil
}

// --- Internal helpers ---

func (c *Client) queryInstant(query string) ([]domain.ContainerMetrics, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("time", fmt.Sprintf("%d", time.Now().Unix()))

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/api/v1/query?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result MetricQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query error: %s (%s)", result.Error, result.ErrorType)
	}

	var metrics []domain.ContainerMetrics
	for _, r := range result.Data.Result {
		if len(r.Value) < 2 {
			continue
		}

		valStr, ok := r.Value[1].(string)
		if !ok {
			continue
		}
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			log.Printf("[Prometheus] Failed to parse value %q: %v", valStr, err)
			continue
		}

		metrics = append(metrics, domain.ContainerMetrics{
			DeploymentName: r.Metric["deployment"],
			Namespace:      r.Metric["namespace"],
			CPUPercent:     val,
			Timestamp:      time.Now(),
		})
	}

	return metrics, nil
}

func (c *Client) queryRange(query string, start, end time.Time, step string) (*RangeQueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", step)

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/api/v1/query_range?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("prometheus range query failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result RangeQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus range query error: %s (%s)", result.Error, result.ErrorType)
	}

	return &result, nil
}

func (c *Client) namespacesRegex() string {
	return strings.Join(c.namespaces, "|")
}

func parseRangeValue(v []interface{}) (time.Time, float64) {
	var ts time.Time
	var val float64

	if len(v) >= 2 {
		if tsFloat, ok := v[0].(float64); ok {
			ts = time.Unix(int64(tsFloat), 0)
		}
		if valStr, ok := v[1].(string); ok {
			val, _ = strconv.ParseFloat(valStr, 64)
		}
	}

	return ts, val
}

func calculateStep(start, end time.Time) string {
	duration := end.Sub(start)

	switch {
	case duration > 5*24*time.Hour: // > 5 days → 5 min step
		return "300"
	case duration > 2*24*time.Hour: // > 2 days → 2 min step
		return "120"
	case duration > 12*time.Hour: // > 12 hours → 1 min step
		return "60"
	default: // ≤ 12 hours → 30s step
		return "30"
	}
}

// DEBUG function to log limit results
func debugLogLimitResults(cpuLimitResults, ramLimitResults *RangeQueryResult) {
    f, _ := os.Create("/tmp/cpu_limit_results.json")
    defer f.Close()
    json.NewEncoder(f).Encode(cpuLimitResults)
    f2, _ := os.Create("/tmp/ram_limit_results.json")
    defer f2.Close()
    json.NewEncoder(f2).Encode(ramLimitResults)
    log.Printf("[DEBUG] Wrote limit results to files")
}
