import { getDeploymentColor } from '../utils/formatters';

export default function DeploymentLegend({ deployments = [], highlighted, onToggle }) {
  if (deployments.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2">
      {deployments.map((name, index) => {
        const color = getDeploymentColor(index);
        const isActive = !highlighted || highlighted === name;

        return (
          <button
            key={name}
            onClick={() => onToggle(highlighted === name ? null : name)}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs transition-all duration-200 cursor-pointer"
            style={{
              background: isActive ? 'rgba(99, 102, 241, 0.08)' : 'transparent',
              border: `1px solid ${isActive ? color + '40' : 'var(--color-border)'}`,
              opacity: isActive ? 1 : 0.4,
              color: isActive ? 'var(--color-text-primary)' : 'var(--color-text-muted)',
            }}
          >
            <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: color }} />
            <span className="truncate max-w-28">{name}</span>
          </button>
        );
      })}
    </div>
  );
}
