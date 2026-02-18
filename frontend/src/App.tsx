import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './auth/AuthProvider'
import { Layout } from './components/Layout'

function Placeholder({ name }: { name: string }) {
  return <div className="p-6 text-gray-500">Page: {name} (coming soon)</div>
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Placeholder name="Login" />} />
          <Route element={<Layout />}>
            <Route path="/" element={<Navigate to="/dictionary" replace />} />
            <Route path="/catalog" element={<Placeholder name="Catalog" />} />
            <Route path="/dictionary" element={<Placeholder name="Dictionary" />} />
            <Route path="/entry/:id" element={<Placeholder name="Entry Detail" />} />
            <Route path="/study" element={<Placeholder name="Study" />} />
            <Route path="/topics" element={<Placeholder name="Topics" />} />
            <Route path="/inbox" element={<Placeholder name="Inbox" />} />
            <Route path="/profile" element={<Placeholder name="Profile" />} />
            <Route path="/explorer" element={<Placeholder name="API Explorer" />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}

export default App
