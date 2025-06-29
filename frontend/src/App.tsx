import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { authService } from './services/authService'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'
import UploadPage from './pages/UploadPage'
import MovieLibraryPage from './pages/MovieLibraryPage'
import AdminDashboardPage from './pages/AdminDashboardPage'
import UserDashboardPage from './pages/UserDashboardPage'
import GuestLandingPage from './pages/GuestLandingPage'
import RoomPage from './pages/RoomPage'
import RoomJoinPage from './pages/RoomJoinPage'
import RoomCreatePage from './pages/RoomCreatePage'
import RoomSuccessPage from './pages/RoomSuccessPage'
import GuestRequestPage from './pages/GuestRequestPage'
import WaitingPage from './pages/WaitingPage'
import './App.css'

function App() {
  const [user, setUser] = useState<{ email: string; role: string } | null>(null)
  const [loading, setLoading] = useState(true)

  // check authentication status on app load
  useEffect(() => {
    const currentUser = authService.getCurrentUser()
    setUser(currentUser)
    setLoading(false)
  }, [])

  const handleLogout = () => {
    authService.logout()
    setUser(null)
    window.location.href = '/login'
  }

  if (loading) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>Loading...</p>
      </div>
    )
  }

  return (
    <Router>
      <div>
        <nav style={{ 
          padding: '1rem', 
          borderBottom: '1px solid #e9ecef',
          backgroundColor: '#f8f9fa',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
            <Link 
              to={user ? "/dashboard" : "/"}
              style={{ 
                fontWeight: 'bold', 
                fontSize: '1.2em', 
                color: '#007bff',
                textDecoration: 'none'
              }}
            >
              Watch Party
            </Link>
            
            {user && user.role === 'admin' && (
              <>
                <Link to="/admin" style={{ color: '#333', textDecoration: 'none' }}>
                  Admin
                </Link>
                <Link to="/admin/upload" style={{ color: '#333', textDecoration: 'none' }}>
                  Upload
                </Link>
                <Link to="/admin/movies" style={{ color: '#333', textDecoration: 'none' }}>
                  Movies
                </Link>
              </>
            )}
          </div>

          <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
            {user ? (
              <>
                <span style={{ color: '#666', fontSize: '0.9em' }}>
                  {user.email} ({user.role})
                </span>
                <button
                  onClick={handleLogout}
                  style={{
                    padding: '0.5rem 1rem',
                    backgroundColor: '#dc3545',
                    color: 'white',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer'
                  }}
                >
                  Logout
                </button>
              </>
            ) : (
              <>
                <Link 
                  to="/login" 
                  style={{ 
                    padding: '0.5rem 1rem',
                    backgroundColor: '#007bff',
                    color: 'white',
                    textDecoration: 'none',
                    borderRadius: '4px'
                  }}
                >
                  Login
                </Link>
                <Link 
                  to="/signup" 
                  style={{ 
                    padding: '0.5rem 1rem',
                    backgroundColor: '#28a745',
                    color: 'white',
                    textDecoration: 'none',
                    borderRadius: '4px'
                  }}
                >
                  Sign Up
                </Link>
              </>
            )}
          </div>
        </nav>
        
        <Routes>
          {/* public routes */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          
          {/* landing page - different for authenticated vs guest users */}
          <Route path="/" element={user ? <UserDashboardPage /> : <GuestLandingPage />} />
          
          {/* user routes - require authentication */}
          <Route path="/dashboard" element={user ? <UserDashboardPage /> : <LoginPage />} />
          
          {/* admin routes - require admin role */}
          <Route path="/admin" element={user?.role === 'admin' ? <AdminDashboardPage /> : <LoginPage />} />
          <Route path="/admin/upload" element={user?.role === 'admin' ? <UploadPage /> : <LoginPage />} />
          <Route path="/admin/movies" element={user?.role === 'admin' ? <MovieLibraryPage /> : <LoginPage />} />
          <Route path="/admin/rooms/create" element={user?.role === 'admin' ? <RoomCreatePage /> : <LoginPage />} />
          
          {/* room routes - work for both authenticated and guest users */}
          <Route path="/rooms/join/:roomId" element={<RoomJoinPage />} />
          <Route path="/rooms/:roomId/success" element={user?.role === 'admin' ? <RoomSuccessPage /> : <LoginPage />} />
          <Route path="/rooms/:roomId" element={<RoomPage />} />
          
          {/* guest routes */}
          <Route path="/guest/request/:roomId" element={<GuestRequestPage />} />
          <Route path="/guest-waiting/:roomId" element={<WaitingPage />} />
          <Route path="/waiting/:roomId" element={<WaitingPage />} />
        </Routes>
      </div>
    </Router>
  )
}

export default App
