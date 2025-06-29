import { useParams, useSearchParams } from 'react-router-dom'
import { useEffect, useState, useCallback } from 'react'
import { useRoom } from '../hooks/useRoom'
import { roomService } from '../services/roomService'
import { authService } from '../services/authService'
import { VideoPlayer } from '../components/VideoPlayer'

interface GuestRequest {
  id: string
  guest_name: string
  message?: string
}

export default function RoomPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const [searchParams] = useSearchParams()
  
  // check if this is a guest access
  const guestToken = searchParams.get('guestToken')
  const guestName = searchParams.get('guestName')
  const isGuest = !!guestToken
  
  // track video errors separately from room errors
  const [videoError, setVideoError] = useState<string | null>(null)
  
  // track if we're applying a sync action to prevent echo
  const [isApplyingSync, setIsApplyingSync] = useState(false)
  
  // guest requests (for hosts)
  const [guestRequests, setGuestRequests] = useState<GuestRequest[]>([])
  
  // callback to wrap sync application
  const handleApplySync = useCallback((applySyncFn: () => void) => {
    setIsApplyingSync(true)
    applySyncFn()
    // reset flag after a short delay to allow events to process
    setTimeout(() => setIsApplyingSync(false), 100)
  }, [setIsApplyingSync])

  // room hook
  const {
    room,
    videoAccess,
    participants,
    isConnected,
    isLoading,
    error,
    suppressOutgoingSync,
    refreshVideoAccess,
    sendSyncAction,
    syncVideoToRoom: baseSyncVideoToRoom,
    setVideoElement
  } = useRoom({
    roomId: roomId || '',
    isGuest,
    guestToken: isGuest ? guestToken : undefined,
    guestName: isGuest && guestName ? guestName : undefined,
    onApplySync: handleApplySync
  })

  // wrap syncVideoToRoom to prevent event echo
  const syncVideoToRoom = useCallback((videoElement: HTMLVideoElement) => {
    setIsApplyingSync(true)
    baseSyncVideoToRoom(videoElement)
    // reset flag after a short delay to allow events to process
    setTimeout(() => setIsApplyingSync(false), 100)
  }, [baseSyncVideoToRoom, setIsApplyingSync])

  // fetch guest requests for admin users only
  useEffect(() => {
    if (!roomId || isGuest) return

    // check if current user is admin
    const currentUser = authService.getCurrentUser()
    if (!currentUser || currentUser.role !== 'admin') {
      return // only admin users can view guest requests
    }

    async function fetchGuestRequests() {
      try {
        const requests = await roomService.getGuestRequests(roomId!)
        setGuestRequests(requests)
      } catch (err) {
        console.error('failed to fetch guest requests:', err)
      }
    }

    // fetch immediately
    fetchGuestRequests()

    // poll every 5 seconds
    const interval = setInterval(fetchGuestRequests, 5000)

    return () => clearInterval(interval)
  }, [roomId, isGuest])

  // video event handlers
  const handleVideoError = useCallback((errorMessage: string) => {
    console.error('video error:', errorMessage)
    setVideoError(errorMessage)
  }, [])

  const handlePlay = useCallback(() => {
    if (isApplyingSync) {
      return
    }
    sendSyncAction({ action: 'play' })
  }, [sendSyncAction, isApplyingSync])

  const handlePause = useCallback(() => {
    if (isApplyingSync) {
      return
    }
    sendSyncAction({ action: 'pause' })
  }, [sendSyncAction, isApplyingSync])

  const handleSeeked = useCallback((time: number) => {
    if (isApplyingSync) {
      return
    }
    sendSyncAction({ action: 'seek', currentTime: time })
  }, [sendSyncAction, isApplyingSync])

  // guest request handlers
  const handleGuestRequest = async (requestId: string, approved: boolean) => {
    if (!roomId) return
    
    try {
      await roomService.respondToGuestRequest(roomId, requestId, approved)
      // remove from pending list
      setGuestRequests(prev => prev.filter(req => req.id !== requestId))
    } catch (err) {
      console.error('failed to respond to guest request:', err)
    }
  }

  // loading state
  if (isLoading) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>loading room...</p>
      </div>
    )
  }

  // error state
  if (error) {
    return (
      <div style={{ 
        padding: '2rem', 
        fontFamily: 'system-ui, sans-serif'
      }}>
        <div style={{
          padding: '1rem',
          backgroundColor: '#f8d7da',
          border: '1px solid #f5c6cb',
          borderRadius: '8px',
          color: '#721c24',
          marginBottom: '1rem'
        }}>
          {error}
        </div>
        <button 
          onClick={() => window.location.href = '/'}
          style={{
            padding: '0.5rem 1rem',
            backgroundColor: '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer'
          }}
        >
          go home
        </button>
      </div>
    )
  }

  if (!room) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>room not found</p>
      </div>
    )
  }

  return (
    <div style={{ 
      padding: '1rem', 
      fontFamily: 'system-ui, sans-serif',
      maxWidth: '1200px',
      margin: '0 auto'
    }}>
      {/* room header */}
      <div style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        marginBottom: '1rem'
      }}>
        <div>
          <h1 style={{ margin: 0, color: '#333' }}>{room.name}</h1>
          {room.description && (
            <p style={{ margin: '0.5rem 0 0 0', color: '#666' }}>
              {room.description}
            </p>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <div style={{
            padding: '0.25rem 0.5rem',
            backgroundColor: isConnected ? '#d4edda' : '#f8d7da',
            color: isConnected ? '#155724' : '#721c24',
            borderRadius: '4px',
            fontSize: '0.875em'
          }}>
            {isConnected ? 'connected' : 'disconnected'}
          </div>
          <div style={{ color: '#666', fontSize: '0.875em' }}>
            {participants.length} participant{participants.length !== 1 ? 's' : ''}
          </div>
        </div>
      </div>

      {/* video player container */}
      <div style={{ 
        backgroundColor: '#000',
        borderRadius: '8px',
        overflow: 'hidden',
        marginBottom: '1rem',
        maxWidth: '100%',
        aspectRatio: '16/9',
        position: 'relative'
      }}>
        {videoAccess?.hls_url && room?.movie_id ? (
          <VideoPlayer
            movieId={room.movie_id}
            guestToken={isGuest ? guestToken : undefined}
            onError={handleVideoError}
            onPlay={handlePlay}
            onPause={handlePause}
            onSeeked={handleSeeked}
            onSyncToRoom={syncVideoToRoom}
            onVideoReady={setVideoElement}
            waitForSync={isGuest && suppressOutgoingSync} // guests should wait for sync before playing
            style={{
              width: '100%',
              height: '100%',
              aspectRatio: '16/9'
            }}
          />
        ) : room?.movie_id ? (
          <div style={{
            height: '400px',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'white',
            gap: '1rem'
          }}>
            <div>{videoError || 'loading video...'}</div>
            {!videoError && !isLoading && (
              <button
                onClick={() => {
                  setVideoError(null)
                  refreshVideoAccess()
                }}
                style={{
                  padding: '0.5rem 1rem',
                  backgroundColor: '#007bff',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer'
                }}
              >
                retry loading video
              </button>
            )}
            {videoError && (
              <div style={{ 
                textAlign: 'center',
                fontSize: '0.875em',
                color: '#ccc',
                marginTop: '0.5rem'
              }}>
                try refreshing the video access or check if the movie is properly processed
              </div>
            )}
          </div>
        ) : (
          <div style={{
            height: '400px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'white'
          }}>
            no movie selected for this room
          </div>
        )}
      </div>

      {/* participants list */}
      <div style={{
        padding: '1rem',
        backgroundColor: '#fff',
        border: '1px solid #e9ecef',
        borderRadius: '8px'
      }}>
        <h3 style={{ margin: '0 0 1rem 0', color: '#333' }}>
          participants ({participants?.length || 0})
        </h3>
        {participants && participants.length > 0 ? (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
            {participants.map((participant) => (
              <div
                key={participant.id}
                style={{
                  padding: '0.25rem 0.5rem',
                  backgroundColor: participant.role === 'host' ? '#007bff' : '#6c757d',
                  color: 'white',
                  borderRadius: '4px',
                  fontSize: '0.875em'
                }}
              >
                {participant.name} {participant.role === 'host' && '(host)'}
              </div>
            ))}
          </div>
        ) : (
          <p style={{ color: '#666', margin: 0 }}>
            no other participants
          </p>
        )}

        {/* guest requests for admin users only */}
        {!isGuest && guestRequests && guestRequests.length > 0 && authService.getCurrentUser()?.role === 'admin' && (
          <div style={{ marginTop: '1rem' }}>
            <h4 style={{ margin: '0 0 0.5rem 0', color: '#333', fontSize: '1em' }}>
              pending guest requests ({guestRequests.length})
            </h4>
            {guestRequests.map((request) => (
              <div
          key={request.id}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0.5rem',
            border: '1px solid #e9ecef',
            borderRadius: '4px',
            marginBottom: '0.5rem',
            backgroundColor: '#f8f9fa'
          }}
              >
          <div>
            <strong style={{ color: 'black' }}>{request.guest_name}</strong>
            {request.message && (
              <div style={{ fontSize: '0.875em', color: '#666' }}>
                {request.message}
              </div>
            )}
          </div>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <button
              onClick={() => handleGuestRequest(request.id, true)}
              style={{
                padding: '0.25rem 0.5rem',
                backgroundColor: '#28a745',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                fontSize: '0.75em',
                cursor: 'pointer'
              }}
            >
              accept
            </button>
            <button
              onClick={() => handleGuestRequest(request.id, false)}
              style={{
                padding: '0.25rem 0.5rem',
                backgroundColor: '#dc3545',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                fontSize: '0.75em',
                cursor: 'pointer'
              }}
            >
              deny
            </button>
          </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
