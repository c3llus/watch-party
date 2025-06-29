import { useRef, useCallback, useMemo, useState } from 'react'
import { useHlsPlayer } from '../../hooks/useHlsPlayer'
import { useNativeHlsPlayer } from '../../hooks/useNativeHlsPlayer'
import { getSignedUrl } from '../../utils/hlsParser'
import 'video.js/dist/video-js.css'

interface VideoPlayerProps {
  movieId: string
  guestToken?: string
  onError?: (error: string) => void
  onPlay?: () => void
  onPause?: () => void
  onSeeked?: (time: number) => void
  onSyncToRoom?: (videoElement: HTMLVideoElement) => void
  onVideoReady?: (videoElement: HTMLVideoElement) => void
  style?: React.CSSProperties
  className?: string
  waitForSync?: boolean
}

export function VideoPlayer({
  movieId,
  guestToken,
  onError,
  onPlay,
  onPause,
  onSeeked,
  onSyncToRoom,
  onVideoReady,
  style,
  className,
  waitForSync = false
}: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [useNative, setUseNative] = useState(true)

  const masterUrlResolver = useCallback(async (): Promise<string> => {
    const masterPlaylistPath = 'master.m3u8'
    const masterSignedUrl = await getSignedUrl(movieId, masterPlaylistPath, guestToken)
    return masterSignedUrl
  }, [movieId, guestToken])

  const getSegmentUrl = useCallback(async (uri: string): Promise<string> => {
    const segmentPath = uri.split('/').slice(-2).join('/')
    const signedUrl = await getSignedUrl(movieId, segmentPath, guestToken)
    return signedUrl
  }, [movieId, guestToken])

  const nativeHls = useNativeHlsPlayer(
    videoRef,
    masterUrlResolver,
    {
      movieId,
      guestToken,
      onError: () => {
        setUseNative(false)
      }
    }
  )

  // fallback to custom HLS implementation
  const customHls = useHlsPlayer(
    videoRef,
    masterUrlResolver,
    getSegmentUrl,
    {
      movieId,
      guestToken,
      onError
    }
  )

  // choose which implementation to use
  const { isLoading } = useNative ? nativeHls : customHls

  // event handlers
  const handlePlay = useCallback(() => {
    if (waitForSync) {
      // for guests waiting for sync, allow the play to proceed but defer the sync action
      // the useRoom hook will handle the robust sync logic
      setTimeout(() => {
        const video = videoRef.current
        if (video && !video.paused) {
          // only pause if we're still in waitForSync mode
          // this gives time for the robust guest sync to take control
          if (waitForSync) {
            video.pause()
          }
        }
      }, 100) // small delay to allow sync logic to take control
      return
    }
    onPlay?.()
  }, [onPlay, waitForSync])

  const handlePause = useCallback(() => {
    onPause?.()
  }, [onPause])

  const handleSeeked = useCallback(() => {
    const currentTime = videoRef.current?.currentTime ?? 0
    onSeeked?.(currentTime)
  }, [onSeeked])

  const handleError = useCallback(() => {
    const errorMsg = 'video playback error occurred'
    onError?.(errorMsg)
  }, [onError])

  // default styles following the style guide
  const defaultStyle: React.CSSProperties = useMemo(() => ({
    width: '100%',
    height: 'auto',
    display: 'block',
    aspectRatio: '16/9',
    backgroundColor: '#000'
  }), [])

  const videoStyle = useMemo(() => ({
    ...defaultStyle,
    ...style
  }), [defaultStyle, style])

  // render video element with overlay states instead of conditional rendering
  return (
    <div style={{ position: 'relative', ...videoStyle }}>
      {/* always render video element so ref gets attached */}
      <video
        ref={videoRef}
        style={{ 
          width: '100%', 
          height: '100%',
          display: 'block',
          aspectRatio: '16/9',
          backgroundColor: '#000'
        }}
        className={className}
        controls={true} // always enable controls - volume should work locally
        muted={false} // explicitly enable volume controls
        playsInline
        preload="metadata"
        onPlay={handlePlay}
        onPause={handlePause}
        onSeeked={handleSeeked}
        onError={handleError}
        onLoadedData={() => {
          const video = videoRef.current
          if (video) {
            onVideoReady?.(video)
            onSyncToRoom?.(video)
          }
        }}
        onCanPlay={() => {
          // also trigger when video can start playing
          const video = videoRef.current
          if (video && video.readyState >= 3) {
            console.log('video can play, readyState:', video.readyState)
            onVideoReady?.(video)
          }
        }}
      />
      
      {/* loading overlay */}
      {isLoading && (
        <div 
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: 'rgba(0, 0, 0, 0.8)',
            color: 'white',
            fontSize: '1rem',
            zIndex: 1
          }}
        >
          <div>loading video...</div>
          <div style={{ fontSize: '0.75rem', marginTop: '0.5rem', opacity: 0.7 }}>
            {useNative ? 'using native HLS' : 'using custom HLS'}
          </div>
        </div>
      )}
    </div>
  )
}
