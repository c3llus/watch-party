import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMovieUpload } from '../hooks/useMovieUpload'
import { FileDropZone, UploadStatusCard } from '../components'

export default function UploadPage() {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [movieTitle, setMovieTitle] = useState('')
  const [movieDescription, setMovieDescription] = useState('')
  const [dragOver, setDragOver] = useState(false)
  
  const { uploads, isUploading, uploadMovie, removeUpload, clearUploads } = useMovieUpload()

  const handleFileSelect = (file: File) => {
    if (file.type.startsWith('video/')) {
      setSelectedFile(file)
      setMovieTitle(file.name.replace(/\.[^/.]+$/, '')) // remove extension
    } else {
      alert('Please select a video file')
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    
    const files = Array.from(e.dataTransfer.files)
    if (files.length > 0) {
      handleFileSelect(files[0])
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
  }

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!selectedFile || !movieTitle.trim()) {
      alert('Please select a file and enter a title')
      return
    }

    try {
      await uploadMovie(selectedFile, movieTitle.trim(), movieDescription.trim())
      
      // reset form
      setSelectedFile(null)
      setMovieTitle('')
      setMovieDescription('')
      
      // clear file input
      const fileInput = document.getElementById('fileInput') as HTMLInputElement
      if (fileInput) {
        fileInput.value = ''
      }
    } catch (error) {
      console.error('upload failed:', error)
      alert(`Upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui, sans-serif' }}>
      <div style={{ marginBottom: '2rem' }}>
        <Link to="/admin" style={{ color: '#007bff', textDecoration: 'none' }}>
          ← Back to Admin Dashboard
        </Link>
      </div>

      <h1 style={{ color: '#333', marginBottom: '0.5rem' }}>Upload Movie</h1>
      <p style={{ color: '#666', marginBottom: '2rem' }}>
        Upload video files for watch parties. Files will be automatically processed for streaming.
      </p>

      <form onSubmit={handleUpload} style={{ marginBottom: '3rem' }}>
        {/* File Drop Zone */}
        <div style={{ marginBottom: '2rem' }}>
          <FileDropZone
            selectedFile={selectedFile}
            onFileSelect={handleFileSelect}
            dragOver={dragOver}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
          />
        </div>

        {/* Movie Title Input */}
        <div style={{ marginBottom: '1.5rem' }}>
          <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold', color: '#333' }}>
            Movie Title *
          </label>
          <input
            type="text"
            value={movieTitle}
            onChange={(e) => setMovieTitle(e.target.value)}
            placeholder="Enter movie title"
            style={{ 
              width: '100%', 
              padding: '0.75rem', 
              border: '2px solid #ddd',
              borderRadius: '8px',
              fontSize: '1rem',
              transition: 'border-color 0.3s ease'
            }}
            onFocus={(e) => e.target.style.borderColor = '#007bff'}
            onBlur={(e) => e.target.style.borderColor = '#ddd'}
            required
          />
        </div>

        {/* Movie Description Input */}
        <div style={{ marginBottom: '2rem' }}>
          <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 'bold', color: '#333' }}>
            Description (Optional)
          </label>
          <textarea
            value={movieDescription}
            onChange={(e) => setMovieDescription(e.target.value)}
            placeholder="Enter movie description"
            rows={3}
            style={{ 
              width: '100%', 
              padding: '0.75rem', 
              border: '2px solid #ddd',
              borderRadius: '8px',
              fontSize: '1rem',
              resize: 'vertical' as const,
              transition: 'border-color 0.3s ease'
            }}
            onFocus={(e) => e.target.style.borderColor = '#007bff'}
            onBlur={(e) => e.target.style.borderColor = '#ddd'}
          />
        </div>

        {/* Upload Button */}
        <button
          type="submit"
          disabled={!selectedFile || !movieTitle.trim() || isUploading}
          style={{
            padding: '1rem 2rem',
            backgroundColor: (!selectedFile || !movieTitle.trim() || isUploading) ? '#ccc' : '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '8px',
            cursor: (!selectedFile || !movieTitle.trim() || isUploading) ? 'not-allowed' : 'pointer',
            fontSize: '1.1em',
            fontWeight: 'bold',
            transition: 'background-color 0.3s ease'
          }}
        >
          {isUploading ? 'Uploading...' : 'Upload Movie'}
        </button>
      </form>

      {/* Upload Progress */}
      {uploads.length > 0 && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
            <h2 style={{ color: '#333', margin: 0 }}>Upload Status</h2>
            <button
              onClick={clearUploads}
              style={{
                padding: '0.5rem 1rem',
                backgroundColor: '#6c757d',
                color: 'white',
                border: 'none',
                borderRadius: '6px',
                cursor: 'pointer',
                fontSize: '0.9em'
              }}
            >
              Clear All
            </button>
          </div>
          
          <div style={{ display: 'grid', gap: '1rem' }}>
            {uploads.map((upload) => (
              <UploadStatusCard
                key={upload.movieId}
                upload={upload}
                onRemove={removeUpload}
              />
            ))}
          </div>
        </div>
      )}

      {/* Information Panel */}
      <div style={{ 
        marginTop: '3rem', 
        padding: '2rem', 
        backgroundColor: '#f8f9fa',
        borderRadius: '12px',
        border: '1px solid #e9ecef'
      }}>
        <h3 style={{ color: '#333', marginTop: 0 }}>Upload Information</h3>
        <ul style={{ color: '#666', lineHeight: '1.6' }}>
          <li><strong>Supported formats:</strong> MP4, AVI, MOV, MKV, WebM, M4V</li>
          {/* TODO: dynamic size */}
          <li><strong>Maximum file size:</strong> 5GB per file</li> 
          <li><strong>Processing:</strong> Videos are automatically converted to HLS format for streaming</li>
          <li><strong>Status updates:</strong> Upload status is updated in real-time during processing</li>
        </ul>
      </div>
    </div>
  )
}
