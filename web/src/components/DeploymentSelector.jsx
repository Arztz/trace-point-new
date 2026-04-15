import { formatPercent } from '../utils/formatters';

export default function DeploymentSelector({ deployments = [], value, onChange }) {
  return (
    <select
      className="select"
      value={value || ''}
      onChange={(e) => onChange(e.target.value || null)}
    >
      <option value="">All Deployments</option>
      {deployments.map((d) => (
        <option key={`${d.namespace}/${d.name}`} value={d.name}>
          {d.name} — CPU: {formatPercent(d.current_cpu)} | RAM: {formatPercent(d.current_ram)}
        </option>
      ))}
    </select>
  );
}
