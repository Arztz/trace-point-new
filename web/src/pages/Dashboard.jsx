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
          fontWeight: 600,
          color,
          cursor: 'default',
          padding: '4px 8px',
          background: isLow ? 'rgba(22, 197, 94, 0.15)' : 'rgba(239, 68, 68, 0.15)',
          border: `1px solid ${isLow ? 'rgba(22, 197, 94, 0.4)' : 'rgba(239, 68, 68, 0.4)'}`,
          borderRadius: '9999px',
          letterSpacing: '0.01em',
          userSelect: 'none',
          lineHeight: 1.50,
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
          background: '#222222',
          border: '1px solid #333333',
          padding: '10px 14px',
          fontSize: '12px',
          whiteSpace: 'nowrap',
          color: '#ffffff',
          boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
          borderRadius: '12px',
          zIndex: 50,
          pointerEvents: 'none',
          lineHeight: 1.50,
        }}>
          <span style={{ color, fontWeight: 600 }}>{label2}</span> · {label}
          <br />
          <span style={{ color: '#8e8e93' }}>
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
    if (highlighted) {
      sorted = sorted.filter((s) => s.deployment_name === highlighted);
    }
    if (cpuFilter) {
      sorted = sorted.filter((s) => s.cpu_classification === cpuFilter);
    }
    if (ramFilter) {
      sorted = sorted.filter((s) => s.ram_classification === ramFilter);
    }
    return sorted;
  }, [data?.summary, highlighted, cpuFilter, ramFilter]);

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
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="page-header h1 text-display">Dashboard</h1>
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

      {isLoading && (
        <div className="space-y-4">
          <div className="skeleton" style={{ height: '400px' }} />
          <div className="grid grid-cols-4 gap-5">
            {[1, 2, 3, 4].map((i) => <div key={i} className="skeleton" style={{ height: '160px', borderRadius: '16px' }} />)}
          </div>
        </div>
      )}

      {error && (
        <div className="card-elevated p-6 text-center">
          <p className="text-sm" style={{ color: '#ef4444' }}>
            Failed to load timeline: {error.message}
          </p>
          <p className="text-xs mt-2" style={{ color: '#8e8e93' }}>
            Make sure the backend is running on port 8088
          </p>
        </div>
      )}

      {data && !isLoading && (
        <>
          <TimelineChart
            metrics={data.metrics || []}
            spikeMarkers={data.spike_markers || []}
            highlighted={highlighted}
            deployments={filteredSummary.map(s => s.deployment_name)}
          />

          {data.summary && data.summary.length > 0 && (
            <>
              <div className="flex items-center justify-between mt-8 mb-5">
                <div className="flex items-center gap-3">
                  <span className="text-xs font-semibold tracking-widest" style={{ color: '#8e8e93', letterSpacing: '1px' }}>
                    DEPLOYMENTS ({filteredSummary.length}/{data.summary.length})
                  </span>
                  {hasActiveFilters && (
                    <button
                      onClick={() => { setCpuFilter(null); setRamFilter(null); }}
                      className="btn btn-ghost text-xs cursor-pointer"
                      style={{ color: '#60a5fa', border: '1px solid rgba(59, 130, 246, 0.4)', padding: '4px 12px' }}
                    >
                      Clear filters
                    </button>
                  )}
                </div>
                <div className="flex items-center gap-5 text-xs" style={{ color: '#8e8e93' }}>
                  <span className="flex items-center gap-2">
                    <span className="w-5 h-0.5 inline-block rounded-full" style={{ background: '#3b82f6' }} />
                    CPU (solid)
                  </span>
                  <span className="flex items-center gap-2">
                    <span className="w-5 h-0.5 inline-block rounded-full border-t-2 border-dashed" style={{ borderColor: '#a855f7', background: 'transparent' }} />
                    RAM (dashed)
                  </span>
                  {highlighted && (
                    <button
                      onClick={() => setHighlighted(null)}
                      className="btn btn-ghost text-xs cursor-pointer"
                      style={{ color: '#60a5fa', border: '1px solid rgba(59, 130, 246, 0.4)', padding: '4px 12px' }}
                    >
                      Unhighlight
                    </button>
                  )}
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
                {filteredSummary.map((s, index) => {
                  const color = deploymentColorMap[s.deployment_name] || '#3b82f6';
                  const isActive = !highlighted || highlighted === s.deployment_name;
                  const isSelected = highlighted === s.deployment_name;

                  return (
                    <div
                      key={s.deployment_name}
                      className="card-elevated p-5 fade-in cursor-pointer"
                      onClick={() => handleCardClick(s.deployment_name)}
                      style={{
                        borderLeft: `4px solid ${color}`,
                        opacity: isActive ? 1 : 0.5,
                        transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                        animationDelay: `${index * 50}ms`,
                      }}
                    >
                      <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center gap-3 min-w-0">
                          <span className="w-2.5 h-2.5 rounded-full" style={{ background: color, boxShadow: `0 0 8px ${color}` }} />
                          <h3 className="text-sm font-semibold truncate" style={{ color: '#ffffff' }}>
                            {s.deployment_name}
                          </h3>
                        </div>
                        <div className="flex items-center gap-2 flex-shrink-0">
                          <span
                            className={`badge cursor-pointer ${
                              s.cpu_classification === 'high' ? 'badge-critical' :
                              s.cpu_classification === 'low' ? 'badge-low' : 'badge-medium'
                            }`}
                            onClick={(e) => handleCpuBadgeClick(e, s.cpu_classification)}
                            title={`Filter by CPU: ${s.cpu_classification}`}
                            style={{
                              outline: cpuFilter === s.cpu_classification ? '2px solid currentColor' : 'none',
                              outlineOffset: '2px',
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
                              outlineOffset: '2px',
                            }}
                          >
                            RAM {s.ram_classification === 'high' ? '⚠' :
                                 s.ram_classification === 'low' ? '↓' : '✓'}
                          </span>
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <p className="text-xs mb-1.5" style={{ color: '#8e8e93', lineHeight: 1.50 }}>Avg CPU</p>
                          <div className="flex items-baseline gap-2">
                            <p className="text-xl font-mono font-semibold" style={{ color: '#3b82f6', lineHeight: 1.10, fontFamily: 'JetBrains Mono, monospace' }}>
                              {formatPercent(s.avg_cpu)}
                            </p>
                            <RecommendHint value={s.avg_cpu} label="Recommend request" />
                          </div>
                        </div>
                        <div>
                          <p className="text-xs mb-1.5" style={{ color: '#8e8e93', lineHeight: 1.50 }}>Max CPU</p>
                          <div className="flex items-baseline gap-2">
                            <p className="text-xl font-mono font-semibold" style={{ color: s.max_cpu > 100 ? '#ef4444' : '#ffffff', lineHeight: 1.10, fontFamily: 'JetBrains Mono, monospace' }}>
                              {formatPercent(s.max_cpu)}
                            </p>
                            <RecommendHint value={s.max_cpu} label="Recommend limit" />
                          </div>
                        </div>
                        <div>
                          <p className="text-xs mb-1.5" style={{ color: '#8e8e93', lineHeight: 1.50 }}>Avg RAM</p>
                          <div className="flex items-baseline gap-2">
                            <p className="text-xl font-mono font-semibold" style={{ color: '#a855f7', lineHeight: 1.10, fontFamily: 'JetBrains Mono, monospace' }}>
                              {formatPercent(s.avg_ram)}
                            </p>
                            <RecommendHint value={s.avg_ram} label="Recommend request" />
                          </div>
                        </div>
                        <div>
                          <p className="text-xs mb-1.5" style={{ color: '#8e8e93', lineHeight: 1.50 }}>Max RAM</p>
                          <div className="flex items-baseline gap-2">
                            <p className="text-xl font-mono font-semibold" style={{ color: s.max_ram > 100 ? '#ef4444' : '#ffffff', lineHeight: 1.10, fontFamily: 'JetBrains Mono, monospace' }}>
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