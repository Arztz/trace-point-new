import { useState, useMemo, useRef } from 'react';
import { useTimeline } from '../hooks/useData';
import TimelineChart from '../components/TimelineChart';
import TimeRangeSelector from '../components/TimeRangeSelector';
import DeploymentSelector from '../components/DeploymentSelector';
import { formatPercent, getDeploymentColor } from '../utils/formatters';

function RecommendHint({ value, label, target = 80 }) {
  const [show, setShow] = useState(false);
  const ref = useRef(null);

  const multiplier = value / target;
  if (Math.abs(multiplier - 1) < 0.05) return null;

  const isLow = multiplier < 1;
  const color = isLow ? '#22c55e' : '#ef4444';
  const label2 = `x${multiplier.toFixed(2)}`;

  return (
    <span ref={ref} style={{ position: 'relative', display: 'inline-block' }}>
      <span
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        style={{
          fontSize: '11px',
          fontWeight: 700,
          color,
          cursor: 'default',
          padding: '2px 6px',
          background: isLow ? 'rgba(22, 197, 94, 0.15)' : 'rgba(239, 68, 68, 0.15)',
          border: `1px solid ${isLow ? 'rgba(22, 197, 94, 0.4)' : 'rgba(239, 68, 68, 0.4)'}`,
          letterSpacing: '0.01em',
          userSelect: 'none',
          lineHeight: 1.15,
        }}
      >
        {label2}
      </span>
      {show && (
        <span style={{
          position: 'absolute',
          bottom: '120%',
          left: '50%',
          transform: 'translateX(-50%)',
          background: '#262626',
          border: '1px solid #333333',
          padding: '8px 12px',
          fontSize: '11px',
          whiteSpace: 'nowrap',
          color: '#ffffff',
          boxShadow: '0 4px 12px rgba(0,0,0,0.5)',
          zIndex: 50,
          pointerEvents: 'none',
          lineHeight: 1.15,
        }}>
          <span style={{ color, fontWeight: 700 }}>{label2}</span> · {label}
          <br />
          <span style={{ color: '#666666' }}>
            {isLow
              ? `Overprovisioned — resize to ${label2} → util lands at ~${target}%`
              : `Underprovisioned — resize to ${label2} → util lands at ~${target}%`}
          </span>
        </span>
      )}
    </span>
  );
}

