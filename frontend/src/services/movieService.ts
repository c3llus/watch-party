import { apiClient } from './apiClient'

// types based on backend API
export interface Movie {
  id: string
  title: string
  description: string
  original_file_path: string
  transcoded_file_path: string
  hls_playlist_url: string
  duration_seconds: number
  file_size: number
  mime_type: string
  status: 'processing' | 'transcoding' | 'available' | 'failed'
  uploaded_by: string
  created_at: string
  processing_started_at?: string
  processing_ended_at?: string
  error_message?: string
}

export interface UploadMovieRequest {
  title: string
  description: string
  filename: string
  filesize: number
  mimetype?: string
}

export interface MovieUploadResponse {
  movie_id: string
  signed_url: string
  file_path: string  // add file path for webhook notification
  message: string
}

export interface MovieStatusResponse {
  movie_id: string
  status: 'processing' | 'transcoding' | 'available' | 'failed'
  title: string
  hls_playlist_url?: string
  processing_started_at?: string
  processing_ended_at?: string
  error_message?: string
}

export interface MovieListResponse {
  movies: Movie[]
  total_count: number
  page: number
  page_size: number
}

export class MovieService {
  // initiate asynchronous upload - returns signed URL
  async initiateUpload(request: UploadMovieRequest): Promise<MovieUploadResponse> {
    return apiClient.post<MovieUploadResponse>('/admin/movies', request)
  }

  // upload file directly to storage using signed URL
  async uploadFileToStorage(
    signedUrl: string, 
    file: File, 
    onProgress?: (progress: number) => void
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()

      // track upload progress
      if (onProgress) {
        xhr.upload.addEventListener('progress', (event) => {
          if (event.lengthComputable) {
            const progress = Math.round((event.loaded * 100) / event.total)
            onProgress(progress)
          }
        })
      }

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve()
        } else {
          reject(new Error(`upload failed with status ${xhr.status}`))
        }
      })

      xhr.addEventListener('error', () => {
        reject(new Error('network error during upload'))
      })

      xhr.addEventListener('abort', () => {
        reject(new Error('upload was aborted'))
      })

      // use PUT method for direct storage upload
      xhr.open('PUT', signedUrl)
      xhr.setRequestHeader('Content-Type', file.type)
      xhr.send(file)
    })
  }

  // poll movie processing status
  async getMovieStatus(movieId: string): Promise<MovieStatusResponse> {
    return apiClient.get<MovieStatusResponse>(`/admin/movies/${movieId}/status`)
  }

  // get movie details
  async getMovie(movieId: string): Promise<Movie> {
    return apiClient.get<Movie>(`/admin/movies/${movieId}`)
  }

  // get movies with pagination
  async getMovies(page = 1, pageSize = 20): Promise<MovieListResponse> {
    return apiClient.get<MovieListResponse>(`/admin/movies?page=${page}&page_size=${pageSize}`)
  }

  // get movies uploaded by current user
  async getMyMovies(page = 1, pageSize = 20): Promise<MovieListResponse> {
    return apiClient.get<MovieListResponse>(`/admin/movies?page=${page}&page_size=${pageSize}`)
  }

  // notify backend that upload is complete (triggers transcoding)
  async notifyUploadComplete(movieId: string, filePath: string): Promise<void> {
    return apiClient.post<void>('/webhooks/upload-complete', {
      movie_id: movieId,
      file_path: filePath
    })
  }

  // get streaming URL for movie
  async getStreamingUrl(movieId: string): Promise<{ stream_url: string }> {
    return apiClient.get<{ stream_url: string }>(`/admin/movies/${movieId}/stream`)
  }
}

export const movieService = new MovieService()
