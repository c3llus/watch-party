import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import type { Room } from '../services/roomService'
import { authService } from '../services/authService'

export default function UserDashboardPage() {
  const [rooms, setRooms] = useState<Room[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [user] = useState(authService.getCurrentUser())

  useEffect(() => {
    // Note: For this PoC, there's no backend endpoint to list user's rooms
    // Users join rooms via shared links or room codes
    setRooms([])
    setIsLoading(false)
  }, [])

  if (isLoading) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>Loading your rooms...</p>
      </div>
    )
  }

  return (
    <div style={{ 
      padding: '2rem', 
      fontFamily: 'system-ui, sans-serif',
      maxWidth: '1200px',
      margin: '0 auto'
    }}>
      {/* header */}
      <div style={{ marginBottom: '2rem' }}>
        <h1 style={{ color: '#333', margin: 0 }}>
          Welcome, {user?.email || 'User'}!
        </h1>
        <p style={{ color: '#666', margin: '0.5rem 0 0 0' }}>
          Your watch party rooms and invitations
        </p>
      </div>

      {/* rooms section */}
      <div style={{ marginBottom: '3rem' }}>
        <h2 style={{ color: '#333', margin: '0 0 1rem 0' }}>
          Available Rooms
        </h2>
        
        {rooms.length === 0 ? (
          <div style={{ 
            textAlign: 'center', 
            padding: '3rem',
            backgroundColor: '#f8f9fa',
            borderRadius: '12px',
            border: '1px solid #e9ecef'
          }}>
            <p style={{ fontSize: '1.2em', color: '#666', margin: '0 0 1rem 0' }}>
              No rooms joined yet
            </p>
            <p style={{ color: '#666', margin: '0 0 2rem 0' }}>
              Join a room using a link or room code, or ask an admin to invite you.
            </p>
            <div style={{ display: 'flex', gap: '1rem', justifyContent: 'center', flexWrap: 'wrap' }}>
              <button
                onClick={() => {
                  const roomId = prompt('Enter room ID:')
                  if (roomId?.trim()) {
                    // redirect to join page with room ID
                    window.location.href = `/rooms/join/${roomId.trim()}`
                  }
                }}
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
                Join with Room ID
              </button>
              <button
                onClick={() => {
                  const link = prompt('Paste room link:')
                  if (link?.trim()) {
                    try {
                      const url = new URL(link.trim())
                      window.location.href = url.pathname + url.search
                    } catch {
                      alert('Invalid room link')
                    }
                  }
                }}
                style={{
                  padding: '0.75rem 1.5rem',
                  backgroundColor: '#28a745',
                  color: 'white',
                  border: 'none',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: 'bold'
                }}
              >
                Join with Link
              </button>
            </div>
          </div>
        ) : (
          <div style={{ 
            display: 'grid', 
            gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', 
            gap: '1.5rem'
          }}>
            {rooms.map((room) => (
              <div
                key={room.id}
                style={{
                  padding: '1.5rem',
                  backgroundColor: '#fff',
                  border: '1px solid #e9ecef',
                  borderRadius: '12px',
                  boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
                }}
              >
                <h3 style={{ margin: '0 0 0.5rem 0', color: '#333' }}>
                  {room.name}
                </h3>
                
                {room.description && (
                  <p style={{ 
                    margin: '0 0 1rem 0', 
                    color: '#666',
                    fontSize: '0.9em'
                  }}>
                    {room.description}
                  </p>
                )}

                {room.movie && (
                  <div style={{
                    padding: '0.75rem',
                    backgroundColor: '#f8f9fa',
                    borderRadius: '6px',
                    marginBottom: '1rem'
                  }}>
                    <div style={{ fontWeight: 'bold', color: '#333' }}>
                      {room.movie.title}
                    </div>
                    {room.movie.description && (
                      <div style={{ 
                        fontSize: '0.85em', 
                        color: '#666',
                        marginTop: '0.25rem'
                      }}>
                        {room.movie.description}
                      </div>
                    )}
                    <div style={{
                      fontSize: '0.75em',
                      color: room.movie.status === 'available' ? '#28a745' : '#6c757d',
                      marginTop: '0.5rem',
                      fontWeight: 'bold'
                    }}>
                      Status: {room.movie.status === 'available' ? 'Ready to Watch' : 'Processing'}
                    </div>
                  </div>
                )}

                <div style={{ 
                  display: 'flex', 
                  gap: '0.75rem',
                  marginTop: '1rem'
                }}>
                  <Link
                    to={`/rooms/join/${room.id}`}
                    style={{
                      flex: 1,
                      padding: '0.75rem',
                      backgroundColor: room.movie?.status === 'available' ? '#007bff' : '#6c757d',
                      color: 'white',
                      textDecoration: 'none',
                      borderRadius: '6px',
                      textAlign: 'center',
                      fontWeight: 'bold',
                      fontSize: '0.9em'
                    }}
                  >
                    {room.movie?.status === 'available' ? 'Join Room' : 'Not Ready'}
                  </Link>
                  
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(`${window.location.origin}/rooms/join/${room.id}`)
                      alert('Room link copied to clipboard!')
                    }}
                    style={{
                      padding: '0.75rem',
                      backgroundColor: '#6c757d',
                      color: 'white',
                      border: 'none',
                      borderRadius: '6px',
                      cursor: 'pointer',
                      fontSize: '0.9em'
                    }}
                    title="Copy room link"
                  >
                    📋
                  </button>
                </div>

                <div style={{
                  marginTop: '1rem',
                  fontSize: '0.75em',
                  color: '#6c757d'
                }}>
                  Room Code: {room.room_code}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* quick actions */}
      <div style={{
        padding: '2rem',
        backgroundColor: '#f8f9fa',
        borderRadius: '12px',
        border: '1px solid #e9ecef'
      }}>
        <h2 style={{ color: '#333', margin: '0 0 1rem 0' }}>
          Quick Actions
        </h2>
        
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
          gap: '1rem'
        }}>
          <div style={{
            padding: '1.5rem',
            backgroundColor: '#fff',
            borderRadius: '8px',
            textAlign: 'center'
          }}>
            <h3 style={{ margin: '0 0 0.5rem 0', color: '#333' }}>
              Join with Room ID
            </h3>
            <p style={{ 
              margin: '0 0 1rem 0', 
              color: '#666',
              fontSize: '0.9em'
            }}>
              Enter a room ID to join directly
            </p>
            <button
              onClick={() => {
                const roomId = prompt('Enter room ID:')
                if (roomId?.trim()) {
                  window.location.href = `/rooms/join/${roomId.trim()}`
                }
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
              Enter Room ID
            </button>
          </div>

          <div style={{
            padding: '1.5rem',
            backgroundColor: '#fff',
            borderRadius: '8px',
            textAlign: 'center'
          }}>
            <h3 style={{ margin: '0 0 0.5rem 0', color: '#333' }}>
              Join with Link
            </h3>
            <p style={{ 
              margin: '0 0 1rem 0', 
              color: '#666',
              fontSize: '0.9em'
            }}>
              Paste a room link to join
            </p>
            <button
              onClick={() => {
                const link = prompt('Paste room link:')
                if (link) {
                  try {
                    const url = new URL(link)
                    window.location.href = url.pathname + url.search
                  } catch {
                    alert('Invalid room link')
                  }
                }
              }}
              style={{
                padding: '0.5rem 1rem',
                backgroundColor: '#28a745',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer'
              }}
            >
              Join Link
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
