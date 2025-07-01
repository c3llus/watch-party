import { useState, useEffect, useCallback, useRef } from 'react'
import { roomService, type Room, type VideoAccess } from '../services/roomService'
import { wsService, type WebSocketMessage, type SyncAction, type BackendRoomState, type BackendSyncMessage } from '../services/webSocketService'
import type { ChatMessage } from '../types/chat'

export interface UseRoomOptions {
  roomId: string
  isGuest?: boolean
  guestToken?: string
  guestName?: string
  currentUserEmail?: string // for authenticated users
  onSyncEvent?: (syncEvent: BackendSyncMessage) => void // callback for sync events (for user logs)
}

export interface UseRoomReturn {
  // room data
  room: Room | null
  videoAccess: VideoAccess | null
  
  // connection state
  isConnected: boolean
  isLoading: boolean
  error: string | null
  
  // video state
  isPlaying: boolean
  currentTime: number
  lastActionBy: string | null
  suppressOutgoingSync: boolean // for debugging
  hasReceivedRoomState: boolean // for debugging
  hasAppliedInitialState: boolean // for debugging
  
  // chat state
  chatMessages: ChatMessage[]
  
  // websocket actions
  connect: () => Promise<void>
  disconnect: () => void
  refreshVideoAccess: () => Promise<void>
  sendSyncAction: (action: Omit<SyncAction, 'timestamp' | 'userId' | 'guestName'>) => void
  sendChatMessage: (message: string) => void
  
  // video sync callback - call this from VideoPlayer to sync to room state
  syncVideoToRoom: (videoElement: HTMLVideoElement) => void
  
  // set video element ref for real-time sync
  setVideoElement: (videoElement: HTMLVideoElement | null) => void
}

