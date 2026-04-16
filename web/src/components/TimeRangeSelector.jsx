export default function TimeRangeSelector({ value, onChange }) {
  const ranges = [
    { value: '1h', label: '1H' },
    { value: '6h', label: '6H' },
    { value: '12h', label: '12H' },
    { value: '1d', label: '1D' },
    { value: '3d', label: '3D' },
    { value: '5d', label: '5D' },
    { value: '7d', label: '7D' },
  ];

  return (
    <div className="flex gap-1 p-1 rounded-lg" style={{ background: 'var(--color-bg-primary)', border: '1px solid var(--color-border)' }}>
      {ranges.map((r) => (
        <button
          key={r.value}
          onClick={() => onChange(r.value)}
          className={`btn ${value === r.value ? 'btn-ghost active' : 'btn-ghost'}`}
          style={{ padding: '4px 12px', fontSize: '12px' }}
        >
          {r.label}
        </button>
      ))}
    </div>
  );
}
