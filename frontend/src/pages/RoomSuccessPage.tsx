import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { roomService, type Room } from '../services/roomService'

export default function RoomSuccessPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const navigate = useNavigate()
  
  const [room, setRoom] = useState<Room | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteMessage, setInviteMessage] = useState('')
  const [isSendingInvite, setIsSendingInvite] = useState(false)
  const [inviteSuccess, setInviteSuccess] = useState<string | null>(null)
  const [inviteError, setInviteError] = useState<string | null>(null)
  const [copiedLink, setCopiedLink] = useState(false)

  // load room details
  useEffect(() => {
    const loadRoom = async () => {
      if (!roomId) return
      
      try {
        setIsLoading(true)
        setError(null)
        const roomData = await roomService.getRoom(roomId)
        setRoom(roomData)
      } catch (err) {
        console.error('failed to load room:', err)
        setError(err instanceof Error ? err.message : 'Failed to load room')
      } finally {
        setIsLoading(false)
      }
    }

    loadRoom()
  }, [roomId])

  const handleInviteSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!roomId || !inviteEmail.trim()) return
    
    try {
      setIsSendingInvite(true)
      setInviteError(null)
      setInviteSuccess(null)
      
      await roomService.inviteUser(
        roomId,
        inviteEmail.trim(),
        inviteMessage.trim() || undefined
      )
      
      setInviteSuccess(`Invitation sent to ${inviteEmail}`)
      setInviteEmail('')
      setInviteMessage('')
      
    } catch (err) {
      console.error('failed to send invite:', err)
      setInviteError(err instanceof Error ? err.message : 'Failed to send invitation')
    } finally {
      setIsSendingInvite(false)
    }
  }

  const copyRoomLink = () => {
    if (!room) return
    
    const roomUrl = `${window.location.origin}/rooms/join/${room.id}`
    navigator.clipboard.writeText(roomUrl).then(() => {
      setCopiedLink(true)
      setTimeout(() => setCopiedLink(false), 2000)
    }).catch(() => {
      // fallback for older browsers
      const textArea = document.createElement('textarea')
      textArea.value = roomUrl
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
      setCopiedLink(true)
      setTimeout(() => setCopiedLink(false), 2000)
    })
  }

  if (isLoading) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>Loading room details...</p>
      </div>
    )
  }

  if (error || !room) {
    return (
      <div style={{ 
        padding: '2rem',
        fontFamily: 'system-ui, sans-serif',
        maxWidth: '800px',
        margin: '0 auto'
      }}>
        <div style={{
          padding: '1rem',
          backgroundColor: '#f8d7da',
          border: '1px solid #f5c6cb',
          borderRadius: '8px',
          color: '#721c24',
          marginBottom: '1rem'
        }}>
          {error || 'Room not found'}
        </div>
        <Link to="/admin" style={{ color: '#007bff', textDecoration: 'none' }}>
          ← Back to Admin Dashboard
        </Link>
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
      <div style={{ marginBottom: '2rem' }}>
        <Link to="/admin" style={{ color: '#007bff', textDecoration: 'none' }}>
          ← Back to Admin Dashboard
        </Link>
      </div>

      {/* success header */}
      <div style={{
        textAlign: 'center',
        marginBottom: '3rem',
        padding: '2rem',
        backgroundColor: '#d4edda',
        border: '1px solid #c3e6cb',
        borderRadius: '12px',
        color: '#155724'
      }}>
        <h1 style={{ margin: '0 0 1rem 0', color: '#155724' }}>
          🎉 Room Created Successfully!
        </h1>
        <h2 style={{ margin: '0 0 0.5rem 0', color: '#155724' }}>
          {room.name}
        </h2>
        {room.description && (
          <p style={{ margin: '0', color: '#155724', opacity: 0.8 }}>
            {room.description}
          </p>
        )}
      </div>

      {/* room link sharing */}
      <div style={{
        marginBottom: '2rem',
        padding: '1.5rem',
        backgroundColor: '#fff',
        border: '1px solid #e9ecef',
        borderRadius: '12px'
      }}>
        <h3 style={{ margin: '0 0 1rem 0', color: '#333' }}>
          Share Room Link
        </h3>
        <div style={{
          display: 'flex',
          gap: '0.5rem',
          marginBottom: '1rem'
        }}>
          <input
            type="text"
            value={`${window.location.origin}/rooms/join/${room.id}`}
            readOnly
            style={{
              flex: 1,
              padding: '0.75rem',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: '#f8f9fa',
              color: '#495057'
            }}
          />
          <button
            onClick={copyRoomLink}
            style={{
              padding: '0.75rem 1rem',
              backgroundColor: copiedLink ? '#28a745' : '#007bff',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              whiteSpace: 'nowrap'
            }}
          >
            {copiedLink ? 'Copied!' : 'Copy Link'}
          </button>
        </div>
        <p style={{ margin: 0, fontSize: '0.875em', color: '#666' }}>
          Share this link with anyone you want to invite to the watch party. 
          Users with accounts can join directly, while guests will need to request access.
        </p>
      </div>

      {/* email invitation */}
      <div style={{
        marginBottom: '2rem',
        padding: '1.5rem',
        backgroundColor: '#fff',
        border: '1px solid #e9ecef',
        borderRadius: '12px'
      }}>
        <h3 style={{ margin: '0 0 1rem 0', color: '#333' }}>
          Send Email Invitation
        </h3>
        
        {inviteSuccess && (
          <div style={{
            padding: '0.75rem',
            backgroundColor: '#d4edda',
            border: '1px solid #c3e6cb',
            borderRadius: '4px',
            color: '#155724',
            marginBottom: '1rem'
          }}>
            {inviteSuccess}
          </div>
        )}
        
        {inviteError && (
          <div style={{
            padding: '0.75rem',
            backgroundColor: '#f8d7da',
            border: '1px solid #f5c6cb',
            borderRadius: '4px',
            color: '#721c24',
            marginBottom: '1rem'
          }}>
            {inviteError}
          </div>
        )}

        <form onSubmit={handleInviteSubmit}>
          <div style={{ marginBottom: '1rem' }}>
            <label style={{ 
              display: 'block', 
              marginBottom: '0.5rem',
              fontWeight: 'bold',
              color: '#333'
            }}>
              Email Address *
            </label>
            <input
              type="email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder="friend@example.com"
              required
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '1px solid #ccc',
                borderRadius: '4px',
                boxSizing: 'border-box'
              }}
            />
          </div>

          <div style={{ marginBottom: '1rem' }}>
            <label style={{ 
              display: 'block', 
              marginBottom: '0.5rem',
              fontWeight: 'bold',
              color: '#333'
            }}>
              Personal Message (optional)
            </label>
            <textarea
              value={inviteMessage}
              onChange={(e) => setInviteMessage(e.target.value)}
              placeholder="Hey! Want to watch a movie together?"
              rows={3}
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

          <button
            type="submit"
            disabled={isSendingInvite || !inviteEmail.trim()}
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: isSendingInvite || !inviteEmail.trim() ? '#6c757d' : '#007bff',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: isSendingInvite || !inviteEmail.trim() ? 'not-allowed' : 'pointer'
            }}
          >
            {isSendingInvite ? 'Sending...' : 'Send Invitation'}
          </button>
        </form>
      </div>

      {/* action buttons */}
      <div style={{
        display: 'flex',
        gap: '1rem',
        justifyContent: 'center'
      }}>
        <button
          onClick={() => navigate(`/rooms/${room.id}`)}
          style={{
            padding: '0.75rem 1.5rem',
            backgroundColor: '#28a745',
            color: 'white',
            border: 'none',
            borderRadius: '8px',
            fontSize: '1rem',
            fontWeight: 'bold',
            cursor: 'pointer'
          }}
        >
          Join Room Now
        </button>
        
        <button
          onClick={() => navigate('/admin')}
          style={{
            padding: '0.75rem 1.5rem',
            backgroundColor: '#6c757d',
            color: 'white',
            border: 'none',
            borderRadius: '8px',
            fontSize: '1rem',
            cursor: 'pointer'
          }}
        >
          Back to Dashboard
        </button>
      </div>
    </div>
  )
}