export default function Dashboard() {
  const [timeRange, setTimeRange] = useState('1h');
  const [selectedDeployment, setSelectedDeployment] = useState(null);
  const [highlighted, setHighlighted] = useState(null);
  const [cpuFilter, setCpuFilter] = useState(null);
  const [ramFilter, setRamFilter] = useState(null);

  const { data, isLoading, error, refetch, isFetching } = useTimeline(timeRange, selectedDeployment);

  const deploymentNames = useMemo(() => {
    if (!data?.metrics) return [];
    return [...new Set(data.metrics.map((m) => m.deployment_name))];
  }, [data?.metrics]);

  const deploymentColorMap = useMemo(() => {
    const map = {};
    deploymentNames.forEach((name, i) => {
      map[name] = getDeploymentColor(i);
    });
    return map;
  }, [deploymentNames]);

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
    e.stopPropagation();
    setCpuFilter((prev) => (prev === classification ? null : classification));
  };

  const handleRamBadgeClick = (e, classification) => {
    e.stopPropagation();
    setRamFilter((prev) => (prev === classification ? null : classification));
  };

  const hasActiveFilters = cpuFilter || ramFilter;

  return (
    <div className="fade-in">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="page-header h1">Dashboard</h1>
          <p className="page-header p">Resource utilization timeline — deployment level</p>
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
            style={{ padding: '8px 16px' }}
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
          <div className="skeleton" style={{ height: '400px' }} />
          <div className="grid grid-cols-4 gap-4">
            {[1, 2, 3, 4].map((i) => <div key={i} className="skeleton" style={{ height: '96px' }} />)}
          </div>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="glass-card p-6 text-center">
          <p className="text-sm" style={{ color: '#ef4444' }}>
            Failed to load timeline: {error.message}
          </p>
          <p className="text-xs mt-2" style={{ color: '#666666' }}>
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
              <div className="flex items-center justify-between mt-8 mb-4">
                <div className="flex items-center gap-3">
                  <span className="text-xs font-black tracking-wide" style={{ color: '#666666', letterSpacing: '0.5px' }}>
                    DEPLOYMENTS ({filteredSummary.length}/{data.summary.length})
                  </span>
                  {hasActiveFilters && (
                    <button
                      onClick={() => { setCpuFilter(null); setRamFilter(null); }}
                      className="btn-ghost px-2 py-1 text-xs cursor-pointer"
                      style={{ color: '#1c69d4', border: '1px solid #1c69d4', padding: '4px 8px' }}
                    >
                      Clear filters
                    </button>
                  )}
                </div>
                <div className="flex items-center gap-4 text-xs" style={{ color: '#666666' }}>
                  <span className="flex items-center gap-1.5">
                    <span className="w-4 h-0.5 inline-block" style={{ background: '#1c69d4' }} />
                    CPU (solid)
                  </span>
                  <span className="flex items-center gap-1.5">
                    <span className="w-4 h-0.5 inline-block" style={{ background: '#a855f7', borderTop: '1px dashed #a855f7' }} />
                    RAM (dashed)
                  </span>
                  {highlighted && (
                    <button
                      onClick={() => setHighlighted(null)}
                      className="btn-ghost px-2 py-1 text-xs cursor-pointer"
                      style={{ color: '#1c69d4', border: '1px solid #1c69d4', padding: '4px 8px' }}
                    >
                      Unhighlight
                    </button>
                  )}
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {filteredSummary.map((s) => {
                  const color = deploymentColorMap[s.deployment_name] || '#1c69d4';
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
                        transition: 'all 0.2s ease',
                      }}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center gap-2 min-w-0">
                          <span className="w-2 h-2" style={{ background: color }} />
                          <h3 className="text-sm font-medium truncate" style={{ color: '#ffffff' }}>
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
                          <p className="text-xs mb-1" style={{ color: '#666666', lineHeight: 1.15 }}>Avg CPU</p>
                          <div className="flex items-baseline gap-1.5">
                            <p className="text-lg font-mono font-semibold" style={{ color: '#1c69d4', lineHeight: 1.15 }}>
                              {formatPercent(s.avg_cpu)}
                            </p>
                            <RecommendHint value={s.avg_cpu} label="Recommend request" />
                          </div>
                        </div>
                        <div>
                          <p className="text-xs mb-1" style={{ color: '#666666', lineHeight: 1.15 }}>Max CPU</p>
                          <div className="flex items-baseline gap-1.5">
                            <p className="text-lg font-mono font-semibold" style={{ color: s.max_cpu > 100 ? '#ef4444' : '#ffffff', lineHeight: 1.15 }}>
                              {formatPercent(s.max_cpu)}
                            </p>
                            <RecommendHint value={s.max_cpu} label="Recommend limit" />
                          </div>
                        </div>
                        <div>
                          <p className="text-xs mb-1" style={{ color: '#666666', lineHeight: 1.15 }}>Avg RAM</p>
                          <div className="flex items-baseline gap-1.5">
                            <p className="text-lg font-mono font-semibold" style={{ color: '#a855f7', lineHeight: 1.15 }}>
                              {formatPercent(s.avg_ram)}
                            </p>
                            <RecommendHint value={s.avg_ram} label="Recommend request" />
                          </div>
                        </div>
                        <div>
                          <p className="text-xs mb-1" style={{ color: '#666666', lineHeight: 1.15 }}>Max RAM</p>
                          <div className="flex items-baseline gap-1.5">
                            <p className="text-lg font-mono font-semibold" style={{ color: s.max_ram > 100 ? '#ef4444' : '#ffffff', lineHeight: 1.15 }}>
                              {formatPercent(s.max_ram)}
                            </p>
                            <RecommendHint value={s.max_ram} label="Recommend limit" />
                          </div>
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
