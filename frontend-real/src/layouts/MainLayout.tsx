import { Outlet, NavLink } from 'react-router'
import { LayoutDashboard, BookOpen, GraduationCap, FolderOpen, Inbox, Settings } from 'lucide-react'

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/dictionary', label: 'Dictionary', icon: BookOpen },
  { to: '/study', label: 'Study', icon: GraduationCap },
  { to: '/topics', label: 'Topics', icon: FolderOpen },
  { to: '/inbox', label: 'Inbox', icon: Inbox },
  { to: '/settings', label: 'Settings', icon: Settings },
]

function MainLayout() {
  return (
    <div className="min-h-screen bg-bg-page">
      {/* Desktop top navigation */}
      <nav className="hidden md:flex border-b border-border bg-bg-card sticky top-0 z-[var(--z-navigation)]">
        <div className="max-w-[var(--container-max)] mx-auto w-full flex items-center">
          <NavLink to="/dashboard" className="font-heading text-xl text-accent px-lg py-sm">
            MyEnglish
          </NavLink>
          <div className="flex">
            {navItems.map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  `flex items-center gap-xs px-[20px] py-3 text-[13px] font-medium border-b-2 transition-colors duration-[var(--duration-normal)] ${
                    isActive
                      ? 'text-accent border-accent'
                      : 'text-text-secondary border-transparent hover:text-text-primary'
                  }`
                }
              >
                <Icon className="size-4" />
                {label}
              </NavLink>
            ))}
          </div>
        </div>
      </nav>

      {/* Page content */}
      <main className="max-w-[var(--container-max)] mx-auto px-md py-lg">
        <Outlet />
      </main>

      {/* Mobile bottom tab bar */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 h-[var(--tab-bar-height)] bg-bg-card border-t border-border flex items-center justify-around z-[var(--z-navigation)]">
        {navItems.slice(0, 5).map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex flex-col items-center gap-0.5 text-[10px] font-medium py-1 ${
                isActive ? 'text-accent' : 'text-text-tertiary'
              }`
            }
          >
            <Icon className="size-5" />
            {label}
          </NavLink>
        ))}
      </nav>

      {/* Bottom padding for mobile nav */}
      <div className="md:hidden h-[var(--tab-bar-height)]" />
    </div>
  )
}

export default MainLayout
