import { useState, useMemo } from 'react';
import { useTimeline } from '../hooks/useData';
import TimelineChart from '../components/TimelineChart';
import TimeRangeSelector from '../components/TimeRangeSelector';
import DeploymentSelector from '../components/DeploymentSelector';
import { formatPercent, getDeploymentColor } from '../utils/formatters';

export default function Dashboard() {
  const [timeRange, setTimeRange] = useState('1h');
  const [selectedDeployment, setSelectedDeployment] = useState(null);
  const [highlighted, setHighlighted] = useState(null);
  const [cpuFilter, setCpuFilter] = useState(null); // 'high' | 'low' | 'ok' | null
  const [ramFilter, setRamFilter] = useState(null); // 'high' | 'low' | 'ok' | null

  const { data, isLoading, error, refetch, isFetching } = useTimeline(timeRange, selectedDeployment);

  const deploymentNames = useMemo(() => {
    if (!data?.metrics) return [];
    return [...new Set(data.metrics.map((m) => m.deployment_name))];
  }, [data?.metrics]);

  // Map deployment name → color index for consistent coloring
  const deploymentColorMap = useMemo(() => {
    const map = {};
    deploymentNames.forEach((name, i) => {
      map[name] = getDeploymentColor(i);
    });
    return map;
  }, [deploymentNames]);

  // Sort summaries by deployment_name and apply badge filters
  const filteredSummary = useMemo(() => {
    if (!data?.summary) return [];
    let sorted = [...data.summary].sort((a, b) =>
      a.deployment_name.localeCompare(b.deployment_name)
    );
    if (cpuFilter) {
      sorted = sorted.filter((s) => s.cpu_classification === cpuFilter);
    }
    if (ramFilter) {
      sorted = sorted.filter((s) => s.ram_classification === ramFilter);
    }
    return sorted;
  }, [data?.summary, cpuFilter, ramFilter]);

  const handleCardClick = (deploymentName) => {
    setHighlighted((prev) => (prev === deploymentName ? null : deploymentName));
  };

  const handleCpuBadgeClick = (e, classification) => {
    e.stopPropagation(); // Don't trigger card highlight
    setCpuFilter((prev) => (prev === classification ? null : classification));
  };

  const handleRamBadgeClick = (e, classification) => {
    e.stopPropagation();
    setRamFilter((prev) => (prev === classification ? null : classification));
  };

  const hasActiveFilters = cpuFilter || ramFilter;

  return (
    <div className="space-y-6 fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold" style={{ color: 'var(--color-text-primary)' }}>
            Dashboard
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            Resource utilization timeline — deployment level
          </p>
        </div>
        <div className="flex items-center gap-3">
          <DeploymentSelector
            deployments={data?.available_deployments || []}
            value={selectedDeployment}
            onChange={setSelectedDeployment}
          />
          <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="btn btn-ghost flex items-center gap-1.5"
            style={{ padding: '6px 12px' }}
            title="Refresh data"
          >
            <span style={{
              display: 'inline-block',
              animation: isFetching ? 'spin 1s linear infinite' : 'none',
            }}>↻</span>
            {isFetching ? 'Loading...' : 'Refresh'}
          </button>
        </div>
      </div>

      {/* Loading state */}
      {isLoading && (
        <div className="space-y-4">
          <div className="skeleton h-[400px] rounded-xl" />
          <div className="grid grid-cols-4 gap-4">
            {[1, 2, 3, 4].map((i) => <div key={i} className="skeleton h-24 rounded-xl" />)}
          </div>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="glass-card p-6 text-center">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load timeline: {error.message}
          </p>
          <p className="text-xs mt-2" style={{ color: 'var(--color-text-muted)' }}>
            Make sure the backend is running on port 8088
          </p>
        </div>
      )}

      {/* Chart */}
      {data && !isLoading && (
        <>
          <TimelineChart
            metrics={data.metrics || []}
            spikeMarkers={data.spike_markers || []}
            highlighted={highlighted}
            deployments={filteredSummary.map(s => s.deployment_name)}
          />

          {/* Summary cards */}
          {data.summary && data.summary.length > 0 && (
            <>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                    DEPLOYMENTS ({filteredSummary.length}/{data.summary.length})
                  </span>
                  {hasActiveFilters && (
                    <button
                      onClick={() => { setCpuFilter(null); setRamFilter(null); }}
                      className="btn-ghost px-2 py-0.5 rounded text-xs cursor-pointer"
                      style={{ color: 'var(--color-accent-blue)', border: '1px solid var(--color-border-accent)' }}
                    >
                      ✕ Clear filters
                    </button>
                  )}
                </div>
                <div className="flex items-center gap-4 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                  <span className="flex items-center gap-1">
                    <span className="w-4 h-0.5 inline-block" style={{ background: 'var(--color-accent-cpu)' }} />
                    CPU (solid)
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-4 h-0.5 inline-block" style={{ background: 'var(--color-accent-ram)', borderTop: '1px dashed var(--color-accent-ram)' }} />
                    RAM (dashed)
                  </span>
                  {highlighted && (
                    <button
                      onClick={() => setHighlighted(null)}
                      className="btn-ghost px-2 py-0.5 rounded text-xs cursor-pointer"
                      style={{ color: 'var(--color-accent-blue)', border: '1px solid var(--color-border-accent)' }}
                    >
                      ✕ Unhighlight
                    </button>
                  )}
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {filteredSummary.map((s) => {
                  const color = deploymentColorMap[s.deployment_name] || '#6366f1';
                  const isActive = !highlighted || highlighted === s.deployment_name;
                  const isSelected = highlighted === s.deployment_name;

                  return (
                    <div
                      key={s.deployment_name}
                      className="glass-card p-4 fade-in cursor-pointer"
                      onClick={() => handleCardClick(s.deployment_name)}
                      style={{
                        borderLeft: `3px solid ${color}`,
                        opacity: isActive ? 1 : 0.4,
                        boxShadow: isSelected ? `0 0 16px ${color}20, inset 0 0 0 1px ${color}30` : undefined,
                        transition: 'all 0.25s ease',
                      }}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center gap-2 min-w-0">
                          <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: color }} />
                          <h3 className="text-sm font-medium truncate" style={{ color: 'var(--color-text-primary)' }}>
                            {s.deployment_name}
                          </h3>
                        </div>
                        <div className="flex items-center gap-1.5 flex-shrink-0">
                          <span
                            className={`badge cursor-pointer ${
                              s.cpu_classification === 'high' ? 'badge-critical' :
                              s.cpu_classification === 'low' ? 'badge-low' : 'badge-medium'
                            }`}
                            onClick={(e) => handleCpuBadgeClick(e, s.cpu_classification)}
                            title={`Filter by CPU: ${s.cpu_classification}`}
                            style={{
                              outline: cpuFilter === s.cpu_classification ? '2px solid currentColor' : 'none',
                              outlineOffset: '1px',
                            }}
                          >
                            CPU {s.cpu_classification === 'high' ? '⚠' :
                                 s.cpu_classification === 'low' ? '↓' : '✓'}
                          </span>
                          <span
                            className={`badge cursor-pointer ${
                              s.ram_classification === 'high' ? 'badge-critical' :
                              s.ram_classification === 'low' ? 'badge-low' : 'badge-medium'
                            }`}
                            onClick={(e) => handleRamBadgeClick(e, s.ram_classification)}
                            title={`Filter by RAM: ${s.ram_classification}`}
                            style={{
                              outline: ramFilter === s.ram_classification ? '2px solid currentColor' : 'none',
                              outlineOffset: '1px',
                            }}
                          >
                            RAM {s.ram_classification === 'high' ? '⚠' :
                                 s.ram_classification === 'low' ? '↓' : '✓'}
                          </span>
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <p className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>Avg CPU</p>
                          <p className="text-lg font-mono font-semibold" style={{ color: 'var(--color-accent-cpu)' }}>
                            {formatPercent(s.avg_cpu)}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>Max CPU</p>
                          <p className="text-lg font-mono font-semibold" style={{ color: s.max_cpu > 100 ? 'var(--color-danger)' : 'var(--color-text-primary)' }}>
                            {formatPercent(s.max_cpu)}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>Avg RAM</p>
                          <p className="text-lg font-mono font-semibold" style={{ color: 'var(--color-accent-ram)' }}>
                            {formatPercent(s.avg_ram)}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>Max RAM</p>
                          <p className="text-lg font-mono font-semibold" style={{ color: s.max_ram > 100 ? 'var(--color-danger)' : 'var(--color-text-primary)' }}>
                            {formatPercent(s.max_ram)}
                          </p>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}
