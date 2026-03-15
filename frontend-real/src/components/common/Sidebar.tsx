import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useAuth } from '@/providers/AuthProvider'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  LayoutDashboard,
  BookOpen,
  GraduationCap,
  Tags,
  Inbox,
  Settings,
  ChevronLeft,
  ChevronRight,
  LogOut,
  X,
  Shield,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

const SIDEBAR_WIDTH_EXPANDED = 'w-60'
const SIDEBAR_WIDTH_COLLAPSED = 'w-16'

const textCollapsed = 'max-w-0 overflow-hidden opacity-0'
const textExpanded = 'max-w-[200px] opacity-100'
const textTransition = 'whitespace-nowrap transition-[opacity,max-width] duration-300 ease-in-out'

/**
 * Per-icon hover animation classes.
 * Applied to the <svg> via group-hover, using CSS transitions/transforms.
 */
const iconHoverAnimations: Record<string, string> = {
  '/dashboard':  'group-hover:scale-110',
  '/dictionary': 'group-hover:-rotate-6 group-hover:scale-105',
  '/study':      'group-hover:-translate-y-0.5 group-hover:rotate-3',
  '/topics':     'group-hover:rotate-12',
  '/inbox':      'group-hover:-translate-y-0.5',
  '/settings':   'group-hover:rotate-90',
  '/admin':      'group-hover:scale-110',
}

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
}

const navItems: NavItem[] = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/dictionary', label: 'Dictionary', icon: BookOpen },
  { to: '/study', label: 'Study', icon: GraduationCap },
  { to: '/topics', label: 'Topics', icon: Tags },
  { to: '/inbox', label: 'Inbox', icon: Inbox },
]

const settingsNavItem: NavItem = { to: '/settings', label: 'Settings', icon: Settings }

const adminNavItem: NavItem = { to: '/admin', label: 'Admin', icon: Shield }

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  onClose?: () => void
}

export function Sidebar({ collapsed, onToggle, onClose }: SidebarProps) {
  const { user, logout } = useAuth()
  const [isLoggingOut, setIsLoggingOut] = useState(false)

  const isMobileOverlay = !!onClose

  const handleLogout = () => {
    setIsLoggingOut(true)
    logout()
  }

  const items = user?.role === 'admin' ? [...navItems, adminNavItem] : navItems

  const ToggleIcon = isMobileOverlay ? X : collapsed ? ChevronRight : ChevronLeft
  const toggleAriaLabel = isMobileOverlay
    ? 'Close navigation'
    : collapsed
      ? 'Expand sidebar'
      : 'Collapse sidebar'

  return (
    <TooltipProvider delayDuration={300}>
      <aside
        aria-label="Main navigation"
        className={cn(
          'flex h-full flex-col overflow-hidden border-r border-border-default bg-surface-secondary transition-[width] duration-300 ease-in-out',
          collapsed ? SIDEBAR_WIDTH_COLLAPSED : SIDEBAR_WIDTH_EXPANDED
        )}
      >
        <div className="flex items-center justify-between p-4">
          <span
            aria-hidden={collapsed || undefined}
            className={cn(
              'font-orelega text-lg text-text-primary',
              textTransition,
              collapsed ? textCollapsed : textExpanded
            )}
          >
            MyEnglish
          </span>
          <button
            type="button"
            onClick={isMobileOverlay ? onClose : onToggle}
            className="rounded-md p-1 text-text-secondary hover:bg-poppy-light hover:text-poppy transition-colors duration-150"
            aria-label={toggleAriaLabel}
          >
            <ToggleIcon size={20} />
          </button>
        </div>

        <nav className="flex flex-1 flex-col gap-1 overflow-y-auto px-2">
          {items.map(({ to, label, icon: Icon }) => (
            <Tooltip key={to} open={collapsed ? undefined : false}>
              <TooltipTrigger asChild>
                <div>
                  <NavLink
                    to={to}
                    onClick={onClose}
                    aria-label={collapsed ? label : undefined}
                    className={({ isActive }) =>
                      cn(
                        'group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors duration-150',
                        isActive
                          ? 'bg-poppy-light text-poppy'
                          : 'text-text-secondary hover:bg-poppy-light hover:text-poppy'
                      )
                    }
                  >
                    {({ isActive }) => (
                      <>
                        <Icon
                          size={20}
                          strokeWidth={isActive ? 2 : 1.75}
                          className={cn(
                            'shrink-0 transition-transform duration-200',
                            iconHoverAnimations[to]
                          )}
                        />
                        <span
                          aria-hidden={collapsed || undefined}
                          className={cn(textTransition, collapsed ? textCollapsed : textExpanded)}
                        >
                          {label}
                        </span>
                      </>
                    )}
                  </NavLink>
                </div>
              </TooltipTrigger>
              <TooltipContent side="right">{label}</TooltipContent>
            </Tooltip>
          ))}
        </nav>

        <div className="mt-auto border-t border-border-default px-2 py-3">
          {user && (
            <p
              aria-hidden={collapsed || undefined}
              className={cn(
                'mb-2 truncate px-3 text-xs text-text-secondary',
                textTransition,
                collapsed ? textCollapsed : textExpanded
              )}
            >
              {user.email || user.username}
            </p>
          )}
          <Tooltip open={collapsed ? undefined : false}>
            <TooltipTrigger asChild>
              <div>
                <NavLink
                  to={settingsNavItem.to}
                  onClick={onClose}
                  aria-label={collapsed ? settingsNavItem.label : undefined}
                  className={({ isActive }) =>
                    cn(
                      'group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors duration-150',
                      isActive
                        ? 'bg-poppy-light text-poppy'
                        : 'text-text-secondary hover:bg-poppy-light hover:text-poppy'
                    )
                  }
                >
                  {({ isActive }) => (
                    <>
                      <Settings
                        size={20}
                        strokeWidth={isActive ? 2 : 1.75}
                        className={cn(
                          'shrink-0 transition-transform duration-200',
                          iconHoverAnimations[settingsNavItem.to]
                        )}
                      />
                      <span
                        aria-hidden={collapsed || undefined}
                        className={cn(textTransition, collapsed ? textCollapsed : textExpanded)}
                      >
                        {settingsNavItem.label}
                      </span>
                    </>
                  )}
                </NavLink>
              </div>
            </TooltipTrigger>
            <TooltipContent side="right">{settingsNavItem.label}</TooltipContent>
          </Tooltip>
          <Tooltip open={collapsed ? undefined : false}>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={handleLogout}
                disabled={isLoggingOut}
                aria-label="Log out"
                className="group flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-text-secondary transition-colors duration-150 hover:bg-poppy-light hover:text-poppy disabled:cursor-not-allowed disabled:opacity-50"
              >
                <LogOut
                  size={20}
                  className="shrink-0 transition-transform duration-200 group-hover:translate-x-0.5"
                />
                <span
                  aria-hidden={collapsed || undefined}
                  className={cn(textTransition, collapsed ? textCollapsed : textExpanded)}
                >
                  {isLoggingOut ? 'Logging out...' : 'Log out'}
                </span>
              </button>
            </TooltipTrigger>
            <TooltipContent side="right">Log out</TooltipContent>
          </Tooltip>
        </div>
      </aside>
    </TooltipProvider>
  )
}
