import { NavLink, Outlet } from 'react-router-dom';

const navItems = [
  { path: '/', label: 'Dashboard', icon: '📊' },
  { path: '/spikes', label: 'Spike Events', icon: '⚡' },
  { path: '/explorer', label: 'Spike Explorer', icon: '🔍' },
  { path: '/gravity', label: 'Gravity Scores', icon: '🎯' },
];

export default function Layout() {
  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 flex-shrink-0 flex flex-col"
        style={{
          background: 'linear-gradient(180deg, #0d0f1a 0%, #141728 100%)',
          borderRight: '1px solid var(--color-border)'
        }}>
        {/* Logo */}
        <div className="p-5 flex items-center gap-3" style={{ borderBottom: '1px solid var(--color-border)' }}>
          <div className="w-8 h-8 rounded-lg flex items-center justify-center text-sm"
            style={{ background: 'linear-gradient(135deg, #6366f1, #8b5cf6)' }}>
            TP
          </div>
          <div>
            <h1 className="text-sm font-semibold" style={{ color: 'var(--color-text-primary)' }}>Trace-Point</h1>
            <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>v1.0.3</p>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-3 flex flex-col gap-1">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all duration-200 ${
                  isActive
                    ? 'font-medium'
                    : 'hover:bg-white/5'
                }`
              }
              style={({ isActive }) => ({
                background: isActive ? 'rgba(99, 102, 241, 0.12)' : undefined,
                color: isActive ? '#818cf8' : 'var(--color-text-secondary)',
                borderLeft: isActive ? '2px solid #6366f1' : '2px solid transparent',
              })}
            >
              <span>{item.icon}</span>
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>

        {/* Status indicator */}
        <div className="p-4" style={{ borderTop: '1px solid var(--color-border)' }}>
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full pulse" style={{ background: '#10b981' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>Monitoring active</span>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto" style={{ background: 'var(--color-bg-primary)' }}>
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
