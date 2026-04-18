import { useState } from 'react';
import { useSpikes, useRetrySpike } from '../hooks/useData';
import { formatTimestamp, formatPercent } from '../utils/formatters';

export default function SpikeList() {
  const [sort, setSort] = useState('time');
  const [order, setOrder] = useState('desc');
  const [selectedSpike, setSelectedSpike] = useState(null);

  const { data, isLoading } = useSpikes({ sort, order, limit: 50 });
  const retryMutation = useRetrySpike();

  const handleRetry = () => {
    if (!selectedSpike) return;
    retryMutation.mutate(selectedSpike.id, {
      onSuccess: (updatedSpike) => {
        setSelectedSpike(updatedSpike);
      }
    });
  };

  const handleSort = (col) => {
    if (sort === col) {
      setOrder(order === 'desc' ? 'asc' : 'desc');
    } else {
      setSort(col);
      setOrder('desc');
    }
  };

  const SortIcon = ({ col }) => {
    if (sort !== col) return <span className="opacity-40 ml-1.5">↕</span>;
    return <span className="ml-1.5">{order === 'desc' ? '↓' : '↑'}</span>;
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="page-header h1 text-display">Spike Events</h1>
          <p className="page-header p">
            {data?.total || 0} events detected
          </p>
        </div>
      </div>

      <div className="glass-card overflow-hidden">
        {isLoading ? (
          <div className="p-8 space-y-3">
            {[1, 2, 3, 4, 5].map((i) => <div key={i} className="skeleton" style={{ height: '56px', borderRadius: '12px' }} />)}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th onClick={() => handleSort('time')}>Timestamp <SortIcon col="time" /></th>
                  <th onClick={() => handleSort('deployment')}>Deployment <SortIcon col="deployment" /></th>
                  <th>Namespace</th>
                  <th onClick={() => handleSort('cpu')}>CPU <SortIcon col="cpu" /></th>
                  <th onClick={() => handleSort('ram')}>RAM <SortIcon col="ram" /></th>
                  <th>Moving Avg</th>
                  <th>Route</th>
                  <th>Culprit</th>
                  <th>Alert</th>
                </tr>
              </thead>
              <tbody>
                {data?.spikes?.map((spike, index) => {
                  const deviation = spike.moving_average_percent > 0
                    ? ((spike.cpu_usage_percent - spike.moving_average_percent) / spike.moving_average_percent) * 100
                    : 0;
                  const severity = deviation > 200 ? 'critical' : deviation > 100 ? 'medium' : 'low';

                  return (
                    <tr key={spike.id} onClick={() => setSelectedSpike(spike)} className="cursor-pointer"
                      style={{ animationDelay: `${index * 30}ms` }}>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatTimestamp(spike.timestamp)}</td>
                      <td className="font-semibold" style={{ color: '#ffffff' }}>{spike.deployment_name}</td>
                      <td style={{ color: '#8e8e93' }}>{spike.namespace}</td>
                      <td className="mono" style={{ color: '#3b82f6' }}>{formatPercent(spike.cpu_usage_percent)}</td>
                      <td className="mono" style={{ color: '#a855f7' }}>{formatPercent(spike.ram_usage_percent)}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatPercent(spike.moving_average_percent)}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{spike.route_name || '—'}</td>
                      <td className="mono text-xs truncate max-w-36" style={{ color: '#8e8e93' }}>{spike.culprit_function || '—'}</td>
                      <td>
                        {spike.alert_sent
                          ? <span className="badge badge-low">Sent</span>
                          : <span className="text-xs" style={{ color: '#5f5f5f' }}>—</span>
                        }
                      </td>
                    </tr>
                  );
                })}
                {(!data?.spikes || data.spikes.length === 0) && (
                  <tr>
                    <td colSpan="9" className="text-center py-16" style={{ color: '#5f5f5f' }}>
                      No spike events recorded yet
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {selectedSpike && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.85)' }}
          onClick={() => setSelectedSpike(null)}>
          <div className="glass-card p-6 max-w-lg w-full fade-in" onClick={(e) => e.stopPropagation()}
            style={{ border: '1px solid #333333', borderRadius: '20px' }}>
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-lg font-semibold text-display">Spike Details</h2>
              <button onClick={() => setSelectedSpike(null)} className="text-xl cursor-pointer w-8 h-8 flex items-center justify-center rounded-full"
                style={{ color: '#8e8e93', background: 'rgba(255,255,255,0.05)', border: 'none' }}>✕</button>
            </div>
            <div className="space-y-0 text-sm">
              <Row label="Deployment" value={selectedSpike.deployment_name} />
              <Row label="Namespace" value={selectedSpike.namespace} />
              <Row label="Timestamp" value={formatTimestamp(selectedSpike.timestamp)} mono />
              <Row label="CPU Usage" value={formatPercent(selectedSpike.cpu_usage_percent)} color="#3b82f6" />
              <Row label="RAM Usage" value={formatPercent(selectedSpike.ram_usage_percent)} color="#a855f7" />
              <Row label="Moving Average" value={formatPercent(selectedSpike.moving_average_percent)} />
              <Row label="Threshold" value={formatPercent(selectedSpike.threshold_percent)} />
              {selectedSpike.route_name && <Row label="Route" value={selectedSpike.route_name} mono />}
              {selectedSpike.trace_id && <Row label="Trace ID" value={selectedSpike.trace_id} mono />}
              {selectedSpike.culprit_function && <Row label="Culprit Function" value={selectedSpike.culprit_function} mono />}
              {selectedSpike.culprit_file_path && <Row label="File Path" value={selectedSpike.culprit_file_path} mono />}
            </div>
            <div className="mt-6 flex justify-end">
              <button
                onClick={handleRetry}
                disabled={retryMutation.isPending}
                className="btn btn-primary flex items-center gap-2"
              >
                {retryMutation.isPending ? (
                  <>
                    <span className="inline-block" style={{ animation: 'spin 1s linear infinite' }}>↻</span>
                    Running...
                  </>
                ) : (
                  <>
                    <span>↻</span>
                    Retry RCA
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function Row({ label, value, mono, color }) {
  return (
    <div className="flex justify-between py-3" style={{ borderBottom: '1px solid #2a2a2a' }}>
      <span style={{ color: '#8e8e93' }}>{label}</span>
      <span className={mono ? 'font-mono text-xs' : ''} style={{ color: color || '#f0f0f0', fontFamily: mono ? 'JetBrains Mono, monospace' : 'inherit' }}>
        {value}
      </span>
    </div>
  );
}