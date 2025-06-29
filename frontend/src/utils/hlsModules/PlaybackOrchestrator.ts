/**
 * PlaybackOrchestrator - Coordinates all HLS playback components
 */

import { BufferManager } from './BufferManager'
import { SegmentFetcher } from './SegmentFetcher'
import { SeekController } from './SeekController'
import type { SegmentInfo } from '../hlsParser'
import { getSignedUrls } from '../hlsParser'

export interface PlaybackConfig {
  initialBufferSize: number
  maxBufferSize: number
  bufferAheadTime: number
  autoRecover: boolean
}

export interface PlaybackState {
  isInitialized: boolean
  isLoading: boolean
  isSeeking: boolean
  currentSegmentIndex: number
  hasError: boolean
  errorCount: number
}

export class PlaybackOrchestrator {
  private video: HTMLVideoElement
  private bufferManager: BufferManager
  private segmentFetcher: SegmentFetcher
  private seekController: SeekController
  
  private config: PlaybackConfig
  private segments: SegmentInfo[] = []
  private movieId: string = ''
  private guestToken: string | undefined
  
  private state: PlaybackState = {
    isInitialized: false,
    isLoading: false,
    isSeeking: false,
    currentSegmentIndex: 0,
    hasError: false,
    errorCount: 0
  }

  private manageBufferTimer: number | null = null
  private readonly manageBufferInterval = 1000 // 1 second

  constructor(video: HTMLVideoElement, config: Partial<PlaybackConfig> = {}) {
    this.video = video
    
    this.config = {
      initialBufferSize: 3,
      maxBufferSize: 10,
      bufferAheadTime: 30,
      autoRecover: true,
      ...config
    }

    this.bufferManager = new BufferManager(video)
    this.segmentFetcher = new SegmentFetcher()
    this.seekController = new SeekController()

    this.setupEventListeners()
  }

  async initialize(segments: SegmentInfo[], movieId: string, guestToken?: string, duration?: number): Promise<void> {
    try {
      console.log('🚀 Initializing PlaybackOrchestrator...')
      
      this.segments = segments
      this.movieId = movieId
      this.guestToken = guestToken
      
      // Initialize components
      await this.bufferManager.initialize()
      this.seekController.setSegments(segments)
      
      // Set duration if provided
      if (duration) {
        this.bufferManager.setDuration(duration)
      }
      
      this.state.isInitialized = true
      this.state.hasError = false
      this.state.errorCount = 0
      
      console.log('✅ PlaybackOrchestrator initialized')
      
      // Start buffer management
      this.startBufferManagement()
      
    } catch (error) {
      console.error('❌ Failed to initialize PlaybackOrchestrator:', error)
      this.state.hasError = true
      throw error
    }
  }

  async seek(targetTime: number): Promise<void> {
    if (!this.state.isInitialized || this.state.isSeeking) {
      console.warn('⚠️ Cannot seek: not initialized or already seeking')
      return
    }

    try {
      console.log(`🎯 Seeking to ${targetTime}s`)
      this.state.isSeeking = true

      // Use seek controller to determine strategy
      const seekResult = this.seekController.seekUsingNearestSegment(targetTime, 3)
      
      // Load the target segment and preload segments
      for (const segmentIndex of seekResult.preloadSegments) {
        const success = await this.loadSegment(segmentIndex)
        if (!success) {
          console.warn(`⚠️ Failed to load segment ${segmentIndex} during seek`)
        }
      }

      // Adjust video position if needed
      const bufferedRanges = this.bufferManager.getBufferedRanges()
      const closestRange = this.seekController.findClosestBufferedRange(targetTime, bufferedRanges)
      
      if (closestRange.adjustedTime !== null && Math.abs(closestRange.distance) < 5.0) {
        console.log(`🔧 Adjusting video position from ${this.video.currentTime}s to ${closestRange.adjustedTime}s`)
        this.video.currentTime = closestRange.adjustedTime
      }

      this.state.currentSegmentIndex = seekResult.targetSegmentIndex

    } catch (error) {
      console.error('❌ Seek failed:', error)
      this.state.hasError = true
    } finally {
      this.state.isSeeking = false
    }
  }

