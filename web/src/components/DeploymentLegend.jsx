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
            className="flex items-center gap-1.5 px-2.5 py-1 text-xs transition-all duration-200 cursor-pointer"
            style={{
              background: isActive ? 'rgba(28, 105, 212, 0.15)' : 'transparent',
              border: `1px solid ${isActive ? color + '80' : '#333333'}`,
              opacity: isActive ? 1 : 0.4,
              color: isActive ? '#ffffff' : '#666666',
            }}
          >
            <span className="w-2 h-2 flex-shrink-0" style={{ background: color }} />
            <span className="truncate max-w-28">{name}</span>
          </button>
        );
      })}
    </div>
  );
}
