import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { roomService } from '../services/roomService'

export default function GuestRequestPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const navigate = useNavigate()
  
  const [guestName, setGuestName] = useState('')
  const [message, setMessage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!roomId || !guestName.trim()) return
    
    try {
      setIsSubmitting(true)
      setError(null)
      
      const response = await roomService.requestGuestAccess(
        roomId,
        guestName.trim(),
        message.trim() || undefined
      )
      
      // redirect to waiting page with request ID
      navigate(`/waiting/${roomId}?requestId=${response.request_id}&guestName=${encodeURIComponent(guestName.trim())}`)
      
    } catch (err) {
      console.error('failed to request guest access:', err)
      setError(err instanceof Error ? err.message : 'Failed to request access')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!roomId) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>Invalid room link</p>
        <button 
          onClick={() => navigate('/')}
          style={{
            padding: '0.5rem 1rem',
            backgroundColor: '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer'
          }}
        >
          Go Home
        </button>
      </div>
    )
  }

  return (
    <div style={{ 
      padding: '2rem',
      fontFamily: 'system-ui, sans-serif',
      maxWidth: '800px',
      margin: '0 auto'
    }}>
      <div style={{
        textAlign: 'center',
        marginBottom: '2rem'
      }}>
        <h1 style={{ margin: '0 0 1rem 0', color: '#333' }}>
          Join Watch Party
        </h1>
        <p style={{ margin: '0', color: '#666' }}>
          Request access to join this watch party room
        </p>
      </div>

      {error && (
        <div style={{
          padding: '0.75rem',
          backgroundColor: '#f8d7da',
          border: '1px solid #f5c6cb',
          borderRadius: '4px',
          color: '#721c24',
          marginBottom: '1rem'
        }}>
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} style={{ maxWidth: '600px', margin: '0 auto' }}>
        <div style={{ marginBottom: '1rem' }}>
          <label style={{ 
            display: 'block', 
            marginBottom: '0.5rem',
            fontWeight: 'bold',
            color: '#333'
          }}>
            Your Name *
          </label>
          <input
            type="text"
            value={guestName}
            onChange={(e) => setGuestName(e.target.value)}
            placeholder="Enter your name"
            required
            disabled={isSubmitting}
            style={{
              width: '100%',
              padding: '0.75rem',
              border: '1px solid #ccc',
              borderRadius: '4px',
              boxSizing: 'border-box'
            }}
          />
        </div>

        <div style={{ marginBottom: '1.5rem' }}>
          <label style={{ 
            display: 'block', 
            marginBottom: '0.5rem',
            fontWeight: 'bold',
            color: '#333'
          }}>
            Message to Host (optional)
          </label>
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="Hi! I'd like to join your watch party..."
            rows={3}
            disabled={isSubmitting}
            style={{
              width: '100%',
              padding: '0.75rem',
              border: '1px solid #ccc',
              borderRadius: '4px',
              boxSizing: 'border-box',
              resize: 'vertical'
            }}
          />
        </div>

        <div style={{
          display: 'flex',
          gap: '1rem',
          justifyContent: 'center'
        }}>
          <button
            type="submit"
            disabled={isSubmitting || !guestName.trim()}
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: isSubmitting || !guestName.trim() ? '#6c757d' : '#28a745',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              cursor: isSubmitting || !guestName.trim() ? 'not-allowed' : 'pointer',
              fontWeight: 'bold'
            }}
          >
            {isSubmitting ? 'Requesting Access...' : 'Request Access'}
          </button>
          
          <button
            type="button"
            onClick={() => navigate('/')}
            disabled={isSubmitting}
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: '#6c757d',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              cursor: isSubmitting ? 'not-allowed' : 'pointer'
            }}
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