  private async loadSegment(segmentIndex: number): Promise<boolean> {
    if (segmentIndex < 0 || segmentIndex >= this.segments.length) {
      console.warn(`⚠️ Invalid segment index: ${segmentIndex}`)
      return false
    }

    // Check if buffer manager is ready
    if (this.bufferManager.hasErrors) {
      console.warn('⚠️ BufferManager has errors, attempting recovery...')
      try {
        await this.bufferManager.recover()
      } catch (error) {
        console.error('❌ Buffer recovery failed:', error)
        return false
      }
    }

    if (!this.bufferManager.ready) {
      console.warn('⚠️ BufferManager not ready')
      return false
    }

    try {
      const segment = this.segments[segmentIndex]
      const segmentPath = `1080p/${segment.filename}`
      
      console.log(`🔄 Loading segment ${segmentIndex + 1} of ${this.segments.length}`)
      
      // Get signed URL
      const urlResponse = await getSignedUrls(this.movieId, segmentPath, this.guestToken)
      const signedUrl = urlResponse.file_urls[segmentPath]
      
      if (!signedUrl) {
        throw new Error(`No signed URL for segment ${segmentPath}`)
      }

      // Fetch segment data
      const segmentData = await this.segmentFetcher.fetchSegment(signedUrl, segmentPath)
      
      // Append to buffer
      const success = await this.bufferManager.appendSegment(segmentData)
      
      if (success) {
        console.log(`✅ Segment ${segmentIndex} loaded and appended successfully`)
        this.logVideoState(`after segment ${segmentIndex} appended`)
        return true
      } else {
        console.warn(`⚠️ Failed to append segment ${segmentIndex}`)
        return false
      }

    } catch (error) {
      console.error(`❌ Error loading segment ${segmentIndex}:`, error)
      this.state.errorCount++
      
      // Auto-recovery if enabled
      if (this.config.autoRecover && this.state.errorCount >= 3) {
        console.log('🔄 Triggering auto-recovery...')
        await this.recover()
      }
      
      return false
    }
  }

  private startBufferManagement(): void {
    if (this.manageBufferTimer) {
      clearInterval(this.manageBufferTimer)
    }

    this.manageBufferTimer = window.setInterval(() => {
      this.manageBuffer()
    }, this.manageBufferInterval)

    // Initial buffer load
    this.manageBuffer()
  }

  private async manageBuffer(): Promise<void> {
    if (!this.state.isInitialized || this.state.isSeeking || this.state.isLoading) {
      return
    }

    this.state.isLoading = true

    try {
      // Load initial segments
      if (this.state.currentSegmentIndex < this.config.initialBufferSize) {
        const success = await this.loadSegment(this.state.currentSegmentIndex)
        if (success) {
          this.state.currentSegmentIndex++
          
          // after loading first few segments, just continue with buffer management
          // don't auto-position to avoid DTS sequence errors
        }
        return
      }

      // Progressive buffer management
      const currentTime = this.video.currentTime
      const bufferedRanges = this.bufferManager.getBufferedRanges()
      
      // Find how much is buffered ahead
      let bufferAhead = 0
      for (const range of bufferedRanges) {
        if (currentTime >= range.start && currentTime <= range.end) {
          bufferAhead = range.end - currentTime
          break
        }
      }

      // Load more segments if buffer is low
      if (bufferAhead < this.config.bufferAheadTime && this.state.currentSegmentIndex < this.segments.length) {
        const success = await this.loadSegment(this.state.currentSegmentIndex)
        if (success) {
          this.state.currentSegmentIndex++
        }
      }

      // Evict old segments
      if (currentTime > 60) { // Start evicting after 1 minute
        this.bufferManager.evictOldSegments(currentTime)
      }

    } catch (error) {
      console.error('❌ Error in buffer management:', error)
      this.state.hasError = true
    } finally {
      this.state.isLoading = false
    }
  }

