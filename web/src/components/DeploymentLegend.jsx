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
            className="flex items-center gap-2 px-3 py-1.5 text-xs transition-all duration-200 cursor-pointer rounded-full"
            style={{
              background: isActive ? 'rgba(59, 130, 246, 0.15)' : 'rgba(255, 255, 255, 0.03)',
              border: `1px solid ${isActive ? color + '60' : '#333333'}`,
              opacity: isActive ? 1 : 0.4,
              color: isActive ? '#ffffff' : '#8e8e93',
              fontWeight: isActive ? 500 : 400,
            }}
          >
            <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: color, boxShadow: isActive ? `0 0 6px ${color}` : 'none' }} />
            <span className="truncate max-w-32">{name}</span>
          </button>
        );
      })}
    </div>
  );
}