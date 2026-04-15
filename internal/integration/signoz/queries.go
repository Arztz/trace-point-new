package signoz

import (
	"fmt"
	"time"
)

// BuildTraceQuery returns a ClickHouse SQL query for fetching traces
// filtered by namespace and deployment during a specific time window.
func BuildTraceQuery(namespace, deployment string, start, end time.Time, limit int) string {
	startNano := start.UnixNano()
	endNano := end.UnixNano()

	deploymentFilter := ""
	if deployment != "" {
		deploymentFilter = fmt.Sprintf(`AND resources_string['k8s.deployment.name'] = '%s'`, deployment)
	}

	return fmt.Sprintf(`SELECT
    traceID,
    name as spanName,
    serviceName,
    toUnixTimestamp(timestamp) as startTime,
    CAST(duration_nano, 'UInt64') as duration,
    attributes_string['http.route'] AS httpUrl,
    attributes_string['http.method'] AS httpMethod,
    toInt32OrZero(attributes_string['http.status_code']) AS statusCode,
    resources_string['k8s.deployment.name'] AS deploymentName,
    resources_string['k8s.namespace.name'] AS namespace,
    resources_string['span.kind'] AS kind
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= toDateTime64(%d / 1000000000, 9)
  AND timestamp <= toDateTime64(%d / 1000000000, 9)
  AND resources_string['k8s.namespace.name'] = '%s'
  AND http_method != ''
  %s
ORDER BY timestamp DESC
LIMIT %d`,
		startNano, endNano, namespace, deploymentFilter, limit)
}

// BuildRouteAggregationQuery returns a ClickHouse SQL query for aggregating
// trace data by route during a spike window.
func BuildRouteAggregationQuery(namespace, deployment string, start, end time.Time) string {
	startNano := start.UnixNano()
	endNano := end.UnixNano()

	deploymentFilter := ""
	if deployment != "" {
		deploymentFilter = fmt.Sprintf(`AND resources_string['k8s.deployment.name'] = '%s'`, deployment)
	}

	return fmt.Sprintf(`SELECT
    resources_string['http.url'] as route,
    count() as trace_count,
    avg(CAST(duration_nano, 'UInt64')) / 1000000 as avg_duration_ms,
    sum(CAST(duration_nano, 'UInt64')) / 1000000 as total_duration_ms,
    countIf(toInt32OrZero(resources_string['http.status_code']) >= 400) as error_count
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= toDateTime64(%d / 1000000000, 9)
  AND timestamp <= toDateTime64(%d / 1000000000, 9)
  AND resources_string['k8s.namespace.name'] = '%s'
  %s
  AND resources_string['http.url'] != ''
GROUP BY route
ORDER BY total_duration_ms DESC
LIMIT 20`,
		startNano, endNano, namespace, deploymentFilter)
}
