import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'
import './App.css'

function App() {
  return (
    <Router>
      <div>
        <nav style={{ padding: '1rem', borderBottom: '1px solid black' }}>
          <Link to="/login" style={{ marginRight: '1rem', color: 'black' }}>Login</Link>
          <Link to="/signup" style={{ marginRight: '1rem', color: 'black' }}>Signup</Link>
        </nav>
        
        <Routes>
          <Route path="/" element={<LoginPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
        </Routes>
      </div>
    </Router>
  )
}

export default App
