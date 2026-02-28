import { Link } from 'react-router'
import { BookPlus, GraduationCap, AlertTriangle, RefreshCw } from 'lucide-react'
import { useAuth } from '@/providers/AuthProvider'
import { useDashboard } from '@/hooks/useDashboard'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { StatCard } from '@/components/dashboard/StatCard'
import { StreakDisplay } from '@/components/dashboard/StreakDisplay'
import { StatusDistribution } from '@/components/dashboard/StatusDistribution'

function getGreeting(): string {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}

function DashboardSkeleton() {
  return (
    <div>
      <Skeleton className="h-8 w-48 mb-sm bg-straw" />
      <Skeleton className="h-5 w-32 mb-lg bg-straw" />
      <Skeleton className="h-10 w-36 mb-lg bg-straw" />
      <div className="grid grid-cols-1 gap-md sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-[104px] rounded-md bg-straw" />
        ))}
      </div>
      <Skeleton className="mt-md h-[120px] rounded-md bg-straw" />
    </div>
  )
}

function DashboardError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-2xl text-center">
      <AlertTriangle className="size-10 text-poppy mb-md" />
      <h2 className="font-heading text-[22px] text-text-primary mb-sm">
        Failed to load dashboard
      </h2>
      <p className="text-[13px] text-text-secondary mb-md">
        Something went wrong. Please try again.
      </p>
      <Button variant="outline" onClick={onRetry}>
        <RefreshCw className="size-4" />
        Retry
      </Button>
    </div>
  )
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-2xl text-center">
      <BookPlus className="size-10 text-thyme mb-md" />
      <h2 className="font-heading text-[22px] text-text-primary mb-sm">
        Your dictionary is empty
      </h2>
      <p className="text-[13px] text-text-secondary mb-md max-w-sm">
        Start by adding some words to your dictionary. You can search our catalog
        or add custom words.
      </p>
      <Button variant="calm" asChild>
        <Link to="/dictionary">Add your first word</Link>
      </Button>
    </div>
  )
}

function DashboardPage() {
  const { user } = useAuth()
  const { data, loading, error, refetch } = useDashboard()

  if (loading && !data) return <DashboardSkeleton />
  if (error && !data) return <DashboardError onRetry={() => refetch()} />

  if (data && data.statusCounts.total === 0) {
    return (
      <div>
        <h1 className="font-heading text-[28px] leading-tight text-text-primary mb-xs">
          {getGreeting()}, {user?.name ?? user?.username ?? 'there'}
        </h1>
        <p className="text-[13px] text-text-secondary mb-lg">
          Let&apos;s get started with your vocabulary
        </p>
        <EmptyState />
      </div>
    )
  }

  if (!data) return null

  const hasActiveSession = data.activeSession?.status === 'ACTIVE'

  return (
    <div>
      {/* Header */}
      <h1 className="font-heading text-[28px] leading-tight text-text-primary mb-xs">
        {getGreeting()}, {user?.name ?? user?.username ?? 'there'}
      </h1>
      <p className="text-[13px] text-text-secondary mb-lg">
        Your learning progress overview
      </p>

      {/* Primary action */}
      <Button variant="calm" size="lg" className="mb-lg" asChild>
        <Link to="/study">
          <GraduationCap className="size-5" />
          {hasActiveSession ? 'Continue Study' : 'Start Study'}
        </Link>
      </Button>

      {/* Stats grid */}
      <div className="grid grid-cols-1 gap-md sm:grid-cols-2 lg:grid-cols-3">
        <StatCard
          label="Cards due"
          value={data.dueCount}
          accent={data.overdueCount > 0 ? 'text-poppy' : undefined}
        />
        <StatCard
          label="New cards available"
          value={data.newCount}
        />
        <StatCard
          label="Reviewed today"
          value={data.reviewedToday}
        />
        <StatCard
          label="New today"
          value={data.newToday}
        />
        {data.overdueCount > 0 && (
          <StatCard
            label="Overdue"
            value={data.overdueCount}
            accent="text-poppy"
          />
        )}
        <StreakDisplay streak={data.streak} />
      </div>

      {/* Status distribution */}
      <StatusDistribution
        statusCounts={data.statusCounts}
        className="mt-md"
      />
    </div>
  )
}

export default DashboardPage
