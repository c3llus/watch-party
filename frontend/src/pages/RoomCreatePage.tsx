import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { roomService } from '../services/roomService'
import { movieService, type Movie } from '../services/movieService'

export default function RoomCreatePage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  
  const [availableMovies, setAvailableMovies] = useState<Movie[]>([])
  const [selectedMovieId, setSelectedMovieId] = useState('')
  const [roomName, setRoomName] = useState('')
  const [roomDescription, setRoomDescription] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isCreating, setIsCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // check for pre-selected movie from URL
  const preSelectedMovieId = searchParams.get('movieId')

  // load available movies
  useEffect(() => {
    const loadMovies = async () => {
      try {
        setIsLoading(true)
        setError(null)
        
        // get all movies, then filter for available ones
        const response = await movieService.getMyMovies(1, 100) // get up to 100 movies
        const available = response.movies.filter(movie => movie.status === 'available')
        setAvailableMovies(available)
        
        // auto-select movie if provided in URL
        if (preSelectedMovieId && available.find(m => m.id === preSelectedMovieId)) {
          setSelectedMovieId(preSelectedMovieId)
          // auto-generate room name based on movie
          const selectedMovie = available.find(m => m.id === preSelectedMovieId)
          if (selectedMovie) {
            setRoomName(`${selectedMovie.title} - Watch Party`)
          }
        }
        
      } catch (err) {
        console.error('failed to load movies:', err)
        setError(err instanceof Error ? err.message : 'Failed to load movies')
      } finally {
        setIsLoading(false)
      }
    }

    loadMovies()
  }, [preSelectedMovieId])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!selectedMovieId || !roomName.trim()) return
    
    try {
      setIsCreating(true)
      setError(null)
      
      const room = await roomService.createRoom(
        selectedMovieId,
        roomName.trim(),
        roomDescription.trim() || undefined
      )
      
      // navigate to success page for sharing and invitations
      navigate(`/rooms/${room.id}/success`)
      
    } catch (err) {
      console.error('failed to create room:', err)
      setError(err instanceof Error ? err.message : 'Failed to create room')
    } finally {
      setIsCreating(false)
    }
  }

  const selectedMovie = availableMovies.find(movie => movie.id === selectedMovieId)

  if (isLoading) {
    return (
      <div style={{ 
        padding: '2rem', 
        textAlign: 'center',
        fontFamily: 'system-ui, sans-serif'
      }}>
        <p>Loading available movies...</p>
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

      <div style={{ marginBottom: '2rem' }}>
        <h1 style={{ color: '#333', margin: 0 }}>Create Watch Party Room</h1>
        <p style={{ color: '#666', margin: '0.5rem 0 0 0' }}>
          Create a new room for watching movies with others
        </p>
      </div>

      {error && (
        <div style={{ 
          padding: '1rem', 
          backgroundColor: '#f8d7da', 
          border: '1px solid #f5c6cb',
          borderRadius: '8px',
          color: '#721c24',
          marginBottom: '2rem'
        }}>
          {error}
        </div>
      )}

      {availableMovies.length === 0 ? (
        <div style={{ 
          textAlign: 'center', 
          padding: '3rem',
          backgroundColor: '#f8f9fa',
          borderRadius: '12px',
          border: '1px solid #e9ecef'
        }}>
          <p style={{ fontSize: '1.2em', color: '#666', margin: '0 0 1rem 0' }}>
            No movies available for rooms
          </p>
          <p style={{ color: '#666', margin: '0 0 2rem 0' }}>
            You need to upload and process movies before creating rooms.
          </p>
          <Link 
            to="/admin/upload"
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: '#007bff',
              color: 'white',
              textDecoration: 'none',
              borderRadius: '8px',
              fontWeight: 'bold'
            }}
          >
            Upload Movies
          </Link>
        </div>
      ) : (
        <form onSubmit={handleSubmit}>
          {/* movie selection */}
          <div style={{ marginBottom: '2rem' }}>
            <label 
              htmlFor="movieSelect"
              style={{ 
                display: 'block', 
                marginBottom: '0.5rem',
                fontWeight: 'bold',
                color: '#333'
              }}
            >
              Select Movie *
            </label>
            <select
              id="movieSelect"
              value={selectedMovieId}
              onChange={(e) => setSelectedMovieId(e.target.value)}
              required
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '1px solid #ccc',
                borderRadius: '4px',
                fontSize: '1rem',
                boxSizing: 'border-box'
              }}
            >
              <option value="">Choose a movie...</option>
              {availableMovies.map((movie) => (
                <option key={movie.id} value={movie.id}>
                  {movie.title} ({movie.duration_seconds ? `${Math.floor(movie.duration_seconds / 60)}min` : 'Unknown duration'})
                </option>
              ))}
            </select>
            
            {selectedMovie && (
              <div style={{
                marginTop: '1rem',
                padding: '1rem',
                backgroundColor: '#f8f9fa',
                border: '1px solid #e9ecef',
                borderRadius: '8px'
              }}>
                <h4 style={{ margin: '0 0 0.5rem 0', color: '#333' }}>
                  {selectedMovie.title}
                </h4>
                {selectedMovie.description && (
                  <p style={{ margin: '0 0 0.5rem 0', color: '#666', fontSize: '0.9em' }}>
                    {selectedMovie.description}
                  </p>
                )}
                <div style={{ fontSize: '0.8em', color: '#666' }}>
                  {selectedMovie.duration_seconds && (
                    <span>Duration: {Math.floor(selectedMovie.duration_seconds / 60)}min • </span>
                  )}
                  {selectedMovie.file_size && (
                    <span>Size: {(selectedMovie.file_size / (1024 * 1024 * 1024)).toFixed(2)} GB</span>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* room name */}
          <div style={{ marginBottom: '1.5rem' }}>
            <label 
              htmlFor="roomName"
              style={{ 
                display: 'block', 
                marginBottom: '0.5rem',
                fontWeight: 'bold',
                color: '#333'
              }}
            >
              Room Name *
            </label>
            <input
              type="text"
              id="roomName"
              value={roomName}
              onChange={(e) => setRoomName(e.target.value)}
              placeholder="e.g., Movie Night with Friends"
              required
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '1px solid #ccc',
                borderRadius: '4px',
                fontSize: '1rem',
                boxSizing: 'border-box'
              }}
            />
          </div>

          {/* room description */}
          <div style={{ marginBottom: '2rem' }}>
            <label 
              htmlFor="roomDescription"
              style={{ 
                display: 'block', 
                marginBottom: '0.5rem',
                fontWeight: 'bold',
                color: '#333'
              }}
            >
              Description (Optional)
            </label>
            <textarea
              id="roomDescription"
              value={roomDescription}
              onChange={(e) => setRoomDescription(e.target.value)}
              placeholder="Add a description for your watch party..."
              rows={3}
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '1px solid #ccc',
                borderRadius: '4px',
                fontSize: '1rem',
                resize: 'vertical',
                boxSizing: 'border-box'
              }}
            />
          </div>

          {/* submit buttons */}
          <div style={{ display: 'flex', gap: '1rem' }}>
            <button
              type="submit"
              disabled={isCreating || !selectedMovieId || !roomName.trim()}
              style={{
                flex: 1,
                padding: '0.75rem 1.5rem',
                backgroundColor: isCreating || !selectedMovieId || !roomName.trim() ? '#6c757d' : '#28a745',
                color: 'white',
                border: 'none',
                borderRadius: '8px',
                fontSize: '1rem',
                fontWeight: 'bold',
                cursor: isCreating || !selectedMovieId || !roomName.trim() ? 'not-allowed' : 'pointer'
              }}
            >
              {isCreating ? 'Creating Room...' : 'Create Room'}
            </button>
            
            <button
              type="button"
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
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* info section */}
      <div style={{
        marginTop: '3rem',
        padding: '2rem',
        backgroundColor: '#e7f3ff',
        border: '1px solid #b8daff',
        borderRadius: '12px'
      }}>
        <h3 style={{ margin: '0 0 1rem 0', color: '#004085' }}>
          How Room Creation Works
        </h3>
        <ul style={{ margin: 0, paddingLeft: '1.5rem', color: '#004085' }}>
          <li>Select a movie that has finished processing</li>
          <li>Give your room a name and optional description</li>
          <li>Once created, you'll get a persistent room link</li>
          <li>Share the link with friends to invite them</li>
          <li>Guests can request access even without accounts</li>
          <li>You can control who joins and manage the watch party</li>
        </ul>
      </div>
    </div>
  )
}
