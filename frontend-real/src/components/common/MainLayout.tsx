import { useEffect, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { AnimatePresence, motion } from 'framer-motion'
import { Menu } from 'lucide-react'
import { Sidebar } from './Sidebar'
import { SideLabel } from './SideLabel'

export function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const sidebarRef = useRef<HTMLDivElement>(null)

  // Close mobile sidebar on Escape + lock body scroll
  useEffect(() => {
    if (!mobileOpen) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMobileOpen(false)
    }
    document.addEventListener('keydown', handleKeyDown)
    document.body.style.overflow = 'hidden'

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [mobileOpen])

  // Focus trap for mobile overlay
  useEffect(() => {
    if (!mobileOpen) return

    const sidebar = sidebarRef.current
    if (!sidebar) return

    const getFocusable = () =>
      sidebar.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])'
      )

    const handleTab = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return

      const focusable = getFocusable()
      if (focusable.length === 0) return

      const first = focusable[0]
      const last = focusable[focusable.length - 1]

      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }

    const focusable = getFocusable()
    focusable[0]?.focus()

    document.addEventListener('keydown', handleTab)
    return () => document.removeEventListener('keydown', handleTab)
  }, [mobileOpen])

  // Reset mobile sidebar when viewport grows to desktop
  useEffect(() => {
    const mql = window.matchMedia('(min-width: 768px)')
    const handler = (e: MediaQueryListEvent) => {
      if (e.matches) setMobileOpen(false)
    }
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [])

  return (
    <div className="flex h-screen overflow-hidden bg-bg-page">
      {/* <SideLabel /> */}
      {/* Desktop sidebar */}
      <div className="hidden md:flex">
        <Sidebar collapsed={collapsed} onToggle={() => setCollapsed((v) => !v)} />
      </div>

      {/* Mobile overlay */}
      <AnimatePresence>
        {mobileOpen && (
          <motion.div
            key="mobile-nav"
            className="fixed inset-0 z-20 flex md:hidden"
            role="dialog"
            aria-modal="true"
            aria-label="Navigation"
            initial="closed"
            animate="open"
            exit="closed"
          >
            <motion.div
              variants={{
                open: { opacity: 1 },
                closed: { opacity: 0 },
              }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 bg-black/40"
              aria-hidden="true"
              onClick={() => setMobileOpen(false)}
            />
            <motion.div
              variants={{
                open: { x: 0 },
                closed: { x: '-100%' },
              }}
              transition={{ duration: 0.3, ease: 'easeInOut' }}
              className="relative z-20"
              ref={sidebarRef}
            >
              <Sidebar
                collapsed={false}
                onToggle={() => setMobileOpen(false)}
                onClose={() => setMobileOpen(false)}
              />
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Mobile header */}
        <header className="flex items-center border-b border-border-default p-4 md:hidden">
          <button
            type="button"
            onClick={() => setMobileOpen(true)}
            className="text-text-secondary"
            aria-label="Open navigation"
          >
            <Menu size={24} />
          </button>
          <span className="ml-3 text-lg font-semibold tracking-tight text-text-primary">MyEnglish</span>
        </header>

        <main className="flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
