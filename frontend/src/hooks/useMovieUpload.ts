import { useState, useCallback } from 'react'
import { movieService, type MovieStatusResponse } from '../services/movieService'

export interface UploadProgress {
  movieId: string
  filename: string
  status: 'uploading' | 'processing' | 'transcoding' | 'available' | 'failed'
  uploadProgress: number
  processingStatus?: MovieStatusResponse
  error?: string
}

export function useMovieUpload() {
  const [uploads, setUploads] = useState<UploadProgress[]>([])
  const [isUploading, setIsUploading] = useState(false)

  // add new upload to the list
  const addUpload = useCallback((upload: UploadProgress) => {
    setUploads(prev => [...prev, upload])
  }, [])

  // update upload progress
  const updateUpload = useCallback((movieId: string, updates: Partial<UploadProgress>) => {
    setUploads(prev => 
      prev.map(upload => 
        upload.movieId === movieId ? { ...upload, ...updates } : upload
      )
    )
  }, [])

  // remove upload from list
  const removeUpload = useCallback((movieId: string) => {
    setUploads(prev => prev.filter(upload => upload.movieId !== movieId))
  }, [])

  // start polling for processing status
  const startStatusPolling = useCallback((movieId: string) => {
    const pollStatus = async () => {
      try {
        const status = await movieService.getMovieStatus(movieId)
        
        updateUpload(movieId, {
          status: status.status,
          processingStatus: status
        })

        // if still processing, continue polling
        if (status.status === 'processing' || status.status === 'transcoding') {
          setTimeout(pollStatus, 3000) // poll every 3 seconds
        }
      } catch (error) {
        console.error('failed to poll status:', error)
        updateUpload(movieId, {
          status: 'failed',
          error: error instanceof Error ? error.message : 'status polling failed'
        })
      }
    }

    // start polling after a short delay
    setTimeout(pollStatus, 2000)
  }, [updateUpload])

  // main upload function
  const uploadMovie = useCallback(async (
    file: File,
    title: string,
    description = ''
  ): Promise<string> => {
    setIsUploading(true)

    try {
      // step 1: initiate upload
      const uploadResponse = await movieService.initiateUpload({
        title,
        description,
        filename: file.name,
        filesize: file.size,
        mimetype: file.type
      })

      const movieId = uploadResponse.movie_id
      
      // add to upload list
      addUpload({
        movieId,
        filename: file.name,
        status: 'uploading',
        uploadProgress: 0
      })

      // step 2: upload file to storage
      await movieService.uploadFileToStorage(
        uploadResponse.signed_url,
        file,
        (progress) => {
          updateUpload(movieId, { uploadProgress: progress })
        }
      )

      // step 3: notify backend of upload completion (triggers transcoding)
      try {
        await movieService.notifyUploadComplete(movieId, uploadResponse.file_path)
      } catch (error) {
        console.error('failed to notify backend of upload completion:', error)
        updateUpload(movieId, {
          status: 'failed',
          error: `Upload completed but failed to start processing: ${error instanceof Error ? error.message : 'Unknown error'}`
        })
        throw error
      }

      // step 4: update status and start polling
      updateUpload(movieId, {
        status: 'processing',
        uploadProgress: 100
      })

      startStatusPolling(movieId)

      return movieId
    } catch (error) {
      console.error('upload failed:', error)
      throw error
    } finally {
      setIsUploading(false)
    }
  }, [addUpload, updateUpload, startStatusPolling])

  // clear all uploads
  const clearUploads = useCallback(() => {
    setUploads([])
  }, [])

  return {
    uploads,
    isUploading,
    uploadMovie,
    removeUpload,
    clearUploads
  }
}
