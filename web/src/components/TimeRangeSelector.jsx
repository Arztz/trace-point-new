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
            padding: '8px 14px',
            fontSize: '12px',
            fontWeight: value === r.value ? 600 : 500,
            background: value === r.value ? '#3b82f6' : 'rgba(255, 255, 255, 0.05)',
            color: value === r.value ? '#ffffff' : '#8e8e93',
            border: value === r.value ? '1px solid #3b82f6' : '1px solid #333333',
            borderRight: i < ranges.length - 1 ? 'none' : '1px solid #333333',
            borderRadius: i === 0 ? '9999px 0 0 9999px' : i === ranges.length - 1 ? '0 9999px 9999px 0' : '0',
            boxShadow: value === r.value ? '0 2px 8px rgba(59, 130, 246, 0.3)' : 'none',
          }}
        >
          {r.label}
        </button>
      ))}
    </div>
  );
}