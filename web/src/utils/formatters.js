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
    case 'medium': return '#f59e0b';
    case 'low': return '#22c55e';
    default: return '#64748b';
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

// 20-color palette for deployment differentiation
export const DEPLOYMENT_COLORS = [
  '#06b6d4', '#a855f7', '#f59e0b', '#10b981', '#ef4444',
  '#6366f1', '#ec4899', '#14b8a6', '#f97316', '#8b5cf6',
  '#22d3ee', '#c084fc', '#fbbf24', '#34d399', '#f87171',
  '#818cf8', '#fb7185', '#2dd4bf', '#fb923c', '#a78bfa',
];

export function getDeploymentColor(index) {
  return DEPLOYMENT_COLORS[index % DEPLOYMENT_COLORS.length];
}

export function classifyImpact(score) {
  if (score >= 6) return 'high';
  if (score >= 3) return 'medium';
  return 'low';
}
