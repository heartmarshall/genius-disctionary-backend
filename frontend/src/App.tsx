import { BrowserRouter, Routes, Route } from 'react-router-dom'

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-50 text-gray-900">
        <h1 className="text-2xl p-4">MyEnglish Test Frontend</h1>
        <Routes>
          <Route path="/" element={<div className="p-4">Home</div>} />
        </Routes>
      </div>
    </BrowserRouter>
  )
}

export default App
