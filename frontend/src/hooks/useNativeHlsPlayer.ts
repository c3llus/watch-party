import { useEffect, useRef, useState, useCallback } from 'react'

interface UseNativeHlsPlayerOptions {
  movieId: string
  guestToken?: string
  onError?: (error: string) => void
}

interface UseNativeHlsPlayerReturn {
  isLoading: boolean
  error: string | null
  seekTo: (time: number) => Promise<void>
  retry: () => Promise<void>
  isInitialized: boolean
}

export function useNativeHlsPlayer(
  videoRef: React.RefObject<HTMLVideoElement | null>,
  masterUrlResolver: () => Promise<string>,
  options: UseNativeHlsPlayerOptions
): UseNativeHlsPlayerReturn {
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isInitialized, setIsInitialized] = useState(false)
  const mountedRef = useRef(true)
  const currentUrlRef = useRef<string | null>(null)

  const { onError } = options

  const initializeNativeHls = useCallback(async () => {
    const videoEl = videoRef.current
    if (!videoEl || !mountedRef.current) return false

    try {
      setIsLoading(true)
      setError(null)

      const masterUrl = await masterUrlResolver()
      
      const canPlayHls = videoEl.canPlayType('application/vnd.apple.mpegurl') !== '' ||
                        videoEl.canPlayType('application/x-mpegurl') !== ''

      if (!canPlayHls) {
        throw new Error('native HLS not supported by browser')
      }

      currentUrlRef.current = masterUrl
      videoEl.src = masterUrl
      
      await new Promise<void>((resolve, reject) => {
        const timeoutId = setTimeout(() => {
          reject(new Error('video load timeout'))
        }, 15000)

        const handleLoadedMetadata = () => {
          clearTimeout(timeoutId)
          videoEl.removeEventListener('loadedmetadata', handleLoadedMetadata)
          videoEl.removeEventListener('error', handleError)
          resolve()
        }

        const handleError = () => {
          clearTimeout(timeoutId)
          videoEl.removeEventListener('loadedmetadata', handleLoadedMetadata)
          videoEl.removeEventListener('error', handleError)
          reject(new Error('video load failed'))
        }

        videoEl.addEventListener('loadedmetadata', handleLoadedMetadata)
        videoEl.addEventListener('error', handleError)
        
        videoEl.load()
      })

      setIsInitialized(true)
      setIsLoading(false)
      return true

    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'native HLS failed'
      setError(errorMsg)
      setIsLoading(false)
      onError?.(errorMsg)
      return false
    }
  }, [videoRef, masterUrlResolver, onError])

  const seekTo = useCallback(async (time: number) => {
    const videoEl = videoRef.current
    if (!videoEl || !isInitialized) {
      throw new Error('video not initialized')
    }

    return new Promise<void>((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        reject(new Error('seek timeout'))
      }, 5000)

      const handleSeeked = () => {
        clearTimeout(timeoutId)
        videoEl.removeEventListener('seeked', handleSeeked)
        resolve()
      }

      videoEl.addEventListener('seeked', handleSeeked)
      videoEl.currentTime = time
    })
  }, [videoRef, isInitialized])

  const retry = useCallback(async () => {
    setIsInitialized(false)
    setError(null)
    setIsLoading(true)
    
    const videoEl = videoRef.current
    if (videoEl && currentUrlRef.current) {
      videoEl.src = ''
      videoEl.load()
      currentUrlRef.current = null
    }
    
    await initializeNativeHls()
  }, [initializeNativeHls, videoRef])

  useEffect(() => {
    const videoEl = videoRef.current
    if (!videoEl || isInitialized) return
    
    mountedRef.current = true
    initializeNativeHls()

    return () => {
      mountedRef.current = false
      if (videoEl && currentUrlRef.current) {
        // avoid empty src error by only clearing if we have a valid URL
        videoEl.removeAttribute('src')
        videoEl.load()
        currentUrlRef.current = null
      }
    }
  }, [initializeNativeHls, videoRef, isInitialized])

  return {
    isLoading,
    error,
    seekTo,
    retry,
    isInitialized
  }
}
