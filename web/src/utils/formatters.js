export function formatPercent(value) {
  if (value === undefined || value === null) return '—';
  return `${value.toFixed(1)}%`;
}

export function formatTimestamp(ts) {
  if (!ts) return '—';
  const d = new Date(ts);
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

export function formatShortTime(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  return d.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatDuration(ms) {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

export function getSeverityColor(severity) {
  switch (severity) {
    case 'critical': return '#ef4444';
    case 'medium': return '#d97706';
    case 'low': return '#16a34a';
    default: return '#757575';
  }
}

export function getSeverityClass(severity) {
  switch (severity) {
    case 'critical': return 'badge-critical';
    case 'medium': return 'badge-medium';
    case 'low': return 'badge-low';
    default: return '';
  }
}

export const DEPLOYMENT_COLORS = [
  '#1c69d4', '#a855f7', '#d97706', '#16a34a', '#ef4444',
  '#0891b2', '#c026d3', '#ca8a04', '#059669', '#dc2626',
  '#0284c7', '#9333ea', '#fbbf24', '#22c55e', '#b91c1c',
  '#0369a1', '#7c3aed', '#a16207', '#15803d', '#991b1b',
];

export function getDeploymentColor(index) {
  return DEPLOYMENT_COLORS[index % DEPLOYMENT_COLORS.length];
}

export function classifyImpact(score) {
  if (score >= 6) return 'high';
  if (score >= 3) return 'medium';
  return 'low';
}
