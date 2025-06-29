import { apiClient } from './apiClient'

export interface VideoStreamingOptions {
  movieId: string
  guestToken?: string
}

export interface VideoAccessResponse {
  movie_id: string
  hls_url: string
  expires_at: string
  cdn_info?: {
    cacheable: boolean
    cache_duration: string
  }
}

class VideoStreamingService {
  // get the video source URL using signed URLs (works for both SaaS and self-hosted)
  async getVideoSource(options: VideoStreamingOptions): Promise<string> {
    const { movieId, guestToken } = options
    
    // always use signed URLs for direct access to storage
    return this.getSignedStreamingURL(movieId, guestToken)
  }

  // get signed streaming URL (direct access to storage with signed URL)
  private async getSignedStreamingURL(movieId: string, guestToken?: string): Promise<string> {
    const endpoint = `/videos/${movieId}/hls`
    
    let response: VideoAccessResponse
    if (guestToken) {
      // use publicGet to avoid auth headers and use 'token' parameter as expected by backend
      response = await apiClient.publicGet<VideoAccessResponse>(`${endpoint}?token=${guestToken}`)
    } else {
      response = await apiClient.get<VideoAccessResponse>(endpoint)
    }
    
    return response.hls_url
  }

  // get batch URLs for HLS segments (used by video player for prefetching)
  async getBatchSegmentURLs(
    movieId: string, 
    files: string[], 
    guestToken?: string
  ): Promise<Record<string, string>> {
    // always use signed URLs for direct access to storage
    return this.getSignedSegmentURLs(movieId, files, guestToken)
  }

  // get signed segment URLs (direct access to storage with signed URLs)
  private async getSignedSegmentURLs(
    movieId: string, 
    files: string[], 
    guestToken?: string
  ): Promise<Record<string, string>> {
    const endpoint = `/videos/${movieId}/urls`
    
    const requestBody = { files }
    
    let response: {
      movie_id: string
      file_urls: Record<string, string>
      expires_at: string
    }
    
    if (guestToken) {
      // use the postWithGuestToken method for guest requests
      response = await apiClient.postWithGuestToken<{
        movie_id: string
        file_urls: Record<string, string>
        expires_at: string
      }>(endpoint, requestBody, guestToken)
    } else {
      response = await apiClient.post<{
        movie_id: string
        file_urls: Record<string, string>
        expires_at: string
      }>(endpoint, requestBody)
    }
    
    return response.file_urls
  }
}

export const videoStreamingService = new VideoStreamingService()
