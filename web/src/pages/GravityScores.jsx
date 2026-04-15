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
    <div className="space-y-6 fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold" style={{ color: 'var(--color-text-primary)' }}>Resource Gravity Scores</h1>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            Identify services needing architectural refactoring
          </p>
        </div>
        <button onClick={handleExport} className="btn btn-primary">
          📁 Export JSON
        </button>
      </div>

      {/* Score legend */}
      <div className="glass-card p-4">
        <div className="flex items-center gap-6 text-xs">
          <span style={{ color: 'var(--color-text-muted)' }}>Score Range:</span>
          <span className="flex items-center gap-1.5">
            <span className="w-2 h-2 rounded-full" style={{ background: 'var(--color-severity-critical)' }} />
            6+ High Impact (priority refactoring)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2 h-2 rounded-full" style={{ background: 'var(--color-severity-medium)' }} />
            3-6 Medium (consider optimization)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2 h-2 rounded-full" style={{ background: 'var(--color-severity-low)' }} />
            0-3 Low (monitor)
          </span>
        </div>
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <div key={i} className="skeleton h-16 rounded-xl" />)}
        </div>
      )}

      {error && (
        <div className="glass-card p-6 text-center">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>Failed to load scores: {error.message}</p>
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
                      <td className="font-medium" style={{ color: 'var(--color-text-primary)' }}>{score.service_name}</td>
                      <td className="mono text-xs">{score.route_name}</td>
                      <td className="mono">{score.spike_count}</td>
                      <td className="mono" style={{ color: 'var(--color-accent-cpu)' }}>{formatPercent(score.max_cpu_percent)}</td>
                      <td className="mono" style={{ color: 'var(--color-accent-ram)' }}>{formatPercent(score.max_ram_percent)}</td>
                      <td className="mono">{formatPercent(score.average_cpu_percent)}</td>
                      <td className="mono">{formatPercent(score.average_ram_percent)}</td>
                      <td>
                        <div className="flex items-center gap-2">
                          <div className="w-16 h-2 rounded-full overflow-hidden" style={{ background: 'var(--color-bg-primary)' }}>
                            <div className="h-full rounded-full"
                              style={{
                                width: `${Math.min(score.resource_gravity_score * 10, 100)}%`,
                                background: impact === 'high' ? 'var(--color-severity-critical)' :
                                  impact === 'medium' ? 'var(--color-severity-medium)' : 'var(--color-severity-low)',
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
                    <td colSpan="9" className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
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
