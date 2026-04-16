import { NavLink, Outlet } from 'react-router-dom';
import DatasourceSelector from './DatasourceSelector';

const navItems = [
  { path: '/', label: 'Dashboard' },
  { path: '/spikes', label: 'Spike Events' },
  { path: '/explorer', label: 'Spike Explorer' },
  { path: '/gravity', label: 'Gravity Scores' },
];

export default function Layout() {
  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar - Dark navigation */}
      <aside className="w-64 flex-shrink-0 flex flex-col"
        style={{ background: '#0d0d0d' }}>
        {/* Logo */}
        <div className="p-5 flex items-center gap-3" style={{ borderBottom: '1px solid #333' }}>
          <div className="w-8 h-8 flex items-center justify-center text-xs font-black tracking-tight"
            style={{ 
              background: '#1c69d4',
              color: '#ffffff'
            }}>
            TP
          </div>
          <div>
            <h1 className="text-sm font-black tracking-tight" style={{ color: '#ffffff' }}>Trace-Point</h1>
            <p className="text-xs" style={{ color: '#666666' }}>v1.0.3</p>
          </div>
        </div>

        {/* Datasource Selector */}
        <div className="px-5 py-4" style={{ borderBottom: '1px solid #333' }}>
          <DatasourceSelector />
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-3 flex flex-col gap-0.5">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === '/'}
              className={({ isActive }) =>
                `px-4 py-3 text-sm transition-all duration-150 ${
                  isActive ? 'font-black tracking-tight' : 'font-normal'
                }`
              }
              style={({ isActive }) => ({
                background: isActive ? '#1c69d4' : 'transparent',
                color: isActive ? '#ffffff' : '#a0a0a0',
              })}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        {/* Status indicator */}
        <div className="p-5" style={{ borderTop: '1px solid #333' }}>
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 pulse" style={{ background: '#22c55e' }} />
            <span className="text-xs" style={{ color: '#666666' }}>Monitoring active</span>
          </div>
        </div>
      </aside>

      {/* Main content - Dark background */}
      <main className="flex-1 overflow-auto" style={{ background: '#0d0d0d' }}>
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
