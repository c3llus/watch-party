import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { movieService, type Movie } from '../services/movieService'

export default function AdminDashboardPage() {
  const [movies, setMovies] = useState<Movie[]>([])
  const [isLoadingMovies, setIsLoadingMovies] = useState(true)

  // load movies
  useEffect(() => {
    const loadMovies = async () => {
      try {
        setIsLoadingMovies(true)
        const response = await movieService.getMovies()
        setMovies(response.movies || [])
      } catch (err) {
        console.error('failed to load movies:', err)
        setMovies([])
      } finally {
        setIsLoadingMovies(false)
      }
    }

    loadMovies()
  }, [])

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui, sans-serif' }}>
      <div style={{ marginBottom: '2rem' }}>
        <h1 style={{ color: '#333', margin: 0 }}>Admin Dashboard</h1>
        <p style={{ color: '#666', margin: '0.5rem 0 0 0' }}>
          Manage movies, rooms, and watch parties
        </p>
      </div>

      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', 
        gap: '2rem',
        maxWidth: '1200px'
      }}>
        {/* movie management */}
        <div style={{
          padding: '2rem',
          backgroundColor: '#fff',
          border: '1px solid #e9ecef',
          borderRadius: '12px',
          boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
        }}>
          <h2 style={{ color: '#333', margin: '0 0 1rem 0', fontSize: '1.5em' }}>
            Movie Management
          </h2>
          <p style={{ color: '#666', margin: '0 0 1.5rem 0' }}>
            Upload and manage your video library
          </p>
          
          {/* movie status display */}
          {isLoadingMovies ? (
            <div style={{ margin: '1rem 0', color: '#666' }}>Loading movies...</div>
          ) : movies.length === 0 ? (
            <div style={{ 
              margin: '1rem 0', 
              padding: '1rem',
              backgroundColor: '#f8f9fa',
              border: '1px solid #e9ecef',
              borderRadius: '8px',
              color: '#666',
              textAlign: 'center'
            }}>
              No movies uploaded yet. Upload your first movie to get started.
            </div>
          ) : (
            <div style={{ margin: '1rem 0' }}>
              <div style={{ fontSize: '0.875em', color: '#666', marginBottom: '0.5rem' }}>
                {movies.length} movie{movies.length !== 1 ? 's' : ''} uploaded
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', maxHeight: '200px', overflowY: 'auto' }}>
                {movies.slice(0, 5).map((movie) => (
                  <div key={movie.id} style={{
                    padding: '0.5rem',
                    backgroundColor: '#f8f9fa',
                    borderRadius: '4px',
                    fontSize: '0.875em'
                  }}>
                    <div style={{ fontWeight: 'bold', color: 'black' }}>{movie.title}</div>
                    <div style={{ color: '#666' }}>
                      Status: <span style={{ 
                        color: movie.status === 'available' ? '#28a745' : 
                              movie.status === 'failed' ? '#dc3545' : '#ffc107'
                      }}>
                        {movie.status}
                      </span>
                    </div>
                  </div>
                ))}
                {movies.length > 5 && (
                  <div style={{ textAlign: 'center', color: '#666', fontSize: '0.875em' }}>
                    and {movies.length - 5} more...
                  </div>
                )}
              </div>
            </div>
          )}
          
          <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
            <Link 
              to="/admin/upload" 
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#007bff',
                color: 'white',
                textDecoration: 'none',
                borderRadius: '8px',
                fontWeight: 'bold',
                display: 'inline-block'
              }}
            >
              Upload Movie
            </Link>
            <Link 
              to="/admin/movies" 
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#6c757d',
                color: 'white',
                textDecoration: 'none',
                borderRadius: '8px',
                fontWeight: 'bold',
                display: 'inline-block'
              }}
            >
              View Library
            </Link>
          </div>
        </div>

      </div>

      {/* quick stats section */}
      <div style={{ 
        marginTop: '3rem',
        padding: '2rem',
        backgroundColor: '#f8f9fa',
        borderRadius: '12px',
        border: '1px solid #e9ecef'
      }}>
        <h2 style={{ color: '#333', margin: '0 0 1rem 0', fontSize: '1.5em' }}>
          Quick Stats
        </h2>
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
          gap: '1rem'
        }}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '2em', fontWeight: 'bold', color: '#007bff' }}>--</div>
            <div style={{ color: '#666' }}>Total Movies</div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '2em', fontWeight: 'bold', color: '#28a745' }}>--</div>
            <div style={{ color: '#666' }}>Active Rooms</div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '2em', fontWeight: 'bold', color: '#fd7e14' }}>--</div>
            <div style={{ color: '#666' }}>Processing Videos</div>
          </div>
        </div>
      </div>
    </div>
  )
}