export function useRoom({ roomId, isGuest = false, guestToken, guestName, currentUserEmail, onSyncEvent }: UseRoomOptions): UseRoomReturn {
  const [room, setRoom] = useState<Room | null>(null)
  const [videoAccess, setVideoAccess] = useState<VideoAccess | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  // chat state
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([])
  
  // video state - sync state management
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [lastActionBy, setLastActionBy] = useState<string | null>(null)
  const [, setLastActionAt] = useState<number>(0)
  
  // track if we've done initial sync on join
  const [hasInitialSync, setHasInitialSync] = useState(false)
  
  // track if we should suppress outgoing sync actions (during initial sync)
  // prevent newly joined users from sending actions until they receive and apply initial room state
  const [suppressOutgoingSync, setSuppressOutgoingSync] = useState(true)
  
  // ref for immediate suppression during sync application (to prevent echo)
  const suppressSyncRef = useRef(false)
  
  // track if we've received initial room state from backend
  const [hasReceivedRoomState, setHasReceivedRoomState] = useState(false)
  
  // track if we've successfully applied the initial room state to prevent deadlock
  const [hasAppliedInitialState, setHasAppliedInitialState] = useState(false)
  
  // ref to video element for real-time sync
  const videoElementRef = useRef<HTMLVideoElement | null>(null)
  
  // current username - matches what backend generates
  const currentUsername = isGuest && guestName 
    ? `${guestName} (Guest)` 
    : (currentUserEmail?.split('@')[0] || 'User')
  
  // function to set video element ref
  const setVideoElement = useCallback((videoElement: HTMLVideoElement | null) => {
    videoElementRef.current = videoElement
  }, [])

  // chat functions
  const sendChatMessage = useCallback((message: string) => {
    if (!isConnected) {
      console.warn('not connected to room, cannot send chat message')
      return
    }
    
    // optimistically add chat message to local state immediately
    const optimisticChatMessage: ChatMessage = {
      id: Date.now().toString(),
      room_id: roomId,
      user_id: 'current-user', // placeholder for current user
      username: currentUsername, // use actual username for proper identification
      message: message.trim(),
      timestamp: new Date().toISOString()
    }
    setChatMessages(prev => [...prev, optimisticChatMessage])
    
    wsService.sendChatMessage(message)
  }, [isConnected, roomId, currentUsername])
  


  // load room data
  const loadRoom = useCallback(async () => {
    if (!roomId) return
    
    try {
      setIsLoading(true)
      setError(null)
      
      let roomData: Room
      
      if (isGuest && guestToken) {
        // for guests, use the dedicated guest endpoint
        const guestRoomData = await roomService.getRoomForGuest(roomId, guestToken)
        // convert guest room format to regular room format for compatibility
        roomData = {
          id: guestRoomData.id,
          name: guestRoomData.name,
          description: guestRoomData.description,
          movie_id: guestRoomData.movie?.id || '', // need movie ID for video access
          host_id: '', // guests don't need host info
          created_at: '',
          room_code: '',
          persistent_link: '',
          movie: guestRoomData.movie ? {
            id: guestRoomData.movie.id,
            title: guestRoomData.movie.title,
            description: guestRoomData.movie.description,
            status: 'ready',
            hls_url: undefined
          } : undefined
        } as Room
      } else {
        roomData = await roomService.getRoomForJoin(roomId)
      }
      
      setRoom(roomData)
      
      if (roomData.movie_id) {
        try {
          const access = await roomService.getVideoAccess(roomData.movie_id, guestToken)
          setVideoAccess(access)
        } catch (videoErr) {
          console.error('failed to get video access:', videoErr)
        }
      }
      
    } catch (err) {
      console.error('failed to load room:', err)
      setError(err instanceof Error ? err.message : 'Failed to load room')
    } finally {
      setIsLoading(false)
    }
  }, [roomId, isGuest, guestToken])

  // refresh video access (for when URLs expire)
  const refreshVideoAccess = useCallback(async () => {
    if (!room?.movie_id) return
    
    try {
      const access = await roomService.getVideoAccess(room.movie_id, guestToken)
      setVideoAccess(access)
    } catch (err) {
      console.error('failed to refresh video access:', err)
    }
  }, [room?.movie_id, guestToken])

  // websocket connection functions
  const connect = useCallback(async () => {
    try {
      const token = isGuest ? undefined : localStorage.getItem('token')
      await wsService.connect(roomId, token || undefined, guestToken, guestName)
      setIsConnected(true)
      
      // non-guests (hosts) can immediately send sync actions
      // guests must wait until they receive and apply initial room state
      if (!isGuest) {
        console.log('non-guest connected, enabling sync actions immediately')
        setHasAppliedInitialState(true)
        setSuppressOutgoingSync(false)
      }
    } catch (err) {
      console.error('failed to connect to websocket:', err)
      setError('failed to connect to room')
      setIsConnected(false)
    }
  }, [roomId, isGuest, guestToken, guestName, setHasAppliedInitialState, setSuppressOutgoingSync])

  // websocket sync action sender
  const sendSyncAction = useCallback((action: Omit<SyncAction, 'timestamp' | 'userId' | 'guestName'>) => {
    console.log('attempting to send sync action:', action, { 
      isConnected, 
      suppressOutgoingSync, 
      suppressSyncRef: suppressSyncRef.current,
      hasReceivedRoomState, 
      hasAppliedInitialState 
    })
    
    if (!isConnected) {
      console.log('not sending sync action - not connected')
      return
    }
    
    // check both state and ref for suppression
    if (suppressOutgoingSync || suppressSyncRef.current) {
      console.log('not sending sync action - suppressed during initial sync phase')
      return
    }
    
    if (!hasAppliedInitialState) {
      console.log('not sending sync action - have not applied initial room state yet')
      return
    }
    
    const fullAction: SyncAction = {
      ...action,
      timestamp: Date.now()
    }
    
    console.log('sending sync action to websocket:', fullAction)
    wsService.sendSyncAction(fullAction)
  }, [isConnected, suppressOutgoingSync, hasReceivedRoomState, hasAppliedInitialState])

  const disconnect = useCallback(() => {
    wsService.disconnect()
    setIsConnected(false)
  }, [])



  // websocket message handler - process sync messages and update video state
  const handleWebSocketMessage = useCallback((message: WebSocketMessage) => {
    console.log('received websocket message:', message.type, message)
    
    switch (message.type) {
      case 'sync':
        if ((message.payload || message.data) && typeof (message.payload || message.data) === 'object') {
          const syncData = (message.payload || message.data) as BackendSyncMessage
          
          // notify parent component about sync events for user logs
          if (onSyncEvent) {
            console.log('useRoom: sending sync event to UserLogs:', {
              action: syncData.action,
              username: syncData.username,
              currentUsername: currentUsername,
              isFromCurrentUser: syncData.username === currentUsername
            })
            onSyncEvent(syncData)
          }
          
          // handle chat messages from sync events
          if (syncData.action === 'chat' && syncData.data?.chat_message) {
            const chatMessage: ChatMessage = {
              id: syncData.user_id || Date.now().toString(),
              room_id: roomId, // use current room ID
              user_id: syncData.user_id || '',
              username: syncData.username || 'Unknown User',
              message: syncData.data.chat_message,
              timestamp: syncData.timestamp || new Date().toISOString()
            }
            console.log('received chat sync message:', chatMessage)
            setChatMessages(prev => [...prev, chatMessage])
          }
          
          console.log('processing sync message:', syncData, { 
            suppressOutgoingSync, 
            hasInitialSync, 
            hasVideoElement: !!videoElementRef.current,
            videoReadyState: videoElementRef.current?.readyState,
            videoSrc: videoElementRef.current?.src
          })
          
          console.log('updating room state from sync message')
          setIsPlaying(syncData.action === 'play')
          setCurrentTime(syncData.current_time || 0)
          setLastActionBy(syncData.username || syncData.user_id || 'unknown')
          setLastActionAt(new Date(syncData.timestamp).getTime() || Date.now())
          
          const videoElement = videoElementRef.current
          console.log('video element for sync:', { 
            hasVideoElement: !!videoElement,
            videoSrc: videoElement?.src,
            readyState: videoElement?.readyState
          })
          
          if (videoElement) {
            console.log('video element available, preparing sync action')
            const applySyncAction = () => {
              console.log('EXECUTING applySyncAction for:', syncData.action)
              
              // temporarily suppress outgoing sync to prevent echo
              const originalSuppress = suppressOutgoingSync
              const originalSuppressRef = suppressSyncRef.current
              setSuppressOutgoingSync(true)
              suppressSyncRef.current = true
              
              try {
                console.log('applying sync action to video:', syncData.action, { 
                  currentTime: videoElement.currentTime, 
                  targetTime: syncData.current_time,
                  isPaused: videoElement.paused,
                  readyState: videoElement.readyState,
                  videoSrc: videoElement.src,
                  videoDuration: videoElement.duration
                })
                
                // check if video is ready for playback - be more lenient for basic operations
                const isVideoReady = videoElement.readyState >= 1 && videoElement.src
                const needsStrictReady = syncData.action === 'seek' // seeking needs more ready state
                
                if (!isVideoReady || (needsStrictReady && (videoElement.readyState < 2 || videoElement.duration === 0))) {
                  console.log('video not ready for sync:', {
                    action: syncData.action,
                    readyState: videoElement.readyState,
                    hasSrc: !!videoElement.src,
                    duration: videoElement.duration,
                    networkState: videoElement.networkState,
                    needsStrictReady
                  })
                  
                  // for play/pause, try anyway if we have some readiness
                  if (isVideoReady && !needsStrictReady) {
                    console.log('proceeding with play/pause despite limited readiness')
                  } else {
                    // schedule retry when video becomes ready
                    const retryWhenReady = () => {
                      const newReadiness = videoElement.readyState >= 1 && videoElement.src
                      const newStrictReadiness = videoElement.readyState >= 2 && videoElement.duration > 0
                      
                      if (newReadiness && (!needsStrictReady || newStrictReadiness)) {
                        console.log('video became ready, retrying sync action')
                        applySyncAction()
                      } else if (videoElementRef.current === videoElement) {
                        // only retry if the video element is still the same
                        setTimeout(retryWhenReady, 200)
                      }
                    }
                    setTimeout(retryWhenReady, 200)
                    return
                  }
                }
                
                switch (syncData.action) {
                  case 'play':
                    console.log('applying play action to video element')
                    if (videoElement.paused) {
                      console.log('video is paused, calling play()')
                      const playPromise = videoElement.play()
                      if (playPromise !== undefined) {
                        playPromise.then(() => {
                          console.log('video play successful')
                        }).catch(err => {
                          console.error('failed to play video during sync:', err)
                          // for HLS stream errors, try to recover
                          if (err.message && err.message.includes('MEDIA_ELEMENT_ERROR')) {
                            console.log('attempting to recover from media error')
                            setTimeout(() => {
                              if (videoElement.paused && videoElement.readyState >= 1) {
                                console.log('retrying play after media error recovery')
                                videoElement.play().catch(e => console.error('retry play failed:', e))
                              }
                            }, 1000)
                          }
                        })
                      }
                    } else {
                      console.log('video is already playing, no action needed')
                    }
                    break
                  case 'pause':
                    console.log('applying pause action to video element')
                    if (!videoElement.paused) {
                      console.log('video is playing, calling pause()')
                      try {
                        videoElement.pause()
                        console.log('video paused successfully')
                      } catch (err) {
                        console.error('failed to pause video:', err)
                      }
                    } else {
                      console.log('video is already paused, no action needed')
                    }
                    break
                  case 'seek':
                    if (syncData.current_time !== undefined) {
                      const timeDiff = Math.abs(videoElement.currentTime - syncData.current_time)
                      console.log('seek sync check:', {
                        currentTime: videoElement.currentTime,
                        targetTime: syncData.current_time,
                        timeDiff,
                        threshold: 0.5,
                        videoDuration: videoElement.duration
                      })
                      
                      if (timeDiff > 0.5) {
                        const wasPlaying = !videoElement.paused
                        console.log('seeking video from', videoElement.currentTime, 'to', syncData.current_time)
                        
                        try {
                          // ensure seek time is within valid range
                          const seekTime = Math.max(0, Math.min(syncData.current_time, videoElement.duration || Infinity))
                          
                          console.log('setting currentTime to:', seekTime)
                          videoElement.currentTime = seekTime
                          
                          // wait for seek to complete before resuming play
                          const handleSeeked = () => {
                            console.log('seek completed, current time now:', videoElement.currentTime)
                            videoElement.removeEventListener('seeked', handleSeeked)
                            
                            if (wasPlaying && videoElement.paused) {
                              console.log('resuming play after seek')
                              const playPromise = videoElement.play()
                              if (playPromise) {
                                playPromise.catch(err => console.error('failed to resume play after seek:', err))
                              }
                            }
                          }
                          
                          videoElement.addEventListener('seeked', handleSeeked)
                          
                          // fallback timeout in case seeked event doesn't fire
                          setTimeout(() => {
                            videoElement.removeEventListener('seeked', handleSeeked)
                            if (wasPlaying && videoElement.paused && videoElement.readyState >= 2) {
                              console.log('seek timeout, attempting to resume play')
                              videoElement.play().catch(err => console.error('timeout resume play failed:', err))
                            }
                          }, 1000)
                          
                        } catch (err) {
                          console.error('error during seek operation:', err)
                        }
                      } else {
                        console.log('time difference too small, skipping seek')
                      }
                    }
                    break
                }
                
                // prevent applying sync actions from others until we've applied our initial state
                // this prevents deadlock between newly joined user and existing users
                if (!hasAppliedInitialState) {
                  console.log('ignoring sync action - have not applied initial room state yet')
                  return
                }
                
                // if this is a guest and we haven't done initial sync yet, mark it as done now
                // since we're successfully processing sync events
                if (isGuest && !hasInitialSync) {
                  console.log('marking guest as initially synced via real-time sync event')
                  setHasInitialSync(true)
                  // guests can send sync actions after receiving their first sync event
                  setHasAppliedInitialState(true)
                  setSuppressOutgoingSync(false)
                }
                
              } catch (error) {
                console.error('error applying real-time sync:', error)
              } finally {
                // restore original suppress state after a short delay to allow video events to settle
                setTimeout(() => {
                  setSuppressOutgoingSync(originalSuppress)
                  suppressSyncRef.current = originalSuppressRef
                }, 100)
              }
            }
            
            console.log('applying sync action directly')
            // ensure video element still exists before applying sync
            if (videoElementRef.current) {
              applySyncAction()
            } else {
              console.warn('video element no longer available, cannot apply sync action')
            }
          } else {
            console.warn('no video element available for sync action:', syncData.action)
          }
        }
        break
        
      case 'state':
        console.log('received room state message:', message.payload)
        if (message.payload && typeof message.payload === 'object') {
          const backendState = message.payload as BackendRoomState
          
          console.log('setting room state:', {
            isPlaying: backendState.is_playing,
            currentTime: backendState.current_time,
            lastUpdated: backendState.last_updated,
            updatedBy: backendState.updated_by
          })
          setIsPlaying(backendState.is_playing)
          setCurrentTime(backendState.current_time)
          setLastActionBy(backendState.updated_by || null)
          const lastActionTimestamp = backendState.last_updated 
            ? new Date(backendState.last_updated).getTime() 
            : Date.now()
          setLastActionAt(lastActionTimestamp)
          
          setHasReceivedRoomState(true)
          console.log('room state received and set, hasReceivedRoomState will be true')
          
          // important: only enable sync actions after initial state is applied to video
          // this prevents deadlock between newly joined user and existing users
          console.log('initial room state received, will enable sync actions after applying to video')
        }
        break
        
      case 'request_state':
        // existing user receives request for current video state from backend (for newly joined user)
        if (message.payload && typeof message.payload === 'object') {
          const requestData = message.payload as { requester_id: string }
          console.log('received state request from backend for requester:', requestData.requester_id)
          
          const videoElement = videoElementRef.current
          if (videoElement) {
            const currentState = {
              isPlaying: !videoElement.paused,
              currentTime: videoElement.currentTime
            }
            console.log('current video state:', {
              isPlaying: currentState.isPlaying,
              currentTime: currentState.currentTime,
              videoReadyState: videoElement.readyState,
              videoDuration: videoElement.duration
            })
            console.log('providing state to backend for requester:', requestData.requester_id)
            wsService.provideCurrentState(requestData.requester_id, currentState)
          } else {
            console.warn('no video element available to provide state')
            console.log('videoElementRef.current:', videoElementRef.current)
          }
        }
        break
        
      case 'chat':
        // handle incoming chat messages
        if (message.payload && typeof message.payload === 'object') {
          const chatMessage = message.payload as ChatMessage
          console.log('received chat message:', chatMessage)
          setChatMessages(prev => [...prev, chatMessage])
        }
        break
      case 'guest_request':
        break
      case 'connected':
        break
      case 'disconnected':
        break
      case 'error':
        break
    }
  }, [onSyncEvent, hasInitialSync, suppressOutgoingSync, isGuest, setSuppressOutgoingSync, hasAppliedInitialState, setHasAppliedInitialState, roomId, currentUsername])

  // helper function to apply room state to video element
  const applyRoomStateToVideo = useCallback((videoElement: HTMLVideoElement) => {
    if (!videoElement) {
      console.log('no video element provided to apply room state')
      return
    }
    
    console.log('applying room state to video:', { isPlaying, currentTime, isGuest })
    
    // sync current time if there's a significant difference
    const timeDiff = Math.abs(videoElement.currentTime - currentTime)
    if (timeDiff > 1) {
      console.log('syncing video time from', videoElement.currentTime, 'to', currentTime)
      videoElement.currentTime = currentTime
    }
    
    // sync play/pause state
    if (isPlaying) {
      if (videoElement.paused) {
        console.log('starting video playback to match room state')
        videoElement.play().catch(err => console.error('failed to play video during room sync:', err))
      }
    } else {
      if (!videoElement.paused) {
        console.log('pausing video to match room state')
        videoElement.pause()
      }
    }
    
    // mark as synced and enable sync actions
    setHasInitialSync(true)
    setHasAppliedInitialState(true)
    setSuppressOutgoingSync(false)
    
    console.log('room state applied to video successfully, sync actions now enabled')
  }, [currentTime, isPlaying, setSuppressOutgoingSync, isGuest, setHasAppliedInitialState])

  // sync video to current room state
  const syncVideoToRoom = useCallback((videoElement: HTMLVideoElement) => {
    if (!videoElement || !isGuest || hasInitialSync) {
      console.log('skipping guest sync:', { hasVideo: !!videoElement, isGuest, hasInitialSync })
      return
    }
    
    console.log('guest sync starting, current state:', { hasReceivedRoomState, isPlaying, currentTime })
    
    // if we already have room state, apply it immediately
    if (hasReceivedRoomState) {
      console.log('room state already available, applying immediately')
      applyRoomStateToVideo(videoElement)
      return
    }
    
    // if we don't have room state yet, request it
    console.log('no room state available, requesting from backend')
    wsService.requestRoomState()
    
    // set up a timeout to ensure we don't wait forever
    setTimeout(() => {
      if (!hasInitialSync && !hasAppliedInitialState) {
        if (hasReceivedRoomState) {
          console.log('applying delayed room state after timeout check')
          applyRoomStateToVideo(videoElement)
        } else {
          console.warn('guest sync timeout, proceeding without initial state')
          setHasInitialSync(true)
          setHasAppliedInitialState(true)
          setSuppressOutgoingSync(false)
        }
      }
    }, 5000) // 5 second timeout
    
  }, [hasInitialSync, hasReceivedRoomState, isGuest, applyRoomStateToVideo, setSuppressOutgoingSync, isPlaying, currentTime, hasAppliedInitialState, setHasAppliedInitialState])

  // sync video to room state when room state is received or video element becomes available
  useEffect(() => {
    if (hasReceivedRoomState && videoElementRef.current && !hasInitialSync) {
      if (isGuest) {
        // for guests, apply room state through the robust sync mechanism
        syncVideoToRoom(videoElementRef.current)
      } else {
        // for hosts, apply state directly
        applyRoomStateToVideo(videoElementRef.current)
      }
    }
  }, [hasReceivedRoomState, hasInitialSync, isGuest, syncVideoToRoom, applyRoomStateToVideo])

  // websocket connection and message handling
  useEffect(() => {
    if (!roomId) return
    
    // create stable handler references for cleanup
    const connectHandler = () => {
      setIsConnected(true)
    }
    
    const disconnectHandler = () => {
      setIsConnected(false)
    }
    
    const errorHandler = (message: WebSocketMessage) => {
      console.error('websocket error:', message.payload || message.data)
      setError(`websocket error: ${message.payload || message.data}`)
    }
    
    // set up message handlers
    wsService.on('sync', handleWebSocketMessage)
    wsService.on('state', handleWebSocketMessage)
    wsService.on('guest_request', handleWebSocketMessage)
    wsService.on('request_state', handleWebSocketMessage)
    wsService.on('chat', handleWebSocketMessage)
    wsService.on('connected', connectHandler)
    wsService.on('disconnected', disconnectHandler)
    wsService.on('error', errorHandler)
    
    return () => {
      // cleanup event handlers
      wsService.off('sync', handleWebSocketMessage)
      wsService.off('state', handleWebSocketMessage)
      wsService.off('guest_request', handleWebSocketMessage)
      wsService.off('request_state', handleWebSocketMessage)
      wsService.off('chat', handleWebSocketMessage)
      wsService.off('connected', connectHandler)
      wsService.off('disconnected', disconnectHandler)
      wsService.off('error', errorHandler)
    }
  }, [roomId, handleWebSocketMessage])

  // auto-connect when room is loaded
  useEffect(() => {
    if (room && !isConnected && !isLoading) {
      connect()
    }
  }, [room, isConnected, isLoading, connect])

  // cleanup on unmount
  useEffect(() => {
    return () => {
      if (isConnected) {
        disconnect()
      }
    }
  }, [disconnect, isConnected])

  // load room on mount
  useEffect(() => {
    loadRoom()
  }, [loadRoom])



  return {
    room,
    videoAccess,
    isConnected,
    isLoading,
    error,
    isPlaying,
    currentTime,
    lastActionBy,
    suppressOutgoingSync, // for debugging
    hasReceivedRoomState, // for debugging
    hasAppliedInitialState, // for debugging
    chatMessages,
    // websocket functions
    connect,
    disconnect,
    refreshVideoAccess,
    sendSyncAction,
    sendChatMessage,
    syncVideoToRoom,
    setVideoElement
  }
}
