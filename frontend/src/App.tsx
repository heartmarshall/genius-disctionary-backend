import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './auth/AuthProvider'
import { LoginPage } from './auth/LoginPage'
import { Layout } from './components/Layout'
import { ToastProvider } from './components/ToastProvider'
import { CatalogPage } from './pages/CatalogPage'
import { DictionaryPage } from './pages/DictionaryPage'
import { EntryDetailPage } from './pages/EntryDetailPage'
import { ExplorerPage } from './pages/ExplorerPage'
import { InboxPage } from './pages/InboxPage'
import { ProfilePage } from './pages/ProfilePage'
import { StudyPage } from './pages/StudyPage'
import { TopicsPage } from './pages/TopicsPage'

function App() {
  return (
    <AuthProvider>
      <ToastProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<Layout />}>
              <Route path="/" element={<Navigate to="/dictionary" replace />} />
              <Route path="/catalog" element={<CatalogPage />} />
              <Route path="/dictionary" element={<DictionaryPage />} />
              <Route path="/entry/:id" element={<EntryDetailPage />} />
              <Route path="/study" element={<StudyPage />} />
              <Route path="/topics" element={<TopicsPage />} />
              <Route path="/inbox" element={<InboxPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/explorer" element={<ExplorerPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ToastProvider>
    </AuthProvider>
  )
}

export default App
