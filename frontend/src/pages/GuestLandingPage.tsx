import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'

export default function GuestLandingPage() {
  const navigate = useNavigate()
  const [roomLink, setRoomLink] = useState('')

  const handleJoinRoom = () => {
    if (!roomLink.trim()) return

    try {
      // try to parse as a full URL
      const url = new URL(roomLink)
      const pathname = url.pathname
      
      // extract room ID from the path
      const roomIdMatch = pathname.match(/\/rooms\/join\/([^/]+)/)
      if (roomIdMatch) {
        navigate(`/rooms/join/${roomIdMatch[1]}`)
        return
      }
    } catch {
      // not a valid URL, treat as room ID
      const roomId = roomLink.trim()
      if (roomId) {
        navigate(`/rooms/join/${roomId}`)
      }
    }
  }

  return (
    <div style={{ 
      padding: '2rem',
      fontFamily: 'system-ui, sans-serif',
      maxWidth: '800px',
      margin: '0 auto'
    }}>
      {/* hero section */}
      <div style={{
        textAlign: 'center',
        padding: '4rem 2rem',
        backgroundColor: '#f8f9fa',
        borderRadius: '12px',
        marginBottom: '3rem'
      }}>
        <h1 style={{ 
          fontSize: '3em', 
          margin: '0 0 1rem 0', 
          color: '#333',
          fontWeight: 'bold'
        }}>
          Watch Together, Anywhere
        </h1>
        <p style={{ 
          fontSize: '1.3em', 
          color: '#666', 
          maxWidth: '600px',
          margin: '0 auto 2rem auto'
        }}>
          Join watch parties with friends and family. Synchronized video playback, 
          real-time interaction, and no account required for guests.
        </p>
        
        <div style={{ display: 'flex', gap: '1rem', justifyContent: 'center', flexWrap: 'wrap' }}>
          <Link
            to="/signup"
            style={{
              padding: '1rem 2rem',
              backgroundColor: '#007bff',
              color: 'white',
              textDecoration: 'none',
              borderRadius: '8px',
              fontSize: '1.1em',
              fontWeight: 'bold'
            }}
          >
            Create Account
          </Link>
          <Link
            to="/login"
            style={{
              padding: '1rem 2rem',
              backgroundColor: '#28a745',
              color: 'white',
              textDecoration: 'none',
              borderRadius: '8px',
              fontSize: '1.1em',
              fontWeight: 'bold'
            }}
          >
            Login
          </Link>
        </div>
      </div>

      {/* join as guest section - made more prominent */}
      <div style={{
        padding: '2.5rem',
        backgroundColor: '#e3f2fd',
        border: '2px solid #2196f3',
        borderRadius: '16px',
        marginBottom: '3rem',
        textAlign: 'center'
      }}>
        <h2 style={{ 
          color: '#1976d2', 
          margin: '0 0 1rem 0',
          fontSize: '2em',
          fontWeight: 'bold'
        }}>
          🎬 Join as Guest
        </h2>
        <p style={{ 
          color: '#1565c0', 
          fontSize: '1.2em',
          margin: '0 0 2rem 0',
          fontWeight: 'medium'
        }}>
          Have a room link or room ID? Join directly without creating an account!
        </p>
        
        <div style={{ 
          display: 'flex', 
          gap: '1rem', 
          maxWidth: '600px', 
          margin: '0 auto',
          flexWrap: 'wrap'
        }}>
          <input
            type="text"
            value={roomLink}
            onChange={(e) => setRoomLink(e.target.value)}
            placeholder="Paste room link or enter room ID..."
            style={{
              flex: 1,
              padding: '1rem',
              border: '2px solid #2196f3',
              borderRadius: '8px',
              fontSize: '1.1rem',
              minWidth: '300px'
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                handleJoinRoom()
              }
            }}
          />
          <button
            onClick={handleJoinRoom}
            disabled={!roomLink.trim()}
            style={{
              padding: '1rem 2rem',
              backgroundColor: roomLink.trim() ? '#2196f3' : '#6c757d',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              cursor: roomLink.trim() ? 'pointer' : 'not-allowed',
              fontSize: '1.1rem',
              fontWeight: 'bold',
              whiteSpace: 'nowrap'
            }}
          >
            Join Room
          </button>
        </div>
      </div>

      {/* features section */}
      <div>
        <h2 style={{ 
          color: '#333', 
          textAlign: 'center',
          margin: '0 0 2rem 0'
        }}>
          Why Watch Party?
        </h2>
        
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', 
          gap: '2rem'
        }}>
          <div style={{
            padding: '2rem',
            textAlign: 'center',
            backgroundColor: '#f8f9fa',
            borderRadius: '8px'
          }}>
            <div style={{ 
              fontSize: '3em', 
              marginBottom: '1rem'
            }}>
              🎬
            </div>
            <h3 style={{ color: '#333', margin: '0 0 1rem 0' }}>
              Synchronized Playback
            </h3>
            <p style={{ color: '#666', margin: 0 }}>
              Watch movies together with perfect synchronization. 
              When one person pauses, everyone pauses.
            </p>
          </div>

          <div style={{
            padding: '2rem',
            textAlign: 'center',
            backgroundColor: '#f8f9fa',
            borderRadius: '8px'
          }}>
            <div style={{ 
              fontSize: '3em', 
              marginBottom: '1rem'
            }}>
              👥
            </div>
            <h3 style={{ color: '#333', margin: '0 0 1rem 0' }}>
              Guest-Friendly
            </h3>
            <p style={{ color: '#666', margin: 0 }}>
              Friends can join without creating accounts. 
              Just share the room link and they're in.
            </p>
          </div>

          <div style={{
            padding: '2rem',
            textAlign: 'center',
            backgroundColor: '#f8f9fa',
            borderRadius: '8px'
          }}>
            <div style={{ 
              fontSize: '3em', 
              marginBottom: '1rem'
            }}>
              🔗
            </div>
            <h3 style={{ color: '#333', margin: '0 0 1rem 0' }}>
              Persistent Rooms
            </h3>
            <p style={{ color: '#666', margin: 0 }}>
              Room links work like Google Meet. 
              Share once and use anytime.
            </p>
          </div>
        </div>
      </div>

      {/* getting started section */}
      <div style={{
        marginTop: '3rem',
        padding: '2rem',
        backgroundColor: '#e7f3ff',
        border: '1px solid #b8daff',
        borderRadius: '12px',
        textAlign: 'center'
      }}>
        <h3 style={{ color: '#004085', margin: '0 0 1rem 0' }}>
          Getting Started
        </h3>
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
          gap: '1rem',
          marginTop: '1rem'
        }}>
          <div>
            <strong style={{ color: '#004085' }}>1. For Hosts</strong>
            <p style={{ margin: '0.5rem 0 0 0', color: '#004085' }}>
              Sign up → Upload movies → Create rooms → Invite friends
            </p>
          </div>
          <div>
            <strong style={{ color: '#004085' }}>2. For Guests</strong>
            <p style={{ margin: '0.5rem 0 0 0', color: '#004085' }}>
              Get room link → Request access → Start watching
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
