import { useState } from 'react'
import { authService } from '../services/authService'

export default function SignupPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      // regular user registration - self-service signup
      const result = await authService.register({ email, password })
      console.log('user registered successfully:', result.user.email)
      
      // redirect to login page after successful registration
      window.location.href = '/login'
    } catch (err) {
      setError(err instanceof Error ? err.message : 'registration failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ padding: '2rem', maxWidth: '400px', margin: '0 auto' }}>
      <h1>Sign Up</h1>
      <p>create your account for watch party access</p>
      
      {error && (
        <div style={{ color: 'red', marginBottom: '1rem' }}>
          {error}
        </div>
      )}
      
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: '1rem' }}>
          <label>Email:</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            style={{ width: '100%', padding: '0.5rem', marginTop: '0.25rem' }}
            required
            disabled={loading}
          />
        </div>
        
        <div style={{ marginBottom: '1rem' }}>
          <label>Password:</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={{ width: '100%', padding: '0.5rem', marginTop: '0.25rem' }}
            required
            minLength={8}
            disabled={loading}
          />
          <small style={{ color: 'gray' }}>minimum 8 characters</small>
        </div>
        
        <button 
          type="submit" 
          style={{ padding: '0.5rem 1rem', width: '100%' }}
          disabled={loading}
        >
          {loading ? 'creating account...' : 'sign up'}
        </button>
      </form>
      
      <p style={{ marginTop: '1rem', textAlign: 'center' }}>
        already have an account? <a href="/login">login</a>
      </p>
    </div>
  )
}
