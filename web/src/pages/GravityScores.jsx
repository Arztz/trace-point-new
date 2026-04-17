import { useGravityScores } from '../hooks/useData';
import { formatPercent, classifyImpact } from '../utils/formatters';
import { api } from '../utils/api';

export default function GravityScores() {
  const { data, isLoading, error } = useGravityScores();

  const handleExport = async () => {
    try {
      const report = await api.exportRefactoring();
      const blob = new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `refactoring-report-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Export failed:', err);
    }
  };

  return (
    <div className="fade-in">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="page-header h1 text-display">Resource Gravity Scores</h1>
          <p className="page-header p">
            Identify services needing architectural refactoring
          </p>
        </div>
        <button onClick={handleExport} className="btn btn-primary">
          Export JSON
        </button>
      </div>

      <div className="card-elevated p-5 mb-6">
        <div className="flex items-center gap-8 text-xs">
          <span style={{ color: '#8e8e93' }}>Score Range:</span>
          <span className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: '#ef4444', boxShadow: '0 0 8px rgba(239, 68, 68, 0.4)' }} />
            6+ High Impact (priority refactoring)
          </span>
          <span className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: '#f59e0b', boxShadow: '0 0 8px rgba(245, 158, 11, 0.4)' }} />
            3-6 Medium (consider optimization)
          </span>
          <span className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: '#22c55e', boxShadow: '0 0 8px rgba(34, 197, 94, 0.4)' }} />
            0-3 Low (monitor)
          </span>
        </div>
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <div key={i} className="skeleton" style={{ height: '72px', borderRadius: '16px' }} />)}
        </div>
      )}

      {error && (
        <div className="card-elevated p-6 text-center">
          <p className="text-sm" style={{ color: '#ef4444' }}>Failed to load scores: {error.message}</p>
        </div>
      )}

      {data && !isLoading && (
        <div className="glass-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Service</th>
                  <th>Route</th>
                  <th>Spikes</th>
                  <th>Max CPU</th>
                  <th>Max RAM</th>
                  <th>Avg CPU</th>
                  <th>Avg RAM</th>
                  <th>Gravity Score</th>
                  <th>Tags</th>
                </tr>
              </thead>
              <tbody>
                {data.scores?.map((score, i) => {
                  const impact = classifyImpact(score.resource_gravity_score);
                  return (
                    <tr key={i} style={{ animationDelay: `${i * 40}ms` }}>
                      <td className="font-semibold" style={{ color: '#ffffff' }}>{score.service_name}</td>
                      <td className="mono text-xs" style={{ color: '#8e8e93' }}>{score.route_name}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{score.spike_count}</td>
                      <td className="mono" style={{ color: '#3b82f6' }}>{formatPercent(score.max_cpu_percent)}</td>
                      <td className="mono" style={{ color: '#a855f7' }}>{formatPercent(score.max_ram_percent)}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatPercent(score.average_cpu_percent)}</td>
                      <td className="mono" style={{ color: '#8e8e93' }}>{formatPercent(score.average_ram_percent)}</td>
                      <td>
                        <div className="flex items-center gap-3">
                          <div className="w-20 h-1.5 rounded-full" style={{ background: '#2a2a2a' }}>
                            <div className="h-full rounded-full"
                              style={{
                                width: `${Math.min(score.resource_gravity_score * 10, 100)}%`,
                                background: impact === 'high' ? '#ef4444' :
                                  impact === 'medium' ? '#f59e0b' : '#22c55e',
                                boxShadow: `0 0 8px ${impact === 'high' ? 'rgba(239, 68, 68, 0.5)' : impact === 'medium' ? 'rgba(245, 158, 11, 0.5)' : 'rgba(34, 197, 94, 0.5)'}`,
                              }} />
                          </div>
                          <span className="mono font-semibold text-sm" style={{ color: '#ffffff', fontFamily: 'JetBrains Mono, monospace' }}>{score.resource_gravity_score.toFixed(1)}</span>
                        </div>
                      </td>
                      <td>
                        {score.is_job_like && <span className="badge badge-medium">Suspected Job</span>}
                        {score.tags?.filter(t => t !== '[Suspected Job]').map((tag, j) => (
                          <span key={j} className="badge badge-low ml-1.5">{tag}</span>
                        ))}
                      </td>
                    </tr>
                  );
                })}
                {(!data.scores || data.scores.length === 0) && (
                  <tr>
                    <td colSpan="9" className="text-center py-16" style={{ color: '#5f5f5f' }}>
                      No gravity scores available yet — needs spike data
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}