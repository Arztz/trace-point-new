import { useEffect, useState } from 'react';
import { useDatasources } from '../hooks/useData';

export default function DatasourceSelector() {
  const { data, isLoading } = useDatasources();
  const [active, setActive] = useState(localStorage.getItem('activeDatasource') || '');

  useEffect(() => {
    if (!isLoading && data?.datasources?.length > 0) {
      if (!active || !data.datasources.find(d => d.id === active)) {
        const defaultDs = data.datasources[0].id;
        setActive(defaultDs);
        localStorage.setItem('activeDatasource', defaultDs);
        window.location.reload(); // Reload to refresh all queries with the new datasource
      }
    }
  }, [data, isLoading, active]);

  const handleChange = (e) => {
    const ds = e.target.value;
    setActive(ds);
    localStorage.setItem('activeDatasource', ds);
    window.location.reload();
  };

  if (isLoading) return <div className="text-xs text-gray-500 loading-pulse">Loading sources...</div>;
  if (!data?.datasources?.length) return null;

  return (
    <div className="flex items-center gap-2">
      <label className="text-xs text-gray-400 font-medium whitespace-nowrap">Datasource:</label>
      <select
        value={active}
        onChange={handleChange}
        className="text-xs bg-slate-800 border border-slate-700 rounded px-2 py-1 outline-none focus:border-indigo-500 transition-colors"
        style={{ color: 'var(--color-text-primary)' }}
      >
        {data.datasources.map(ds => (
          <option key={ds.id} value={ds.id}>{ds.name}</option>
        ))}
      </select>
    </div>
  );
}
