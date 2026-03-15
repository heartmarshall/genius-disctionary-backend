import { useLocation } from 'react-router-dom'

const ROUTE_LABELS: Record<string, string> = {
  '/dashboard': 'MyEnglish · Dashboard',
  '/dictionary': 'MyEnglish · Dictionary',
  '/study': 'MyEnglish · Study',
  '/topics': 'MyEnglish · Topics',
  '/inbox': 'MyEnglish · Inbox',
  '/settings': 'MyEnglish · Settings',
  '/admin': 'MyEnglish · Admin',
}

export function SideLabel() {
  const location = useLocation()
  const basePath = '/' + (location.pathname.split('/')[1] ?? '')
  const label = ROUTE_LABELS[basePath] ?? 'MyEnglish'

  return (
    <div
      className="hidden md:block fixed left-3 top-1/2 -translate-y-1/2 -rotate-90 origin-center z-0 select-none pointer-events-none"
      aria-hidden="true"
    >
      <span className="text-[10px] uppercase tracking-[3px] text-border-default whitespace-nowrap">
        {label}
      </span>
    </div>
  )
}
