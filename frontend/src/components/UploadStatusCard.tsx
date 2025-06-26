import type { UploadProgress } from '../hooks/useMovieUpload'

interface UploadStatusCardProps {
  upload: UploadProgress
  onRemove: (movieId: string) => void
}

export function UploadStatusCard({ upload, onRemove }: UploadStatusCardProps) {
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'uploading': return '📤'
      case 'processing': return '⚙️'
      case 'transcoding': return '🎬'
      case 'available': return '✅'
      case 'failed': return '❌'
      default: return '⏳'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'uploading': return 'Uploading to storage...'
      case 'processing': return 'Processing upload...'
      case 'transcoding': return 'Converting to streaming format...'
      case 'available': return 'Ready for streaming'
      case 'failed': return 'Upload failed'
      default: return 'Unknown status'
    }
  }

  return (
    <div 
      style={{ 
        border: '2px solid #f0f0f0', 
        padding: '1.5rem', 
        borderRadius: '12px',
        backgroundColor: '#fafafa'
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <div>
          <span style={{ fontSize: '1.1em', fontWeight: 'bold', color: '#333' }}>
            {upload.filename}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          <span style={{ fontSize: '1.2em' }}>
            {getStatusIcon(upload.status)}
          </span>
          <span style={{ 
            color: upload.status === 'available' ? '#28a745' : 
                   upload.status === 'failed' ? '#dc3545' : '#007bff',
            fontWeight: 'bold'
          }}>
            {getStatusText(upload.status)}
          </span>
          <button
            onClick={() => onRemove(upload.movieId)}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: '#6c757d',
              fontSize: '1.2em'
            }}
            title="Remove from list"
          >
            ×
          </button>
        </div>
      </div>
      
      {upload.status === 'uploading' && (
        <div style={{ marginBottom: '0.5rem' }}>
          <div style={{ 
            width: '100%', 
            height: '12px', 
            backgroundColor: '#e9ecef',
            borderRadius: '6px',
            overflow: 'hidden'
          }}>
            <div style={{
              width: `${upload.uploadProgress}%`,
              height: '100%',
              backgroundColor: '#007bff',
              transition: 'width 0.3s ease',
              borderRadius: '6px'
            }}></div>
          </div>
          <p style={{ fontSize: '0.9em', marginTop: '0.5rem', color: '#666', margin: 0 }}>
            {upload.uploadProgress}% uploaded to storage
          </p>
        </div>
      )}
      
      {upload.status === 'processing' && (
        <p style={{ fontSize: '0.9em', color: '#666', margin: 0 }}>
          File uploaded successfully. Processing started...
        </p>
      )}
      
      {upload.status === 'transcoding' && (
        <p style={{ fontSize: '0.9em', color: '#666', margin: 0 }}>
          Converting video to streaming format. This may take several minutes depending on video length.
        </p>
      )}

      {upload.status === 'available' && upload.processingStatus && (
        <div style={{ backgroundColor: '#d4edda', padding: '1rem', borderRadius: '8px', border: '1px solid #c3e6cb' }}>
          <p style={{ fontSize: '0.9em', color: '#155724', margin: '0 0 0.5rem 0', fontWeight: 'bold' }}>
            ✅ Video processing completed successfully!
          </p>
          <p style={{ fontSize: '0.8em', color: '#155724', margin: 0 }}>
            Your movie is now ready for streaming and can be used in watch party rooms.
          </p>
        </div>
      )}
      
      {upload.status === 'failed' && (
        <div style={{ backgroundColor: '#f8d7da', padding: '1rem', borderRadius: '8px', border: '1px solid #f5c6cb' }}>
          <p style={{ fontSize: '0.9em', color: '#721c24', margin: '0 0 0.5rem 0', fontWeight: 'bold' }}>
            ❌ Upload or processing failed
          </p>
          <p style={{ fontSize: '0.8em', color: '#721c24', margin: 0 }}>
            {upload.error || upload.processingStatus?.error_message || 'Unknown error occurred. Please try again.'}
          </p>
        </div>
      )}
    </div>
  )
}
