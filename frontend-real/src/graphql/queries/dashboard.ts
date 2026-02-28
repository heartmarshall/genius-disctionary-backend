import { gql } from '@apollo/client/core'

export interface StatusCounts {
  new: number
  learning: number
  review: number
  relearning: number
  total: number
}

export interface ActiveSession {
  id: string
  status: 'ACTIVE' | 'FINISHED' | 'ABANDONED'
}

export interface DashboardData {
  dashboard: {
    dueCount: number
    newCount: number
    reviewedToday: number
    newToday: number
    streak: number
    overdueCount: number
    statusCounts: StatusCounts
    activeSession: ActiveSession | null
  }
}

export const DASHBOARD_QUERY = gql`
  query Dashboard {
    dashboard {
      dueCount
      newCount
      reviewedToday
      newToday
      streak
      overdueCount
      statusCounts {
        new
        learning
        review
        relearning
        total
      }
      activeSession {
        id
        status
      }
    }
  }
`
