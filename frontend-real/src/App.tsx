import { BrowserRouter, Routes, Route, Navigate } from 'react-router'
import { GoogleOAuthProvider } from '@react-oauth/google'
import { ApolloProvider } from '@/providers/ApolloProvider'
import { AuthProvider } from '@/providers/AuthProvider'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { GuestRoute } from '@/components/GuestRoute'
import { Toaster } from '@/components/ui/sonner'
import AuthLayout from '@/layouts/AuthLayout'
import MainLayout from '@/layouts/MainLayout'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import DashboardPage from '@/pages/DashboardPage'
import DictionaryPage from '@/pages/DictionaryPage'
import DictionaryEntryPage from '@/pages/DictionaryEntryPage'
import StudyPage from '@/pages/StudyPage'
import TopicsPage from '@/pages/TopicsPage'
import InboxPage from '@/pages/InboxPage'
import SettingsPage from '@/pages/SettingsPage'
import AdminPage from '@/pages/AdminPage'
import NotFoundPage from '@/pages/NotFoundPage'

const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || ''

function GoogleOAuthWrapper({ children }: { children: React.ReactNode }) {
  if (!googleClientId) return <>{children}</>
  return <GoogleOAuthProvider clientId={googleClientId}>{children}</GoogleOAuthProvider>
}

function App() {
  return (
    <GoogleOAuthWrapper>
    <ApolloProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            {/* Auth routes — redirect to dashboard if already logged in */}
            <Route
              element={
                <GuestRoute>
                  <AuthLayout />
                </GuestRoute>
              }
            >
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
            </Route>

            {/* Protected app routes */}
            <Route
              element={
                <ProtectedRoute>
                  <MainLayout />
                </ProtectedRoute>
              }
            >
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<DashboardPage />} />
              <Route path="/dictionary" element={<DictionaryPage />} />
              <Route path="/dictionary/:id" element={<DictionaryEntryPage />} />
              <Route path="/study" element={<StudyPage />} />
              <Route path="/topics" element={<TopicsPage />} />
              <Route path="/inbox" element={<InboxPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/admin" element={<AdminPage />} />
            </Route>

            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </BrowserRouter>
        <Toaster position="bottom-right" />
      </AuthProvider>
    </ApolloProvider>
    </GoogleOAuthWrapper>
  )
}

export default App
