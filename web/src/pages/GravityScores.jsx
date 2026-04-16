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
          <h1 className="page-header h1">Resource Gravity Scores</h1>
          <p className="page-header p">
            Identify services needing architectural refactoring
          </p>
        </div>
        <button onClick={handleExport} className="btn btn-primary">
          Export JSON
        </button>
      </div>

      {/* Score legend */}
      <div className="glass-card p-4 mb-6">
        <div className="flex items-center gap-6 text-xs">
          <span style={{ color: '#666666' }}>Score Range:</span>
          <span className="flex items-center gap-1.5">
            <span className="w-2 h-2" style={{ background: '#ef4444' }} />
            6+ High Impact (priority refactoring)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2 h-2" style={{ background: '#f59e0b' }} />
            3-6 Medium (consider optimization)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2 h-2" style={{ background: '#22c55e' }} />
            0-3 Low (monitor)
          </span>
        </div>
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <div key={i} className="skeleton" style={{ height: '64px' }} />)}
        </div>
      )}

      {error && (
        <div className="glass-card p-6 text-center">
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
                    <tr key={i}>
                      <td className="font-medium" style={{ color: '#ffffff' }}>{score.service_name}</td>
                      <td className="mono text-xs">{score.route_name}</td>
                      <td className="mono">{score.spike_count}</td>
                      <td className="mono" style={{ color: '#1c69d4' }}>{formatPercent(score.max_cpu_percent)}</td>
                      <td className="mono" style={{ color: '#a855f7' }}>{formatPercent(score.max_ram_percent)}</td>
                      <td className="mono">{formatPercent(score.average_cpu_percent)}</td>
                      <td className="mono">{formatPercent(score.average_ram_percent)}</td>
                      <td>
                        <div className="flex items-center gap-2">
                          <div className="w-16 h-1.5" style={{ background: '#333333' }}>
                            <div className="h-full"
                              style={{
                                width: `${Math.min(score.resource_gravity_score * 10, 100)}%`,
                                background: impact === 'high' ? '#ef4444' :
                                  impact === 'medium' ? '#f59e0b' : '#22c55e',
                              }} />
                          </div>
                          <span className="mono font-semibold text-xs">{score.resource_gravity_score.toFixed(1)}</span>
                        </div>
                      </td>
                      <td>
                        {score.is_job_like && <span className="badge badge-medium">Suspected Job</span>}
                        {score.tags?.filter(t => t !== '[Suspected Job]').map((tag, j) => (
                          <span key={j} className="badge badge-low ml-1">{tag}</span>
                        ))}
                      </td>
                    </tr>
                  );
                })}
                {(!data.scores || data.scores.length === 0) && (
                  <tr>
                    <td colSpan="9" className="text-center py-12" style={{ color: '#666666' }}>
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
