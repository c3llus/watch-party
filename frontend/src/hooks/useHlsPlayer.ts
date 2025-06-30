import { useEffect, useRef, useState, useCallback } from 'react'
import { createHlsPlayer, type HlsPlayer } from '../services/hlsService'

interface UseHlsPlayerOptions {
  movieId: string
  guestToken?: string
  onError?: (error: string) => void
}

// global instance manager to prevent multiple concurrent players in React StrictMode
const instanceManager = {
  currentInstance: null as { movieId: string; player: HlsPlayer; videoElement: HTMLVideoElement } | null,
  allInstances: new Map<string, HlsPlayer>(),
  
  generateKey(movieId: string, videoElement: HTMLVideoElement): string {
    return `${movieId}_${videoElement.dataset.playerId || Math.random().toString(36)}`
  },
  
  canInitialize(movieId: string, videoElement: HTMLVideoElement): boolean {
    // ensure video element has a unique identifier
    if (!videoElement.dataset.playerId) {
      videoElement.dataset.playerId = Math.random().toString(36).substring(2)
    }
    
    // if no current instance, allow initialization
    if (!this.currentInstance) return true;
    
    // if same movie and same video element, allow (reuse)
    if (this.currentInstance.movieId === movieId && this.currentInstance.videoElement === videoElement) {
      return true;
    }
    
    // if different movie or different video element, clean up previous and allow
    if (this.currentInstance.movieId !== movieId || this.currentInstance.videoElement !== videoElement) {
      this.cleanupInstance(this.currentInstance);
      this.currentInstance = null;
      return true;
    }
    
    return false;
  },
  
  setInstance(movieId: string, player: HlsPlayer, videoElement: HTMLVideoElement) {
    const key = this.generateKey(movieId, videoElement)
    
    // cleanup any existing instance for this key
    const existingPlayer = this.allInstances.get(key)
    if (existingPlayer && existingPlayer !== player) {
      existingPlayer.cleanup()
    }
    
    // set as current and track in map
    this.currentInstance = { movieId, player, videoElement };
    this.allInstances.set(key, player)
  },
  
  cleanup(movieId: string, videoElement: HTMLVideoElement) {
    const key = this.generateKey(movieId, videoElement)
    
    // cleanup from map
    const player = this.allInstances.get(key)
    if (player) {
      player.cleanup()
      this.allInstances.delete(key)
    }
    
    // cleanup current instance if it matches
    if (this.currentInstance && 
        this.currentInstance.movieId === movieId && 
        this.currentInstance.videoElement === videoElement) {
      this.cleanupInstance(this.currentInstance);
      this.currentInstance = null;
    }
  },
  
  cleanupInstance(instance: { movieId: string; player: HlsPlayer; videoElement: HTMLVideoElement }) {
    try {
      instance.player.cleanup()
      
      // ensure video element is completely cleaned
      const videoEl = instance.videoElement
      if (videoEl.src && videoEl.src.startsWith('blob:')) {
        videoEl.removeAttribute('src')
        videoEl.load()
      }
    } catch (error) {
      console.error('error during instance cleanup:', error)
    }
  },
  
  cleanupAll() {
    
    // cleanup all tracked instances
    for (const [key, player] of this.allInstances.entries()) {
      try {
        player.cleanup()
      } catch (error) {
        console.error('error during force cleanup of instance:', key, error)
      }
    }
    
    this.allInstances.clear()
    
    // cleanup current instance
    if (this.currentInstance) {
      this.cleanupInstance(this.currentInstance)
      this.currentInstance = null
    }
  },
  
  getPlayer(movieId: string, videoElement: HTMLVideoElement): HlsPlayer | null {
    if (this.currentInstance && 
        this.currentInstance.movieId === movieId && 
        this.currentInstance.videoElement === videoElement) {
      return this.currentInstance.player;
    }
    return null;
  }
};

