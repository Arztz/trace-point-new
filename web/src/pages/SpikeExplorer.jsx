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
    <div className="fade-in">
      <div className="mb-8">
        <h1 className="page-header h1 text-display">Spike Explorer</h1>
        <p className="page-header p">
          Historical spike analysis with sliding window algorithm
        </p>
      </div>

      <form onSubmit={handleSubmit} className="glass-card p-6 mb-6">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-5">
          <div>
            <label className="text-xs block mb-2 font-medium tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>START</label>
            <input type="datetime-local" className="input w-full"
              value={params.start} onChange={(e) => setParams({ ...params, start: e.target.value })} />
          </div>
          <div>
            <label className="text-xs block mb-2 font-medium tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>END</label>
            <input type="datetime-local" className="input w-full"
              value={params.end} onChange={(e) => setParams({ ...params, end: e.target.value })} />
          </div>
          <div>
            <label className="text-xs block mb-2 font-medium tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>WINDOW</label>
            <select className="select w-full"
              value={params.window} onChange={(e) => setParams({ ...params, window: e.target.value })}>
              <option value="5m">5 minutes</option>
              <option value="15m">15 minutes</option>
              <option value="30m">30 minutes</option>
              <option value="1h">1 hour</option>
            </select>
          </div>
          <div>
            <label className="text-xs block mb-2 font-medium tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>NAMESPACE</label>
            <input type="text" className="input w-full" placeholder="All"
              value={params.namespace} onChange={(e) => setParams({ ...params, namespace: e.target.value })} />
          </div>
          <div>
            <label className="text-xs block mb-2 font-medium tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>THRESHOLD %</label>
            <input type="number" className="input w-full"
              value={params.threshold} onChange={(e) => setParams({ ...params, threshold: Number(e.target.value) })} />
          </div>
          <div className="flex items-end">
            <button type="submit" className="btn btn-primary w-full">Analyze</button>
          </div>
        </div>
      </form>

      {isLoading && (
        <div className="glass-card p-10 text-center">
          <div className="inline-block w-8 h-8 border-2 border-t-transparent mb-4 rounded-full animate-spin"
            style={{ borderColor: '#3b82f6', borderTopColor: 'transparent', animation: 'spin 1s linear infinite' }} />
          <p className="text-sm" style={{ color: '#8e8e93' }}>Analyzing spike patterns...</p>
        </div>
      )}

      {error && (
        <div className="glass-card p-6 text-center">
          <p className="text-sm" style={{ color: '#ef4444' }}>Analysis failed: {error.message}</p>
        </div>
      )}

      {data && !isLoading && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-5 mb-6">
            <StatCard label="Total Spikes" value={data.summary?.total_spikes || 0} />
            <StatCard label="Deployments" value={data.summary?.analyzed_deployments || 0} />
            <StatCard label="Time Range" value={`${(data.summary?.time_range_hours || 0).toFixed(1)}h`} />
            <StatCard label="CPU Spikes" value={data.summary?.spikes_by_type?.cpu || 0} color="#3b82f6" />
          </div>

          <div className="glass-card overflow-hidden">
            <div className="p-5" style={{ borderBottom: '1px solid #2a2a2a' }}>
              <h3 className="text-sm font-semibold tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>
                DETECTED SPIKES ({data.spikes?.length || 0})
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
                  {data.spikes?.map((spike, index) => (
                    <tr key={spike.id} style={{ animationDelay: `${index * 25}ms` }}>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatTimestamp(spike.timestamp)}</td>
                      <td className="font-semibold" style={{ color: '#ffffff' }}>{spike.deployment_name}</td>
                      <td><span className="badge badge-medium">{spike.type}</span></td>
                      <td className="mono" style={{ color: '#3b82f6' }}>{formatPercent(spike.cpu_percent)}</td>
                      <td className="mono" style={{ color: '#a855f7' }}>{formatPercent(spike.ram_percent)}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatPercent(spike.moving_average_cpu)}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatPercent(spike.moving_average_ram)}</td>
                      <td className="mono font-semibold" style={{ color: '#ffffff' }}>{formatPercent(spike.deviation_percent)}</td>
                      <td><span className={`badge ${getSeverityClass(spike.severity)}`}>{spike.severity}</span></td>
                    </tr>
                  ))}
                  {(!data.spikes || data.spikes.length === 0) && (
                    <tr>
                      <td colSpan="9" className="text-center py-16" style={{ color: '#5f5f5f' }}>
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
    <div className="card-elevated p-5 text-center">
      <p className="text-xs mb-2 font-medium tracking-wide" style={{ color: '#8e8e93', letterSpacing: '0.5px' }}>{label}</p>
      <p className="text-2xl font-mono font-bold" style={{ color: color || '#ffffff', lineHeight: 1.10, fontFamily: 'JetBrains Mono, monospace' }}>{value}</p>
    </div>
  );
}