  private setupEventListeners(): void {
    this.video.addEventListener('seeking', () => {
      if (!this.state.isSeeking) {
        const targetTime = this.video.currentTime
        console.log(`🎯 User seeking to: ${targetTime}s`)
        this.seek(targetTime)
      }
    })

    this.video.addEventListener('error', (event) => {
      console.error('🚨 Video error:', event, this.video.error)
      this.state.hasError = true
      
      // disable auto-recovery to prevent infinite loops
      // if (this.config.autoRecover) {
      //   console.log('🔄 Auto-recovery triggered by video error')
      //   this.recover()
      // }
      console.log('❌ Video error occurred - auto-recovery disabled to prevent loops')
    })

    this.video.addEventListener('loadedmetadata', () => {
      console.log('📺 Video metadata loaded, duration:', this.video.duration)
    })

    this.video.addEventListener('canplay', () => {
      console.log('▶️ Video can start playing')
    })
  }

  async recover(): Promise<void> {
    try {
      console.log('🔄 Starting PlaybackOrchestrator recovery...')
      
      // Stop buffer management
      if (this.manageBufferTimer) {
        clearInterval(this.manageBufferTimer)
        this.manageBufferTimer = null
      }

      // Save current video time to resume from correct position
      const currentTime = this.video.currentTime
      console.log(`💾 Saving video time for recovery: ${currentTime}s`)

      // Reset state - after recovery we start fresh with a new MediaSource
      this.state = {
        isInitialized: false,
        isLoading: false,
        isSeeking: false,
        currentSegmentIndex: 0,
        hasError: false,
        errorCount: 0
      }

      // Clear caches
      this.segmentFetcher.clearCache()

      // Recover buffer manager
      await this.bufferManager.recover()

      // Reinitialize
      if (this.segments.length > 0) {
        const duration = this.segments.reduce((sum, seg) => sum + seg.duration, 0)
        await this.initialize(this.segments, this.movieId, this.guestToken, duration)
        
        // After recovery, we need to start from segment 0 for MediaSource to work correctly
        // but we'll seek to the correct position after initial segments are loaded
        this.state.currentSegmentIndex = 0
        console.log(`🎯 Restarting from segment 0 after recovery, will seek to ${currentTime}s later`)
        
        // Set a flag to seek after initial buffer is established
        if (currentTime > 0) {
          // Wait a bit for initial segments to load, then seek
          setTimeout(() => {
            try {
              console.log(`🎯 Seeking to saved position ${currentTime}s after recovery`)
              this.video.currentTime = currentTime
            } catch (error) {
              console.error('❌ Failed to seek after recovery:', error)
            }
          }, 2000) // Wait 2 seconds for initial buffer
        }
      }

      console.log('✅ PlaybackOrchestrator recovery completed')
    } catch (error) {
      console.error('❌ PlaybackOrchestrator recovery failed:', error)
      this.state.hasError = true
      throw error
    }
  }



  private logVideoState(context: string): void {
    const video = this.video
    console.log(`[${context}] Video state:`)
    console.log(`  readyState: ${video.readyState} (0=HAVE_NOTHING, 1=HAVE_METADATA, 2=HAVE_CURRENT_DATA, 3=HAVE_FUTURE_DATA, 4=HAVE_ENOUGH_DATA)`)
    console.log(`  paused: ${video.paused}`)
    console.log(`  currentTime: ${video.currentTime}`)
    console.log(`  duration: ${video.duration}`)
    console.log(`  networkState: ${video.networkState}`)
    console.log(`  buffered ranges: ${video.buffered.length}`)
    
    for (let i = 0; i < video.buffered.length; i++) {
      console.log(`    range ${i}: ${video.buffered.start(i).toFixed(2)}s - ${video.buffered.end(i).toFixed(2)}s`)
    }
  }

  getState(): Readonly<PlaybackState> {
    return { ...this.state }
  }

  dispose(): void {
    if (this.manageBufferTimer) {
      clearInterval(this.manageBufferTimer)
      this.manageBufferTimer = null
    }

    this.bufferManager.dispose()
    this.segmentFetcher.clearCache()
    
    this.state.isInitialized = false
  }
}
