import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'
import UploadPage from './pages/UploadPage'
import MovieLibraryPage from './pages/MovieLibraryPage'
import AdminDashboardPage from './pages/AdminDashboardPage'
import './App.css'

function App() {
  return (
    <Router>
      <div>
        <nav style={{ padding: '1rem', borderBottom: '1px solid black' }}>
          <Link to="/login" style={{ marginRight: '1rem', color: 'black' }}>Login</Link>
          <Link to="/signup" style={{ marginRight: '1rem', color: 'black' }}>Signup</Link>
          <Link to="/admin" style={{ marginRight: '1rem', color: 'black' }}>Admin</Link>
          <Link to="/admin/upload" style={{ marginRight: '1rem', color: 'black' }}>Upload</Link>
          <Link to="/admin/movies" style={{ marginRight: '1rem', color: 'black' }}>Movies</Link>
        </nav>
        
        <Routes>
          <Route path="/" element={<LoginPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route path="/admin" element={<AdminDashboardPage />} />
          <Route path="/admin/upload" element={<UploadPage />} />
          <Route path="/admin/movies" element={<MovieLibraryPage />} />
        </Routes>
      </div>
    </Router>
  )
}

export default App
