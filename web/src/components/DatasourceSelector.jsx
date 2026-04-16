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
        window.location.reload();
      }
    }
  }, [data, isLoading, active]);

  const handleChange = (e) => {
    const ds = e.target.value;
    setActive(ds);
    localStorage.setItem('activeDatasource', ds);
    window.location.reload();
  };

  if (isLoading) return <div className="text-xs" style={{ color: '#666666' }}>Loading sources...</div>;
  if (!data?.datasources?.length) return null;

  return (
    <div className="flex items-center gap-2">
      <label className="text-xs whitespace-nowrap" style={{ color: '#666666', fontWeight: 400 }}>Datasource:</label>
      <select
        value={active}
        onChange={handleChange}
        className="select"
        style={{ 
          padding: '4px 24px 4px 8px', 
          fontSize: '11px',
          background: '#1a1a1a',
          color: '#ffffff',
          border: '1px solid #333',
        }}
      >
        {data.datasources.map(ds => (
          <option key={ds.id} value={ds.id} style={{ background: '#1a1a1a' }}>{ds.name}</option>
        ))}
      </select>
    </div>
  );
}
