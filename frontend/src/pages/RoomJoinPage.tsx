import { useEffect, useState } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { roomService, type Room } from '../services/roomService'
import { authService } from '../services/authService'

export default function RoomJoinPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  
  const [room, setRoom] = useState<Room | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [userRole, setUserRole] = useState<string | null>(null)

  useEffect(() => {
    const checkAccessAndRedirect = async () => {
      if (!roomId) {
        navigate('/')
        return
      }

      try {
        setIsLoading(true)
        setError(null)

        // check if user is authenticated
        const currentUser = authService.getCurrentUser()
        setUserRole(currentUser?.role || null)

        // check for guest tokens in URL
        const guestToken = searchParams.get('guestToken')
        const guestName = searchParams.get('guestName')

        if (guestToken && guestName) {
          // direct guest access with approved token
          navigate(`/rooms/${roomId}?guestToken=${guestToken}&guestName=${encodeURIComponent(guestName)}`)
          return
        }

        if (currentUser) {
          // authenticated users can try to get room details and join
          try {
            const roomData = await roomService.getRoomForJoin(roomId)
            setRoom(roomData)
            // redirect directly to room
            navigate(`/rooms/${roomId}`)
            return
          } catch (err) {
            // if getRoomForJoin fails, user doesn't have access
            console.error('user does not have access to room:', err)
            setError('You do not have access to this room. Please request an invitation from the host.')
          }
        } else {
          // user is not authenticated, redirect to guest request page
          // don't try to fetch room details here since guest endpoints don't require auth
          navigate(`/guest/request/${roomId}`)
          return
        }

      } catch (err) {
        console.error('failed to check room access:', err)
        setError(err instanceof Error ? err.message : 'Failed to access room')
      } finally {
        setIsLoading(false)
      }
    }

    checkAccessAndRedirect()
  }, [roomId, navigate, searchParams])

  // loading state
  if (isLoading) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>Checking room access...</p>
      </div>
    )
  }

  // error state (access denied for authenticated users)
  if (error) {
    return (
      <div style={{ 
        padding: '2rem',
        fontFamily: 'system-ui, sans-serif',
        maxWidth: '600px',
        margin: '0 auto'
      }}>
        <div style={{
          textAlign: 'center',
          padding: '3rem 2rem',
          backgroundColor: '#f8d7da',
          border: '1px solid #f5c6cb',
          borderRadius: '12px'
        }}>
          <h1 style={{ margin: '0 0 1rem 0', color: '#721c24' }}>
            Access Denied
          </h1>
          <p style={{ margin: '0 0 1.5rem 0', color: '#721c24' }}>
            {error}
          </p>
          
          {room && (
            <div style={{
              padding: '1rem',
              backgroundColor: '#fff',
              border: '1px solid #f5c6cb',
              borderRadius: '8px',
              marginBottom: '2rem',
              textAlign: 'left'
            }}>
              <h3 style={{ margin: '0 0 0.5rem 0', color: '#333' }}>
                {room.name}
              </h3>
              {room.description && (
                <p style={{ margin: '0 0 0.5rem 0', color: '#666' }}>
                  {room.description}
                </p>
              )}
              {room.movie && (
                <p style={{ margin: 0, color: '#666', fontSize: '0.9em' }}>
                  Movie: {room.movie.title}
                </p>
              )}
            </div>
          )}

          <div style={{ display: 'flex', gap: '1rem', justifyContent: 'center' }}>
            <button 
              onClick={() => navigate('/')}
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#007bff',
                color: 'white',
                border: 'none',
                borderRadius: '8px',
                cursor: 'pointer',
                fontWeight: 'bold'
              }}
            >
              Go Home
            </button>
            
            {userRole !== 'guest' && (
              <button 
                onClick={() => navigate('/login')}
                style={{
                  padding: '0.75rem 1.5rem',
                  backgroundColor: '#6c757d',
                  color: 'white',
                  border: 'none',
                  borderRadius: '8px',
                  cursor: 'pointer'
                }}
              >
                Try Different Account
              </button>
            )}
          </div>
        </div>

        {/* guest access option */}
        <div style={{
          marginTop: '2rem',
          padding: '2rem',
          backgroundColor: '#e7f3ff',
          border: '1px solid #b8daff',
          borderRadius: '12px',
          textAlign: 'center'
        }}>
          <h2 style={{ margin: '0 0 1rem 0', color: '#004085' }}>
            Join as Guest
          </h2>
          <p style={{ margin: '0 0 1.5rem 0', color: '#004085' }}>
            Don't have access? You can request to join as a guest.
          </p>
          <button 
            onClick={() => navigate(`/guest/request/${roomId}`)}
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: '#007bff',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              cursor: 'pointer',
              fontWeight: 'bold'
            }}
          >
            Request Guest Access
          </button>
        </div>
      </div>
    )
  }

  // fallback (shouldn't reach here normally)
  return (
    <div style={{ 
      padding: '2rem', 
      textAlign: 'center',
      fontFamily: 'system-ui, sans-serif'
    }}>
      <p>Redirecting...</p>
    </div>
  )
}
