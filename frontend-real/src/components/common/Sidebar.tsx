import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  BookOpen,
  GraduationCap,
  Tags,
  Inbox,
  Settings,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/dictionary', label: 'Dictionary', icon: BookOpen },
  { to: '/study', label: 'Study', icon: GraduationCap },
  { to: '/topics', label: 'Topics', icon: Tags },
  { to: '/inbox', label: 'Inbox', icon: Inbox },
  { to: '/settings', label: 'Settings', icon: Settings },
]

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  onClose?: () => void
}

export function Sidebar({ collapsed, onToggle, onClose }: SidebarProps) {
  return (
    <aside
      className={cn(
        'flex h-full flex-col border-r border-border-default bg-surface-secondary transition-[width] duration-200',
        collapsed ? 'w-16' : 'w-60'
      )}
    >
      <div className="flex items-center justify-between p-4">
        {!collapsed && (
          <span className="text-lg font-semibold text-text-primary">MyEnglish</span>
        )}
        <button
          type="button"
          onClick={onToggle}
          className="rounded-md p-1 text-text-secondary hover:bg-poppy-light hover:text-poppy transition-colors duration-150"
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? <ChevronRight size={20} /> : <ChevronLeft size={20} />}
        </button>
      </div>

      <nav className="flex flex-1 flex-col gap-1 px-2">
        {navItems.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            onClick={onClose}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors duration-150',
                isActive
                  ? 'bg-poppy-light text-poppy'
                  : 'text-text-secondary hover:bg-poppy-light hover:text-poppy'
              )
            }
          >
            {({ isActive }) => (
              <>
                <Icon size={20} fill={isActive ? 'currentColor' : 'none'} />
                {!collapsed && <span>{label}</span>}
              </>
            )}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
