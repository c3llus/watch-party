import { config } from '../config/environment'

export interface SyncAction {
  action: 'play' | 'pause' | 'seek'
  currentTime?: number
  timestamp: number
  userId?: string  // for authenticated users
  guestName?: string  // for guest users
}

// backend sync message format (what we receive from backend)
export interface BackendSyncMessage {
  action: 'play' | 'pause' | 'seek'
  current_time: number
  timestamp: string // ISO timestamp from backend
  user_id?: string
  username?: string
}

export interface WebSocketMessage {
  type: 'sync' | 'participants' | 'state' | 'guest_request' | 'guest_approved' | 'error' | 'connected' | 'disconnected' | 'request_state' | 'provide_state'
  payload?: unknown  // backend uses 'payload' instead of 'data'
  data?: unknown     // keep for backwards compatibility
  room_id?: string
  user_id?: string
  guest_name?: string
}

export interface SyncState {
  is_playing: boolean
  current_time: number
  last_action_by?: string
  last_action_at: number
}

// backend room state format (matches backend RoomState struct)
export interface BackendRoomState {
  room_id: string
  is_playing: boolean
  current_time: number
  duration: number
  playback_rate: number
  last_updated: string
  updated_by: string
}

type WebSocketEventHandler = (message: WebSocketMessage) => void

class WebSocketService {
  private ws: WebSocket | null = null
  private roomId: string | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private isConnecting = false
  private eventHandlers: Map<string, WebSocketEventHandler[]> = new Map()

  // connect to room WebSocket
  async connect(roomId: string, token?: string, guestToken?: string, guestName?: string): Promise<void> {
    if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
      return
    }

    this.isConnecting = true
    this.roomId = roomId

    try {
      // build WebSocket URL with authentication
      let wsUrl = `${config.wsUrl}/ws/room/${roomId}`
      const params = new URLSearchParams()

      if (token) {
        params.append('token', token)
      }
      if (guestToken) {
        params.append('guestToken', guestToken)
      }
      if (guestName) {
        params.append('guestName', guestName)
      }

      if (params.toString()) {
        wsUrl += `?${params.toString()}`
      }

      this.ws = new WebSocket(wsUrl)
      
      this.ws.onopen = () => {
        console.log('websocket connected to room:', roomId)
        this.isConnecting = false
        this.reconnectAttempts = 0
        this.emit('connected', { type: 'connected' })
        
        // note: backend automatically sends initial room state, so we don't need to request it here
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          
          console.log('websocket received message:', message.type, message)
          
          // validate message structure
          if (!message || typeof message !== 'object') {
            console.warn('received invalid websocket message structure:', message)
            return
          }
          
          if (!message.type) {
            console.warn('received websocket message without type field:', message)
            return
          }
          
          this.handleMessage(message)
        } catch (error) {
          console.error('failed to parse websocket message:', error, 'raw data:', event.data)
        }
      }

      this.ws.onclose = (event) => {
        this.isConnecting = false
        this.ws = null
        
        this.emit('disconnected', { type: 'disconnected' })
        
        // attempt to reconnect if not a clean close
        if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect()
        }
      }

      this.ws.onerror = (error) => {
        console.error('websocket error:', error)
        this.isConnecting = false
        this.emit('error', { type: 'error', data: error })
      }

    } catch (error) {
      this.isConnecting = false
      throw error
    }
  }

  // disconnect from WebSocket
  disconnect(): void {
    if (this.ws) {
      this.ws.close(1000, 'user disconnected')
      this.ws = null
    }
    this.roomId = null
    this.reconnectAttempts = 0
  }

  // send sync action to other participants
  sendSyncAction(action: SyncAction): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('websocket not connected, cannot send sync action')
      return
    }

    // construct message in backend SyncMessage format
    const syncMessage = {
      action: action.action,
      data: {
        current_time: action.currentTime || 0,
        timestamp: action.timestamp
      }
    }

    console.log('sending sync action:', syncMessage)
    this.ws.send(JSON.stringify(syncMessage))
    console.log('sync action sent successfully')
  }

  // send chat message (if implemented)
  sendChatMessage(message: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('websocket not connected, cannot send chat message')
      return
    }

    const wsMessage = {
      type: 'chat',
      data: { message }
    }

    this.ws.send(JSON.stringify(wsMessage))
  }

  // request current room state from backend
  requestRoomState(): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('websocket not connected, cannot request room state')
      return
    }

    const message = {
      type: 'request_state'
    }

    console.log('📤 REQUESTING ROOM STATE from backend:', message)
    this.ws.send(JSON.stringify(message))
  }

  // provide current video state when requested by backend (for newly joined users)
  provideCurrentState(requesterID: string, currentState: { isPlaying: boolean; currentTime: number }): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('websocket not connected, cannot provide state')
      return
    }

    const statePayload = {
      room_id: this.roomId,
      is_playing: currentState.isPlaying,
      current_time: currentState.currentTime,
      last_updated: new Date().toISOString(),
      updated_by: 'existing_user', // the existing user providing the state
      duration: 0,
      playback_rate: 1
    }

    const message = {
      type: 'provide_state',
      requester_id: requesterID,
      state: statePayload
    }

    console.log('sending live state to backend for requester:', requesterID)
    console.log('state being sent:', statePayload)
    console.log('full message:', message)
    this.ws.send(JSON.stringify(message))
    console.log('live state message sent to backend')
  }

  // event handling
  on(event: string, handler: WebSocketEventHandler): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, [])
    }
    this.eventHandlers.get(event)!.push(handler)
  }

  off(event: string, handler: WebSocketEventHandler): void {
    const handlers = this.eventHandlers.get(event)
    if (handlers) {
      const index = handlers.indexOf(handler)
      if (index !== -1) {
        handlers.splice(index, 1)
      }
    }
  }

  private emit(event: string, message: WebSocketMessage): void {
    const handlers = this.eventHandlers.get(event)
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(message)
        } catch (error) {
          console.error('error in websocket event handler:', error)
        }
      })
    }
  }

  private handleMessage(message: WebSocketMessage): void {
    switch (message.type) {
      case 'sync': {
        const syncData = message.payload || message.data
        this.emit('sync', {
          type: 'sync',
          data: syncData
        })
        break
      }
      case 'participants':
        this.emit('participants', message)
        break
      case 'state':
        this.emit('state', message)
        break
      case 'request_state':
        console.log('received state request from backend:', message)
        this.emit('request_state', message)
        break
      case 'provide_state':
        console.log('received state provision from backend:', message)
        this.emit('provide_state', message)
        break
      case 'guest_request':
        this.emit('guest_request', message)
        break
      case 'guest_approved':
        this.emit('guest_approved', message)
        break
      case 'error':
        this.emit('error', message)
        break
      default:
        console.warn('unknown websocket message type:', message.type)
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('max reconnect attempts reached')
      return
    }

    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts)
    this.reconnectAttempts++

    setTimeout(() => {
      if (this.roomId) {
        const token = localStorage.getItem('token')
        this.connect(this.roomId, token || undefined)
      }
    }, delay)
  }

  // check if connected
  get isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }

  // get current room ID
  get currentRoomId(): string | null {
    return this.roomId
  }
}

// singleton instance
export const wsService = new WebSocketService()
