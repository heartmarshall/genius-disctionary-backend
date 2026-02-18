import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'

const navItems = [
  { to: '/catalog', label: 'Catalog' },
  { to: '/dictionary', label: 'Dictionary' },
  { to: '/study', label: 'Study' },
  { to: '/topics', label: 'Topics' },
  { to: '/inbox', label: 'Inbox' },
  { to: '/profile', label: 'Profile' },
  { to: '/explorer', label: 'API Explorer' },
]

export function Layout() {
  const { isAuthenticated, logout, token } = useAuth()

  return (
    <div className="flex h-screen">
      <nav className="w-56 bg-gray-900 text-gray-100 flex flex-col shrink-0">
        <div className="p-4 text-lg font-bold border-b border-gray-700">
          MyEnglish Test
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `block px-4 py-2 text-sm hover:bg-gray-800 ${isActive ? 'bg-gray-800 text-white font-medium' : 'text-gray-400'}`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </div>
        <div className="p-4 border-t border-gray-700 text-xs">
          {isAuthenticated ? (
            <div className="space-y-2">
              <div className="text-green-400">Authenticated</div>
              <div className="truncate text-gray-500" title={token ?? ''}>
                {token?.slice(0, 20)}...
              </div>
              <button onClick={logout} className="text-red-400 hover:text-red-300">
                Logout
              </button>
            </div>
          ) : (
            <NavLink to="/login" className="text-yellow-400 hover:text-yellow-300">
              Login
            </NavLink>
          )}
        </div>
      </nav>

      <main className="flex-1 overflow-y-auto bg-gray-50">
        <Outlet />
      </main>
    </div>
  )
}
