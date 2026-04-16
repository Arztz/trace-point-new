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
    <div className="flex gap-0">
      {ranges.map((r, i) => (
        <button
          key={r.value}
          onClick={() => onChange(r.value)}
          className="btn"
          style={{ 
            padding: '6px 12px', 
            fontSize: '11px',
            fontWeight: value === r.value ? 700 : 400,
            background: value === r.value ? '#1c69d4' : 'transparent',
            color: value === r.value ? '#ffffff' : '#a0a0a0',
            border: value === r.value ? '1px solid #1c69d4' : '1px solid #333333',
            borderRight: i < ranges.length - 1 ? 'none' : '1px solid #333333',
          }}
        >
          {r.label}
        </button>
      ))}
    </div>
  );
}
