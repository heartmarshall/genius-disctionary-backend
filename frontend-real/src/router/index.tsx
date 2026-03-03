import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AuthLayout } from '@/components/common/AuthLayout'
import { MainLayout } from '@/components/common/MainLayout'
import { ProtectedRoute } from '@/components/common/ProtectedRoute'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { DictionaryPage } from '@/pages/DictionaryPage'
import { DictionaryEntryPage } from '@/pages/DictionaryEntryPage'
import { StudyPage } from '@/pages/StudyPage'
import { TopicsPage } from '@/pages/TopicsPage'
import { InboxPage } from '@/pages/InboxPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { AdminPage } from '@/pages/AdminPage'
import { NotFoundPage } from '@/pages/NotFoundPage'

export const router = createBrowserRouter([
  {
    element: <AuthLayout />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
    ],
  },
  {
    element: (
      <ProtectedRoute>
        <MainLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: '/dashboard', element: <DashboardPage /> },
      { path: '/dictionary', element: <DictionaryPage /> },
      { path: '/dictionary/:id', element: <DictionaryEntryPage /> },
      { path: '/study', element: <StudyPage /> },
      { path: '/topics', element: <TopicsPage /> },
      { path: '/inbox', element: <InboxPage /> },
      { path: '/settings', element: <SettingsPage /> },
      { path: '/admin', element: <AdminPage /> },
    ],
  },
  { path: '*', element: <NotFoundPage /> },
])
