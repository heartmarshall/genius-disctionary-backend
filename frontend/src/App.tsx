import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './auth/AuthProvider'
import { LoginPage } from './auth/LoginPage'
import { Layout } from './components/Layout'
import { CatalogPage } from './pages/CatalogPage'
import { DictionaryPage } from './pages/DictionaryPage'
import { EntryDetailPage } from './pages/EntryDetailPage'
import { InboxPage } from './pages/InboxPage'
import { ProfilePage } from './pages/ProfilePage'
import { StudyPage } from './pages/StudyPage'
import { TopicsPage } from './pages/TopicsPage'

function Placeholder({ name }: { name: string }) {
  return <div className="p-6 text-gray-500">Page: {name} (coming soon)</div>
}

function App() {
  return (
    <AuthProvider>
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
            <Route path="/explorer" element={<Placeholder name="API Explorer" />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}

export default App
