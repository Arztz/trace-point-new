import { useMemo, Fragment } from 'react';
import {
  ComposedChart, Line, Area, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, ReferenceLine,
} from 'recharts';
import { formatShortTime, getDeploymentColor, formatPercent } from '../utils/formatters';

export default function TimelineChart({ metrics = [], spikeMarkers = [], highlighted, deployments = [] }) {
  const { chartData, deploymentNames, yDomain } = useMemo(() => {
    if (!metrics.length) return { chartData: [], deploymentNames: [], yDomain: [0, 100] };

    const filteredMetrics = highlighted
      ? metrics.filter((m) => m.deployment_name === highlighted)
      : metrics;

    const nameSet = new Set();
    const timeMap = new Map();

    filteredMetrics.forEach((m) => {
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

    let maxVal = 0;
    filteredMetrics.forEach((m) => {
      maxVal = Math.max(maxVal, m.cpu_percent || 0, m.ram_percent || 0);
    });

    const roundedMax = maxVal <= 10 ? 20 : maxVal <= 50 ? 60 : maxVal <= 100 ? 120 : Math.ceil(maxVal / 50) * 50;
    const domain = [0, Math.max(roundedMax * 1.1, 100)];

    return { chartData: sorted, deploymentNames: Array.from(nameSet), yDomain: domain };
  }, [metrics, highlighted]);

  if (!chartData.length) {
    return (
      <div className="flex items-center justify-center glass-card" style={{ height: '400px', borderRadius: '20px' }}>
        <p style={{ color: '#8e8e93' }}>No metrics data available</p>
      </div>
    );
  }

  const CustomTooltip = ({ active, payload, label }) => {
    if (!active || !payload?.length || !highlighted) return null;

    const filtered = payload.filter((entry) => entry.dataKey.startsWith(`${highlighted}_`));
    if (!filtered.length) return null;

    return (
      <div className="p-4" style={{
        background: '#222222',
        border: '1px solid #333333',
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
        borderRadius: '12px',
        maxWidth: '320px',
      }}>
        <p className="font-semibold mb-2" style={{ color: '#ffffff', fontFamily: 'Outfit, sans-serif' }}>
          {formatShortTime(label)}
        </p>
        {filtered.map((entry, i) => {
          const type = entry.dataKey.endsWith('_cpu') ? 'CPU' : 'RAM';
          return (
            <div key={i} className="flex justify-between gap-6 py-1">
              <span style={{ color: entry.color }}>{highlighted} ({type})</span>
              <span className="font-mono" style={{ color: '#ffffff', fontFamily: 'JetBrains Mono, monospace' }}>
                {formatPercent(entry.value)}
              </span>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="glass-card p-5">
      <ResponsiveContainer width="100%" height={400}>
        <ComposedChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#2a2a2a" />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatShortTime}
            stroke="#8e8e93"
            fontSize={11}
            tickLine={false}
          />
          <YAxis
            stroke="#8e8e93"
            fontSize={11}
            tickLine={false}
            tickFormatter={(v) => `${v.toFixed(0)}%`}
            domain={yDomain}
          />
          {highlighted && <Tooltip content={<CustomTooltip />} />}

          {yDomain[1] >= 100 && (
            <ReferenceLine y={100} stroke="#ef4444" strokeDasharray="8 4" strokeOpacity={0.4} />
          )}

          {spikeMarkers.map((spike, i) => (
            <ReferenceLine
              key={i}
              x={new Date(spike.timestamp).getTime()}
              stroke="#ef4444"
              strokeDasharray="4 4"
              strokeOpacity={0.5}
            />
          ))}

          {deploymentNames.map((name, index) => {
            const color = getDeploymentColor(index);

            return (
              <Fragment key={name}>
                <Line
                  key={`${name}_cpu`}
                  type="monotone"
                  dataKey={`${name}_cpu`}
                  stroke={color}
                  strokeWidth={2.5}
                  dot={false}
                  connectNulls
                  name={`${name} CPU`}
                  strokeLinecap="round"
                />
                <Area
                  key={`${name}_ram`}
                  type="monotone"
                  dataKey={`${name}_ram`}
                  stroke={color}
                  strokeWidth={1.5}
                  strokeDasharray="5 3"
                  strokeOpacity={0.6}
                  fill={color}
                  fillOpacity={0.08}
                  dot={false}
                  connectNulls
                  name={`${name} RAM`}
                  strokeLinecap="round"
                />
              </Fragment>
            );
          })}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}