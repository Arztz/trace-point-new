const API_BASE = '/api/v1';

async function fetchJSON(url, options = {}) {
  const activeDatasource = localStorage.getItem('activeDatasource');
  let finalUrl = url;
  if (activeDatasource && !url.includes('datasource=')) {
    const separator = url.includes('?') ? '&' : '?';
    finalUrl = `${url}${separator}datasource=${encodeURIComponent(activeDatasource)}`;
  }

  const response = await fetch(`${API_BASE}${finalUrl}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }
  return response.json();
}

export const api = {
  // Datasources
  getDatasources: () => fetchJSON('/datasources'),

  // Timeline
  getTimeline: (timeRange = '1h', deploymentName = '') => {
    const params = new URLSearchParams({ time_range: timeRange });
    if (deploymentName) params.set('deployment_name', deploymentName);
    return fetchJSON(`/timeline?${params}`);
  },

  // Spikes
  getSpikes: (params = {}) => {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value) searchParams.set(key, value);
    });
    return fetchJSON(`/spikes?${searchParams}`);
  },

  getSpike: (id) => fetchJSON(`/spikes/${id}`),
  getSpikeDetails: (id) => fetchJSON(`/spikes/${id}/details`),
  retrySpikeCorrelation: (id) => fetchJSON(`/spikes/${id}/retry`, { method: 'POST' }),

  // Analysis
  analyzeSpikes: (params = {}) => {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== '') searchParams.set(key, String(value));
    });
    return fetchJSON(`/spikes/analyze?${searchParams}`);
  },

  // Export
  exportSpikes: () => fetchJSON('/export'),
  exportRefactoring: () => fetchJSON('/export/refactoring'),

  // Config & Scores
  getConfig: () => fetchJSON('/config'),
  getGravityScores: () => fetchJSON('/gravity-scores'),
};
