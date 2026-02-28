import { useQuery } from '@apollo/client/react'
import { DASHBOARD_QUERY, type DashboardData } from '@/graphql/queries/dashboard'

export function useDashboard() {
  const { data, loading, error, refetch } = useQuery<DashboardData>(DASHBOARD_QUERY, {
    pollInterval: 30_000,
  })

  return {
    data: data?.dashboard ?? null,
    loading,
    error: error ?? null,
    refetch,
  }
}
