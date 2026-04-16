import { useMemo, Fragment } from 'react';
import {
  ComposedChart, Line, Area, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, ReferenceLine,
} from 'recharts';
import { formatShortTime, getDeploymentColor, formatPercent } from '../utils/formatters';

export default function TimelineChart({ metrics = [], spikeMarkers = [], highlighted, deployments = [] }) {
  // Transform data: pivot metrics into chart-friendly format
  const { chartData, deploymentNames } = useMemo(() => {
    if (!metrics.length) return { chartData: [], deploymentNames: [] };

    const nameSet = new Set();
    const timeMap = new Map();

    metrics.forEach((m) => {
      const ts = new Date(m.timestamp).getTime();
      nameSet.add(m.deployment_name);

      if (!timeMap.has(ts)) {
        timeMap.set(ts, { timestamp: ts });
      }
      const entry = timeMap.get(ts);
      entry[`${m.deployment_name}_cpu`] = m.cpu_percent;
      entry[`${m.deployment_name}_ram`] = m.ram_percent;
    });

    const sorted = Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp);
    return { chartData: sorted, deploymentNames: Array.from(nameSet) };
  }, [metrics]);

  if (!chartData.length) {
    return (
      <div className="flex items-center justify-center h-80 glass-card">
        <p style={{ color: 'var(--color-text-muted)' }}>No metrics data available</p>
      </div>
    );
  }

  const CustomTooltip = ({ active, payload, label }) => {
    if (!active || !payload?.length || !highlighted) return null;

    // Only show entries for the highlighted deployment
    const filtered = payload.filter((entry) => entry.dataKey.startsWith(`${highlighted}_`));
    if (!filtered.length) return null;

    return (
      <div className="p-3 rounded-lg text-xs" style={{
        background: 'rgba(15, 17, 23, 0.95)',
        border: '1px solid var(--color-border-accent)',
        backdropFilter: 'blur(8px)',
        maxWidth: '300px',
      }}>
        <p className="font-medium mb-2" style={{ color: 'var(--color-text-primary)' }}>
          {formatShortTime(label)}
        </p>
        {filtered.map((entry, i) => {
          const type = entry.dataKey.endsWith('_cpu') ? 'CPU' : 'RAM';
          return (
            <div key={i} className="flex justify-between gap-4 py-0.5">
              <span style={{ color: entry.color }}>{highlighted} ({type})</span>
              <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>
                {formatPercent(entry.value)}
              </span>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="glass-card p-4">
      <ResponsiveContainer width="100%" height={400}>
        <ComposedChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(99, 102, 241, 0.08)" />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatShortTime}
            stroke="var(--color-text-muted)"
            fontSize={11}
            tickLine={false}
          />
          <YAxis
            stroke="var(--color-text-muted)"
            fontSize={11}
            tickLine={false}
            tickFormatter={(v) => `${v.toFixed(0)}%`}
            domain={[0, 'auto']}
          />
          {highlighted && <Tooltip content={<CustomTooltip />} />}

          {/* Reference line at 100% */}
          <ReferenceLine y={100} stroke="#ef4444" strokeDasharray="8 4" strokeOpacity={0.3} />

          {/* Spike markers */}
          {spikeMarkers.map((spike, i) => (
            <ReferenceLine
              key={i}
              x={new Date(spike.timestamp).getTime()}
              stroke="#ef4444"
              strokeDasharray="4 4"
              strokeOpacity={0.5}
            />
          ))}

          {/* Deployment lines */}
          {deploymentNames.map((name, index) => {
            const color = getDeploymentColor(index);
            const isActive = !highlighted || highlighted === name;
            const opacity = isActive ? 1 : 0.1;

            return (
              <Fragment key={name}>
                <Line
                  key={`${name}_cpu`}
                  type="monotone"
                  dataKey={`${name}_cpu`}
                  stroke={color}
                  strokeWidth={isActive ? 2 : 1}
                  strokeOpacity={opacity}
                  dot={false}
                  connectNulls
                  name={`${name} CPU`}
                />
                <Area
                  key={`${name}_ram`}
                  type="monotone"
                  dataKey={`${name}_ram`}
                  stroke={color}
                  strokeWidth={1}
                  strokeDasharray="4 2"
                  strokeOpacity={opacity * 0.7}
                  fill={color}
                  fillOpacity={opacity * 0.05}
                  dot={false}
                  connectNulls
                  name={`${name} RAM`}
                />
              </Fragment>
            );
          })}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
