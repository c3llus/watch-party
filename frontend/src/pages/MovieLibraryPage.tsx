import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { movieService, type Movie } from '../services/movieService'

export default function MovieLibraryPage() {
  const [movies, setMovies] = useState<Movie[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalCount, setTotalCount] = useState(0)
  const [pageSize] = useState(10)

  const loadMovies = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await movieService.getMyMovies(currentPage, pageSize)
      setMovies(response.movies)
      setTotalCount(response.total_count)
    } catch (err) {
      console.error('failed to load movies:', err)
      setError(err instanceof Error ? err.message : 'Failed to load movies')
    } finally {
      setLoading(false)
    }
  }, [currentPage, pageSize])

  useEffect(() => {
    loadMovies()
  }, [loadMovies])

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDuration = (seconds: number) => {
    if (seconds === 0) return 'Unknown'
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    
    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`
    } else {
      return `${secs}s`
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'available': return '#28a745'
      case 'processing': return '#007bff'
      case 'transcoding': return '#fd7e14'
      case 'failed': return '#dc3545'
      default: return '#6c757d'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'available': return 'Available'
      case 'processing': return 'Processing'
      case 'transcoding': return 'Transcoding'
      case 'failed': return 'Failed'
      default: return status
    }
  }

  const totalPages = Math.ceil(totalCount / pageSize)

  if (loading && movies.length === 0) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        <p>Loading movies...</p>
      </div>
    )
  }

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui, sans-serif' }}>
      <div style={{ marginBottom: '2rem' }}>
        <Link to="/admin" style={{ color: '#007bff', textDecoration: 'none' }}>
          ← Back to Admin Dashboard
        </Link>
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem' }}>
        <div>
          <h1 style={{ color: '#333', margin: 0 }}>Movie Library</h1>
          <p style={{ color: '#666', margin: '0.5rem 0 0 0' }}>
            Manage your uploaded movies and their processing status
          </p>
        </div>
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
          + Upload New Movie
        </Link>
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

      {movies.length === 0 && !loading ? (
        <div style={{ 
          textAlign: 'center', 
          padding: '3rem',
          backgroundColor: '#f8f9fa',
          borderRadius: '12px',
          border: '1px solid #e9ecef'
        }}>
          <p style={{ fontSize: '1.2em', color: '#666', margin: '0 0 1rem 0' }}>
            No movies uploaded yet
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
            Upload Your First Movie
          </Link>
        </div>
      ) : (
        <>
          <div style={{ display: 'grid', gap: '1rem', marginBottom: '2rem' }}>
            {movies.map((movie) => (
              <div 
                key={movie.id}
                style={{
                  border: '2px solid #f0f0f0',
                  borderRadius: '12px',
                  padding: '1.5rem',
                  backgroundColor: '#fff'
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
                  <div style={{ flex: 1 }}>
                    <h3 style={{ margin: '0 0 0.5rem 0', color: '#333' }}>
                      {movie.title}
                    </h3>
                    {movie.description && (
                      <p style={{ margin: '0 0 0.5rem 0', color: '#666' }}>
                        {movie.description}
                      </p>
                    )}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', flexWrap: 'wrap' }}>
                      <span style={{ 
                        padding: '0.25rem 0.75rem',
                        backgroundColor: getStatusColor(movie.status),
                        color: 'white',
                        borderRadius: '20px',
                        fontSize: '0.8em',
                        fontWeight: 'bold'
                      }}>
                        {getStatusText(movie.status)}
                      </span>
                      <span style={{ color: '#666', fontSize: '0.9em' }}>
                        {formatFileSize(movie.file_size)}
                      </span>
                      {movie.duration_seconds > 0 && (
                        <span style={{ color: '#666', fontSize: '0.9em' }}>
                          {formatDuration(movie.duration_seconds)}
                        </span>
                      )}
                      <span style={{ color: '#666', fontSize: '0.9em' }}>
                        {new Date(movie.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    {movie.status === 'available' && (
                      <button
                        onClick={() => {
                          // TODO: integrate with room creation
                          alert('Room creation integration coming soon!')
                        }}
                        style={{
                          padding: '0.5rem 1rem',
                          backgroundColor: '#28a745',
                          color: 'white',
                          border: 'none',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          fontSize: '0.9em'
                        }}
                      >
                        Create Room
                      </button>
                    )}
                  </div>
                </div>

                {movie.status === 'failed' && movie.error_message && (
                  <div style={{ 
                    backgroundColor: '#f8d7da', 
                    padding: '0.75rem', 
                    borderRadius: '6px',
                    border: '1px solid #f5c6cb',
                    marginTop: '1rem'
                  }}>
                    <p style={{ margin: 0, color: '#721c24', fontSize: '0.9em' }}>
                      <strong>Error:</strong> {movie.error_message}
                    </p>
                  </div>
                )}

                {movie.status === 'transcoding' && (
                  <div style={{ 
                    backgroundColor: '#fff3cd', 
                    padding: '0.75rem', 
                    borderRadius: '6px',
                    border: '1px solid #ffeaa7',
                    marginTop: '1rem'
                  }}>
                    <p style={{ margin: 0, color: '#856404', fontSize: '0.9em' }}>
                      <strong>Processing:</strong> Video is being converted to streaming format. This may take several minutes.
                    </p>
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '1rem' }}>
              <button
                onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                disabled={currentPage <= 1}
                style={{
                  padding: '0.5rem 1rem',
                  backgroundColor: currentPage <= 1 ? '#ccc' : '#007bff',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  cursor: currentPage <= 1 ? 'not-allowed' : 'pointer'
                }}
              >
                Previous
              </button>
              
              <span style={{ color: '#666' }}>
                Page {currentPage} of {totalPages} ({totalCount} movies)
              </span>
              
              <button
                onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                disabled={currentPage >= totalPages}
                style={{
                  padding: '0.5rem 1rem',
                  backgroundColor: currentPage >= totalPages ? '#ccc' : '#007bff',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  cursor: currentPage >= totalPages ? 'not-allowed' : 'pointer'
                }}
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
