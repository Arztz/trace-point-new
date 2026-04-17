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
      <aside className="w-64 flex-shrink-0 flex flex-col"
        style={{ background: '#181e25' }}>
        <div className="p-6 flex items-center gap-3" style={{ borderBottom: '1px solid #2a2a2a' }}>
          <div className="w-10 h-10 flex items-center justify-center text-sm font-black tracking-tight rounded-xl"
            style={{
              background: 'linear-gradient(135deg, #1456f0, #3daeff)',
              color: '#ffffff',
              boxShadow: '0 4px 14px rgba(20, 86, 240, 0.4)',
            }}>
            TP
          </div>
          <div>
            <h1 className="text-base font-semibold tracking-tight" style={{ color: '#ffffff', fontFamily: 'Outfit, sans-serif' }}>Trace-Point</h1>
            <p className="text-xs" style={{ color: '#8e8e93' }}>v1.0.3</p>
          </div>
        </div>

        <div className="px-5 py-5" style={{ borderBottom: '1px solid #2a2a2a' }}>
          <DatasourceSelector />
        </div>

        <nav className="flex-1 p-4 flex flex-col gap-2">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === '/'}
              className={({ isActive }) =>
                `px-4 py-3 text-sm transition-all duration-200 rounded-xl ${
                  isActive ? 'font-semibold' : 'font-normal'
                }`
              }
              style={({ isActive }) => ({
                background: isActive ? 'rgba(59, 130, 246, 0.15)' : 'transparent',
                color: isActive ? '#60a5fa' : '#8e8e93',
                border: isActive ? '1px solid rgba(59, 130, 246, 0.3)' : '1px solid transparent',
              })}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="p-5" style={{ borderTop: '1px solid #2a2a2a' }}>
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full pulse" style={{ background: '#22c55e', boxShadow: '0 0 8px rgba(34, 197, 94, 0.5)' }} />
            <span className="text-xs" style={{ color: '#8e8e93' }}>Monitoring active</span>
          </div>
        </div>
      </aside>

      <main className="flex-1 overflow-auto" style={{ background: '#181e25' }}>
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}