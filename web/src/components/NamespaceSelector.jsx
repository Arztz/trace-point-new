export default function NamespaceSelector({ deployments = [], value, onChange }) {
  // Extract unique namespaces from deployments
  const namespaces = [...new Set(deployments.map(d => d.namespace))].sort();
  
  return (
    <select
      className="select"
      value={value || ''}
      onChange={(e) => onChange(e.target.value || null)}
    >
      <option value="">All Namespace</option>
      {namespaces.map((ns) => (
        <option key={ns} value={ns}>
          {ns}
        </option>
      ))}
    </select>
  );
}