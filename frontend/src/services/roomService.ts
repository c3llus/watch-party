import { apiClient } from './apiClient'
import { videoStreamingService } from './videoStreamingService'
import { configService } from './configService'

export interface Room {
  id: string
  movie_id: string
  host_id: string
  name: string
  description?: string
  created_at: string
  room_code: string
  persistent_link: string
  movie?: {
    id: string
    title: string
    description?: string
    status: string
    hls_url?: string
  }
  host?: {
    id: string
    email: string
    role: string
    created_at: string
  }
  member_count?: number
}

export interface RoomParticipant {
  id: string
  name: string
  role: 'host' | 'user' | 'guest'
  connected_at: string
}

export interface GuestRequest {
  id: string
  room_id: string
  guest_name: string
  message?: string
  status: 'pending' | 'approved' | 'denied'
  created_at: string
}

export interface UserRoomAccessRequest {
  user_id: string
  room_id: string
  request_message: string
  status: 'requested' | 'approved' | 'denied'
  requested_at: string
  reviewed_by?: string
  reviewed_at?: string
}

export interface RoomState {
  room_id: string
  is_playing: boolean
  current_time: number
  participants: RoomParticipant[]
  last_action_by?: string
}

export interface VideoAccess {
  movie_id: string
  hls_url: string
  expires_at: string
  cdn_info?: {
    enabled: boolean
    base_url?: string
  }
}

class RoomService {
  // create a new room (admin only)
  async createRoom(movieId: string, name: string, description?: string): Promise<Room> {
    const response = await apiClient.post<{ room: Room; message: string }>('/rooms', {
      movie_id: movieId,
      name,
      description
    })
    return response.room
  }

  // get room details by ID
  async getRoom(roomId: string): Promise<Room> {
    return apiClient.get<Room>(`/rooms/${roomId}`)
  }

  // get room for joining (works for persistent links)
  async getRoomForJoin(roomId: string): Promise<Room> {
    const response = await apiClient.get<{room: Room, message: string}>(`/rooms/join/${roomId}`)
    return response.room
  }

  // get all rooms (admin only)
  async getRooms(): Promise<Room[]> {
    return apiClient.get<Room[]>('/rooms')
  }

  // invite user to room by email
  async inviteUser(roomId: string, email: string, message?: string): Promise<void> {
    return apiClient.post<void>(`/rooms/${roomId}/invite`, {
      email,
      message
    })
  }

  // request guest access
  async requestGuestAccess(roomId: string, guestName: string, message?: string): Promise<{ request_id: string }> {
    return apiClient.post<{ request_id: string }>(`/rooms/${roomId}/request-access`, {
      guest_name: guestName,
      message
    })
  }

  // get pending guest requests (host only)
  async getGuestRequests(roomId: string): Promise<GuestRequest[]> {
    const response = await apiClient.get<{ requests: GuestRequest[] }>(`/rooms/${roomId}/guest-requests`)
    return response.requests
  }

  // approve/deny guest request (host only)
  async respondToGuestRequest(roomId: string, requestId: string, approved: boolean): Promise<{ session_token?: string; expires_at?: string }> {
    return apiClient.post<{ session_token?: string; expires_at?: string }>(`/rooms/${roomId}/guest-requests/${requestId}/approve`, {
      approved
    })
  }

  // check guest request status (for polling)
  async checkGuestRequestStatus(requestId: string): Promise<{ status: 'pending' | 'approved' | 'denied'; session_token?: string; expires_at?: string }> {
    return apiClient.publicGet<{ status: 'pending' | 'approved' | 'denied'; session_token?: string; expires_at?: string }>(`/guest-requests/${requestId}/status`)
  }

  // request room access as authenticated user
  async requestRoomAccess(roomId: string, message?: string): Promise<{ status: string; message: string }> {
    return apiClient.post<{ status: string; message: string }>(`/rooms/${roomId}/room-access`, {
      request_message: message || ''
    })
  }

  // check room access request status for authenticated users
  async checkRoomAccessRequestStatus(roomId: string): Promise<{ status: 'pending' | 'approved' | 'denied' }> {
    return apiClient.get<{ status: 'pending' | 'approved' | 'denied' }>(`/rooms/${roomId}/room-access/status`)
  }

  // get pending room access requests (admin only)
  async getRoomAccessRequests(roomId: string): Promise<UserRoomAccessRequest[]> {
    const response = await apiClient.get<{ requests: UserRoomAccessRequest[] }>(`/rooms/${roomId}/room-access`)
    return response.requests
  }

  // approve/deny room access request (admin only)
  async respondToRoomAccessRequest(roomId: string, userId: string, approved: boolean): Promise<{ status: string; message: string }> {
    return apiClient.post<{ status: string; message: string }>(`/rooms/${roomId}/room-access/${userId}/approve`, {
      approved
    })
  }

  // get video access for streaming (deployment mode aware)
  async getVideoAccess(movieId: string, guestToken?: string): Promise<VideoAccess> {
    // use the new video streaming service that handles both CDN and direct modes
    const hlsUrl = await videoStreamingService.getVideoSource({ movieId, guestToken })
    
    return {
      movie_id: movieId,
      hls_url: hlsUrl,
      expires_at: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(), // 2 hours from now
      cdn_info: {
        enabled: await configService.isCDNEnabled(),
        base_url: undefined
      }
    }
  }

  // get batch signed URLs for video files (HLS segments) - deployment mode aware
  async getVideoFileURLs(movieId: string, files: string[], guestToken?: string): Promise<{
    movie_id: string
    file_urls: Record<string, string>
    expires_at: string
    cdn_info: {
      cacheable: boolean
      cache_duration: string
    }
  }> {
    // use the new video streaming service that handles both CDN and direct modes
    const fileUrls = await videoStreamingService.getBatchSegmentURLs(movieId, files, guestToken)
    const isCDNEnabled = await configService.isCDNEnabled()
    
    return {
      movie_id: movieId,
      file_urls: fileUrls,
      expires_at: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(), // 2 hours from now
      cdn_info: {
        cacheable: isCDNEnabled,
        cache_duration: isCDNEnabled ? "24h" : "5m"
      }
    }
  }

  // delete room (admin only)
  async deleteRoom(roomId: string): Promise<void> {
    return apiClient.delete<void>(`/rooms/${roomId}`)
  }

  // get room details for guests (requires guest token)
  async getRoomForGuest(roomId: string, guestToken: string): Promise<{ id: string; name: string; description?: string; movie?: { id: string; title: string; description: string } }> {
    return apiClient.publicGet<{ id: string; name: string; description?: string; movie?: { id: string; title: string; description: string } }>(`/guest/rooms/${roomId}?guestToken=${guestToken}`)
  }
}

export const roomService = new RoomService()
