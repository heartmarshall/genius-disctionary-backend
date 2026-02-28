import { Outlet, NavLink, useNavigate } from 'react-router'
import { useAuth } from '@/providers/AuthProvider'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import {
  LayoutDashboard,
  BookOpen,
  GraduationCap,
  FolderOpen,
  Inbox,
  Settings,
  LogOut,
  User,
} from 'lucide-react'

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/dictionary', label: 'Dictionary', icon: BookOpen },
  { to: '/study', label: 'Study', icon: GraduationCap },
  { to: '/topics', label: 'Topics', icon: FolderOpen },
  { to: '/inbox', label: 'Inbox', icon: Inbox },
  { to: '/settings', label: 'Settings', icon: Settings },
]

function MainLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-bg-page">
      {/* Desktop top navigation */}
      <nav className="hidden md:flex border-b border-border bg-bg-card sticky top-0 z-[var(--z-navigation)]">
        <div className="max-w-[var(--container-max)] mx-auto w-full flex items-center">
          <NavLink to="/dashboard" className="font-heading text-xl text-accent px-lg py-sm">
            MyEnglish
          </NavLink>
          <div className="flex flex-1">
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

          {/* User dropdown */}
          <div className="px-md">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="gap-xs text-text-secondary">
                  <User className="size-4" />
                  <span className="max-w-[120px] truncate text-xs">
                    {user?.username || user?.email}
                  </span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel className="font-normal">
                  <div className="text-sm font-medium">{user?.name || user?.username}</div>
                  <div className="text-xs text-muted-foreground truncate">{user?.email}</div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={handleLogout}>
                  <LogOut className="size-4" />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
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
