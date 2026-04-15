import { useState } from 'react';
import { useAnalyzeSpikes } from '../hooks/useData';
import { formatTimestamp, formatPercent, getSeverityClass } from '../utils/formatters';

export default function SpikeExplorer() {
  const [params, setParams] = useState({
    start: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 16),
    end: new Date().toISOString().slice(0, 16),
    window: '30m',
    namespace: '',
    deployment: '',
    threshold: 50,
  });
  const [submitted, setSubmitted] = useState(null);

  const { data, isLoading, error } = useAnalyzeSpikes(
    submitted ? {
      start: new Date(submitted.start).toISOString(),
      end: new Date(submitted.end).toISOString(),
      window: submitted.window,
      namespace: submitted.namespace,
      deployment: submitted.deployment,
      threshold: submitted.threshold,
    } : null,
    !!submitted
  );

  const handleSubmit = (e) => {
    e.preventDefault();
    setSubmitted({ ...params });
  };

  return (
    <div className="space-y-6 fade-in">
      <div>
        <h1 className="text-2xl font-semibold" style={{ color: 'var(--color-text-primary)' }}>Spike Explorer</h1>
        <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
          Historical spike analysis with sliding window algorithm
        </p>
      </div>

      {/* Controls */}
      <form onSubmit={handleSubmit} className="glass-card p-5">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <div>
            <label className="text-xs block mb-1.5" style={{ color: 'var(--color-text-muted)' }}>Start</label>
            <input type="datetime-local" className="input w-full"
              value={params.start} onChange={(e) => setParams({ ...params, start: e.target.value })} />
          </div>
          <div>
            <label className="text-xs block mb-1.5" style={{ color: 'var(--color-text-muted)' }}>End</label>
            <input type="datetime-local" className="input w-full"
              value={params.end} onChange={(e) => setParams({ ...params, end: e.target.value })} />
          </div>
          <div>
            <label className="text-xs block mb-1.5" style={{ color: 'var(--color-text-muted)' }}>Window</label>
            <select className="select w-full"
              value={params.window} onChange={(e) => setParams({ ...params, window: e.target.value })}>
              <option value="5m">5 minutes</option>
              <option value="15m">15 minutes</option>
              <option value="30m">30 minutes</option>
              <option value="1h">1 hour</option>
            </select>
          </div>
          <div>
            <label className="text-xs block mb-1.5" style={{ color: 'var(--color-text-muted)' }}>Namespace</label>
            <input type="text" className="input w-full" placeholder="All"
              value={params.namespace} onChange={(e) => setParams({ ...params, namespace: e.target.value })} />
          </div>
          <div>
            <label className="text-xs block mb-1.5" style={{ color: 'var(--color-text-muted)' }}>Threshold %</label>
            <input type="number" className="input w-full"
              value={params.threshold} onChange={(e) => setParams({ ...params, threshold: Number(e.target.value) })} />
          </div>
          <div className="flex items-end">
            <button type="submit" className="btn btn-primary w-full">Analyze</button>
          </div>
        </div>
      </form>

      {/* Loading */}
      {isLoading && (
        <div className="glass-card p-8 text-center">
          <div className="inline-block w-6 h-6 border-2 border-t-transparent rounded-full animate-spin mb-3"
            style={{ borderColor: 'var(--color-accent-blue)', borderTopColor: 'transparent' }} />
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>Analyzing...</p>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="glass-card p-6 text-center">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>Analysis failed: {error.message}</p>
        </div>
      )}

      {/* Results */}
      {data && !isLoading && (
        <>
          {/* Summary */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard label="Total Spikes" value={data.summary?.total_spikes || 0} />
            <StatCard label="Deployments Analyzed" value={data.summary?.analyzed_deployments || 0} />
            <StatCard label="Time Range" value={`${(data.summary?.time_range_hours || 0).toFixed(1)}h`} />
            <StatCard label="CPU Spikes" value={data.summary?.spikes_by_type?.cpu || 0} color="var(--color-accent-cpu)" />
          </div>

          {/* Spikes table */}
          <div className="glass-card overflow-hidden">
            <div className="p-4" style={{ borderBottom: '1px solid var(--color-border)' }}>
              <h3 className="text-sm font-medium">
                Detected Spikes ({data.spikes?.length || 0})
              </h3>
            </div>
            <div className="overflow-x-auto">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Timestamp</th>
                    <th>Deployment</th>
                    <th>Type</th>
                    <th>CPU</th>
                    <th>RAM</th>
                    <th>Avg CPU</th>
                    <th>Avg RAM</th>
                    <th>Deviation</th>
                    <th>Severity</th>
                  </tr>
                </thead>
                <tbody>
                  {data.spikes?.map((spike) => (
                    <tr key={spike.id}>
                      <td className="mono">{formatTimestamp(spike.timestamp)}</td>
                      <td className="font-medium" style={{ color: 'var(--color-text-primary)' }}>{spike.deployment_name}</td>
                      <td><span className="badge badge-medium">{spike.type}</span></td>
                      <td className="mono" style={{ color: 'var(--color-accent-cpu)' }}>{formatPercent(spike.cpu_percent)}</td>
                      <td className="mono" style={{ color: 'var(--color-accent-ram)' }}>{formatPercent(spike.ram_percent)}</td>
                      <td className="mono">{formatPercent(spike.moving_average_cpu)}</td>
                      <td className="mono">{formatPercent(spike.moving_average_ram)}</td>
                      <td className="mono font-semibold">{formatPercent(spike.deviation_percent)}</td>
                      <td><span className={`badge ${getSeverityClass(spike.severity)}`}>{spike.severity}</span></td>
                    </tr>
                  ))}
                  {(!data.spikes || data.spikes.length === 0) && (
                    <tr>
                      <td colSpan="9" className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
                        No spikes found in the selected range
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function StatCard({ label, value, color }) {
  return (
    <div className="glass-card p-4 text-center">
      <p className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>{label}</p>
      <p className="text-2xl font-mono font-bold" style={{ color: color || 'var(--color-text-primary)' }}>{value}</p>
    </div>
  );
}
