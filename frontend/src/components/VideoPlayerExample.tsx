import { useEffect, useState } from 'react'
import { videoStreamingService, configService } from '../services'

interface VideoPlayerProps {
  movieId: string
  guestToken?: string
}

export function VideoPlayer({ movieId, guestToken }: VideoPlayerProps) {
  const [videoSrc, setVideoSrc] = useState<string>('')
  const [deploymentMode, setDeploymentMode] = useState<string>('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function setupVideo() {
      try {
        const config = await configService.getDeploymentConfig()
        setDeploymentMode(config.mode)

        const hlsUrl = await videoStreamingService.getVideoSource({
          movieId,
          guestToken
        })
        
        setVideoSrc(hlsUrl)
        
      } catch (error) {
        console.error('failed to setup video:', error)
      } finally {
        setLoading(false)
      }
    }

    if (movieId) {
      setupVideo()
    }
  }, [movieId, guestToken])

  if (loading) {
    return <div>loading video...</div>
  }

  return (
    <div>
      <div className="video-info">
        <small>mode: {deploymentMode}</small>
      </div>
      <video
        controls
        width="100%"
        height="400"
        src={videoSrc}
        onError={(e) => console.error('video error:', e)}
      >
        your browser does not support the video tag
      </video>
    </div>
  )
}
