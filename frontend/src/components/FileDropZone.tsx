import React from 'react'

interface FileDropZoneProps {
  selectedFile: File | null
  onFileSelect: (file: File) => void
  dragOver: boolean
  onDragOver: (e: React.DragEvent) => void
  onDragLeave: (e: React.DragEvent) => void
  onDrop: (e: React.DragEvent) => void
}

export function FileDropZone({
  selectedFile,
  onFileSelect,
  dragOver,
  onDragOver,
  onDragLeave,
  onDrop
}: FileDropZoneProps) {
  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (files && files.length > 0) {
      onFileSelect(files[0])
    }
  }

  return (
    <div
      onDrop={onDrop}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      style={{
        border: `2px dashed ${dragOver ? '#007bff' : '#ddd'}`,
        borderRadius: '12px',
        padding: '3rem',
        textAlign: 'center' as const,
        backgroundColor: dragOver ? '#f8f9fa' : '#fff',
        cursor: 'pointer',
        transition: 'all 0.3s ease'
      }}
      onClick={() => document.getElementById('fileInput')?.click()}
    >
      {selectedFile ? (
        <div>
          <p style={{ fontSize: '1.2em', margin: '0 0 0.5rem 0', color: '#333' }}>
            📹 <strong>{selectedFile.name}</strong>
          </p>
          <p style={{ color: '#666', margin: '0 0 0.5rem 0' }}>
            {formatFileSize(selectedFile.size)}
          </p>
          <p style={{ color: '#007bff', fontSize: '0.9em', margin: 0 }}>
            Click to select a different file
          </p>
        </div>
      ) : (
        <div>
          <p style={{ fontSize: '1.2em', margin: '0 0 0.5rem 0', color: '#333' }}>
            📁 Drag and drop a video file here
          </p>
          <p style={{ color: '#666', margin: '0 0 1rem 0' }}>
            or click to select a file
          </p>
          <p style={{ color: '#999', fontSize: '0.9em', margin: 0 }}>
            Supported formats: MP4, AVI, MOV, MKV, WebM, M4V (max 5GB)
          </p>
        </div>
      )}
      
      <input
        id="fileInput"
        type="file"
        accept="video/*"
        onChange={handleFileInput}
        style={{ display: 'none' }}
      />
    </div>
  )
}
