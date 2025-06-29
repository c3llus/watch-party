import { useParams, useSearchParams, useNavigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { roomService } from '../services/roomService'

export default function WaitingPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  
  const requestId = searchParams.get('requestId')
  const guestName = searchParams.get('guestName')
  const isGuest = !!requestId && !!guestName
  
  const [status, setStatus] = useState<'pending' | 'approved' | 'denied' | 'error'>('pending')
  const [error, setError] = useState<string | null>(null)
  const [isPolling, setIsPolling] = useState(true)

  useEffect(() => {
    if (!roomId) {
      setError('Missing room ID')
      setStatus('error')
      return
    }

    if (isGuest && !requestId) {
      setError('Missing request ID for guest access')
      setStatus('error')
      return
    }

    let pollInterval: number

    const pollRequestStatus = async () => {
      try {
        let response: { status: 'pending' | 'approved' | 'denied'; session_token?: string; expires_at?: string }
        
        if (isGuest) {
          // guest polling - uses public endpoint
          response = await roomService.checkGuestRequestStatus(requestId!)
        } else {
          // user polling - uses authenticated endpoint
          response = await roomService.checkRoomAccessRequestStatus(roomId)
        }
        
        if (response.status === 'approved') {
          setStatus('approved')
          setIsPolling(false)
          
          if (isGuest) {
            // redirect to room with guest token
            if (response.session_token) {
              navigate(`/rooms/${roomId}?guestToken=${response.session_token}&guestName=${encodeURIComponent(guestName || 'Guest')}`)
            }
          } else {
            // redirect to room as authenticated user
            navigate(`/rooms/${roomId}`)
          }
        } else if (response.status === 'denied') {
          setStatus('denied')
          setIsPolling(false)
        }
        // if still pending, continue polling
      } catch (err) {
        console.error('failed to check request status:', err)
        setError('Failed to check request status')
        setStatus('error')
        setIsPolling(false)
      }
    }

    // poll immediately
    pollRequestStatus()

    // then poll every 3 seconds
    if (isPolling) {
      pollInterval = setInterval(pollRequestStatus, 3000)
    }

    return () => {
      if (pollInterval) {
        clearInterval(pollInterval)
      }
    }
  }, [roomId, requestId, guestName, navigate, isPolling, isGuest])

  const handleGoHome = () => {
    navigate('/')
  }

  const getRequestTypeText = () => {
    return isGuest ? 'guest access' : 'room access'
  }

  const renderContent = () => {
    switch (status) {
      case 'pending':
        return (
          <div style={{ textAlign: 'center' }}>
            <div style={{
              width: '40px',
              height: '40px',
              border: '4px solid #f3f3f3',
              borderTop: '4px solid #007bff',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite',
              margin: '0 auto 1rem'
            }} />
            <h2 style={{ color: '#333', marginBottom: '1rem' }}>
              Waiting for Approval
            </h2>
            <p style={{ color: '#666', marginBottom: '1rem' }}>
              Your request for {getRequestTypeText()} has been sent.
            </p>
            <p style={{ color: '#666', fontSize: '0.875em' }}>
              {isGuest ? 'The host' : 'An admin'} will review your request shortly...
            </p>
          </div>
        )
      
      case 'approved':
        return (
          <div style={{ textAlign: 'center' }}>
            <div style={{
              width: '40px',
              height: '40px',
              backgroundColor: '#28a745',
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              margin: '0 auto 1rem',
              color: 'white',
              fontSize: '1.5em'
            }}>
              ✓
            </div>
            <h2 style={{ color: '#28a745', marginBottom: '1rem' }}>
              Request Approved!
            </h2>
            <p style={{ color: '#666' }}>
              Redirecting to the watch party...
            </p>
          </div>
        )
      
      case 'denied':
        return (
          <div style={{ textAlign: 'center' }}>
            <div style={{
              width: '40px',
              height: '40px',
              backgroundColor: '#dc3545',
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              margin: '0 auto 1rem',
              color: 'white',
              fontSize: '1.5em'
            }}>
              ✗
            </div>
            <h2 style={{ color: '#dc3545', marginBottom: '1rem' }}>
              Request Denied
            </h2>
            <p style={{ color: '#666', marginBottom: '2rem' }}>
              The host has declined your request to join this watch party.
            </p>
            <button
              onClick={handleGoHome}
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#007bff',
                color: 'white',
                border: 'none',
                borderRadius: '8px',
                cursor: 'pointer',
                fontSize: '1rem'
              }}
            >
              Go Home
            </button>
          </div>
        )
      
      case 'error':
        return (
          <div style={{ textAlign: 'center' }}>
            <div style={{
              width: '40px',
              height: '40px',
              backgroundColor: '#ffc107',
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              margin: '0 auto 1rem',
              color: 'white',
              fontSize: '1.5em'
            }}>
              !
            </div>
            <h2 style={{ color: '#dc3545', marginBottom: '1rem' }}>
              Something went wrong
            </h2>
            <p style={{ color: '#666', marginBottom: '2rem' }}>
              {error || 'Unable to check your request status.'}
            </p>
            <button
              onClick={handleGoHome}
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#007bff',
                color: 'white',
                border: 'none',
                borderRadius: '8px',
                cursor: 'pointer',
                fontSize: '1rem'
              }}
            >
              Go Home
            </button>
          </div>
        )
      
      default:
        return null
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: '#f8f9fa',
      fontFamily: 'system-ui, sans-serif'
    }}>
      <div style={{
        maxWidth: '400px',
        width: '100%',
        padding: '2rem',
        backgroundColor: 'white',
        borderRadius: '12px',
        boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)'
      }}>
        {renderContent()}
      </div>
      
      {/* CSS animation for spinner */}
      <style>
        {`
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
        `}
      </style>
    </div>
  )
}