// global cleanup handlers to prevent background players
if (typeof window !== 'undefined') {
  // cleanup all instances when page becomes hidden (user switches tabs/minimizes)
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      // pause current video element but don't cleanup - user might come back
      if (instanceManager.currentInstance) {
        try {
          const videoEl = instanceManager.currentInstance.videoElement
          if (videoEl && !videoEl.paused) {
            videoEl.pause()
          }
        } catch (error) {
          console.warn('error pausing video on visibility change:', error)
        }
      }
    }
  })
  
  // cleanup all instances when user navigates away
  window.addEventListener('beforeunload', () => {
    instanceManager.cleanupAll()
  })
  
  // cleanup all instances when user navigates away (modern browsers)
  window.addEventListener('pagehide', () => {
    instanceManager.cleanupAll()
  })
}

export function useHlsPlayer(
  videoRef: React.RefObject<HTMLVideoElement | null>,
  masterUrlResolver: () => Promise<string>,
  getSegmentUrl: (uri: string) => Promise<string>,
  options: UseHlsPlayerOptions
) {
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const playerRef = useRef<HlsPlayer | null>(null)
  const isInitializedRef = useRef(false)
  const mountedRef = useRef(true)
  const lastKnownTimeRef = useRef(0) // track last known playback time for retry

  const { movieId, onError } = options

  // define retry function with useCallback to create stable reference
  const retry = useCallback(async () => {
    
    const videoEl = videoRef.current
    
    // store current time if video element is available
    if (videoEl && !isNaN(videoEl.currentTime) && videoEl.currentTime > 0) {
      lastKnownTimeRef.current = videoEl.currentTime
    }
    
    // if we have a player instance, use restart method
    if (playerRef.current) {
      try {
        setError(null)
        setIsLoading(true)
        
        await playerRef.current.restart(lastKnownTimeRef.current)
        lastKnownTimeRef.current = 0 // reset after use
        
        setIsLoading(false)
        return
      } catch (error) {
        console.error('restart failed, falling back to complete reinitialization:', error)
      }
    }
    
    // fallback: complete reinitialization
    if (playerRef.current) {
      playerRef.current.cleanup()
      playerRef.current = null
    }
    
    // force cleanup of instance manager
    if (videoEl) {
      instanceManager.cleanup(movieId, videoEl)
    }
    
    // reset all state
    isInitializedRef.current = false
    setError(null)
    setIsLoading(true)
  }, [movieId, videoRef]) // dependencies for retry function

  // initialize player - use polling to detect when video element becomes available
  useEffect(() => {
    const currentVideoEl = videoRef.current;  // Capture at effect start
    mountedRef.current = true;
    
    if (!movieId || isInitializedRef.current) {
      return
    }

    let pollInterval: ReturnType<typeof setInterval> | undefined

    const initializePlayer = async () => {
      const videoEl = videoRef.current
      
      if (!videoEl || !mountedRef.current) {
        return false
      }

      // Check if we can initialize (prevent multiple instances)
      if (!instanceManager.canInitialize(movieId, videoEl)) {
        console.log('skipping initialization - another instance is active or initializing for movie:', movieId);
        console.log('instance manager state:', {
          currentMovieId: instanceManager.currentInstance?.movieId,
          sameVideoElement: instanceManager.currentInstance?.videoElement === videoEl
        });
        return true; // stop polling
      }

      // Check if this movie already has a player
      const existingPlayer = instanceManager.getPlayer(movieId, videoEl);
      if (existingPlayer && existingPlayer.isInitialized()) {
        console.log('reusing existing player for movieId:', movieId);
        playerRef.current = existingPlayer;
        isInitializedRef.current = true;
        setIsLoading(false);
        return true;
      }

      // Mark as initializing to prevent double initialization
      console.log('starting initialization for movieId:', movieId);

      try {
        setIsLoading(true)
        setError(null)
        
        console.log('initializing HLS player for movie:', movieId)
        
        // create player instance
        const player = createHlsPlayer(
          videoEl,
          masterUrlResolver,
          getSegmentUrl,
          {
            onFatalError: (error: string) => {
              console.log('fatal error received from HLS service, triggering automatic retry:', error)
              
              // immediately capture current time before it gets reset
              const currentTime = videoEl.currentTime
              if (!isNaN(currentTime) && currentTime > 0) {
                lastKnownTimeRef.current = currentTime
                console.log('captured current time for automatic retry:', currentTime)
              }
              
              // trigger automatic retry after a short delay
              setTimeout(() => {
                if (mountedRef.current) {
                  console.log('executing automatic retry for fatal error')
                  retry().catch((retryError: unknown) => {
                    console.error('automatic retry failed:', retryError)
                    if (mountedRef.current) {
                      const errorMessage = `automatic retry failed: ${retryError instanceof Error ? retryError.message : 'unknown error'}`
                      setError(errorMessage)
                      onError?.(errorMessage)
                    }
                  })
                }
              }, 1000) // 1 second delay to let error state settle
            }
          }
        )
        
        playerRef.current = player
        instanceManager.setInstance(movieId, player, videoEl);
        
        // start playback
        await player.start()
        
        // if this is a retry and we have a stored time, seek to it
        if (lastKnownTimeRef.current > 0) {
          console.log('seeking to last known time after retry:', lastKnownTimeRef.current)
          await player.seekTo(lastKnownTimeRef.current)
          lastKnownTimeRef.current = 0 // reset after use
        }
        
        if (mountedRef.current) {
          isInitializedRef.current = true
          setIsLoading(false)
          console.log('HLS player initialized successfully')
          return true
        }
        
      } catch (err) {
        console.error('failed to initialize HLS player:', err)
        if (mountedRef.current) {
          const errorMessage = err instanceof Error ? err.message : 'failed to initialize video player'
          setError(errorMessage)
          setIsLoading(false)
          onError?.(errorMessage)
        }
        return true // stop polling on error
      }
      
      return false
    }

    // start polling for video element
    console.log('starting video element polling for movieId:', movieId)
    
    // eslint-disable-next-line prefer-const
    pollInterval = setInterval(async () => {
      const success = await initializePlayer()
      if (success) {
        clearInterval(pollInterval)
      }
    }, 100) // check every 100ms
    
    // also try immediately
    initializePlayer().then(success => {
      if (success) {
        clearInterval(pollInterval)
      }
    })

    return () => {
      mountedRef.current = false;
      if (pollInterval) {
        clearInterval(pollInterval)
      }
      
      // Only cleanup if this was our instance
      if (isInitializedRef.current && currentVideoEl) {
        console.log('cleaning up HLS player for movieId:', movieId)
        instanceManager.cleanup(movieId, currentVideoEl);
        playerRef.current = null;
        isInitializedRef.current = false;
      }
    }
  }, [videoRef, masterUrlResolver, getSegmentUrl, movieId, onError, retry])

  // cleanup on unmount
  useEffect(() => {
    const currentVideoEl = videoRef.current;
    return () => {
      if (playerRef.current && isInitializedRef.current && currentVideoEl) {
        instanceManager.cleanup(movieId, currentVideoEl);
        playerRef.current = null;
        isInitializedRef.current = false;
      }
    }
  }, [movieId, videoRef])

  const seekTo = async (time: number) => {
    if (playerRef.current && playerRef.current.isInitialized()) {
      try {
        await playerRef.current.seekTo(time)
      } catch (err) {
        console.error('seek failed:', err)
        const errorMessage = err instanceof Error ? err.message : 'seek operation failed'
        setError(errorMessage)
        onError?.(errorMessage)
      }
    } else {
      console.warn('cannot seek: player not initialized')
    }
  }

  return {
    isLoading,
    error,
    seekTo,
    retry,
    isInitialized: playerRef.current?.isInitialized() ?? false
  }
}
