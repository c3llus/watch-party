/**
 * BufferManager - Wraps and abstracts MediaSource + SourceBuffer
 * Handles appending buffers with error recovery
 */

export interface BufferedRange {
  start: number
  end: number
}

export class BufferManager {
  private mediaSource: MediaSource | null = null
  private sourceBuffer: SourceBuffer | null = null
  private video: HTMLVideoElement
  private mimeType: string
  private isInitialized = false
  private errorCount = 0
  private readonly maxErrors = 3

  constructor(video: HTMLVideoElement, mimeType: string = 'video/mp2t; codecs="avc1.64001f,mp4a.40.2"') {
    this.video = video
    this.mimeType = mimeType
  }

  async initialize(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.mediaSource = new MediaSource()
      
      // attach MediaSource to video element first
      this.video.src = URL.createObjectURL(this.mediaSource)
      
      this.mediaSource.addEventListener('sourceopen', () => {
        try {
          if (!this.mediaSource) {
            reject(new Error('MediaSource is null'))
            return
          }

          console.log('🔗 MediaSource opened, adding SourceBuffer...')
          this.sourceBuffer = this.mediaSource.addSourceBuffer(this.mimeType)
          
          this.sourceBuffer.addEventListener('error', (event) => {
            console.error('🚨 SourceBuffer error:', event)
            this.handleSourceBufferError()
          })

          this.sourceBuffer.addEventListener('updateend', () => {
            console.log('✅ SourceBuffer update ended')
          })

          this.isInitialized = true
          console.log('✅ BufferManager initialized')
          resolve()
        } catch (error) {
          console.error('❌ Failed to initialize SourceBuffer:', error)
          reject(error)
        }
      })

      this.mediaSource.addEventListener('sourceclose', () => {
        console.log('📤 MediaSource closed')
        this.isInitialized = false
      })

      this.mediaSource.addEventListener('error', (event) => {
        console.error('🚨 MediaSource error:', event)
        reject(new Error('MediaSource error'))
      })
    })
  }

  async appendSegment(segment: ArrayBuffer): Promise<boolean> {
    if (!this.isInitialized || !this.sourceBuffer || this.sourceBuffer.updating) {
      console.warn('⚠️ BufferManager not ready for append')
      return false
    }

    // Check if video element has errors
    if (this.video.error) {
      console.error('❌ Video element has error, cannot append:', this.video.error.message)
      
      // If we've had too many errors, attempt recovery
      if (this.errorCount >= this.maxErrors) {
        console.log('🔄 Too many errors, attempting recovery...')
        await this.recover()
        return false
      }
      
      this.errorCount++
      return false
    }

    return new Promise((resolve) => {
      if (!this.sourceBuffer) {
        resolve(false)
        return
      }

      const handleUpdateEnd = () => {
        this.sourceBuffer!.removeEventListener('updateend', handleUpdateEnd)
        this.sourceBuffer!.removeEventListener('error', handleError)
        console.log('✅ Segment appended successfully')
        this.errorCount = 0 // Reset error count on success
        resolve(true)
      }

      const handleError = (event: Event) => {
        this.sourceBuffer!.removeEventListener('updateend', handleUpdateEnd)
        this.sourceBuffer!.removeEventListener('error', handleError)
        console.error('❌ Error appending segment:', event)
        this.errorCount++
        resolve(false)
      }

      this.sourceBuffer.addEventListener('updateend', handleUpdateEnd)
      this.sourceBuffer.addEventListener('error', handleError)

      try {
        this.sourceBuffer.appendBuffer(segment)
      } catch (error) {
        this.sourceBuffer.removeEventListener('updateend', handleUpdateEnd)
        this.sourceBuffer.removeEventListener('error', handleError)
        console.error('❌ Exception appending segment:', error)
        this.errorCount++
        resolve(false)
      }
    })
  }

  setDuration(duration: number): void {
    if (this.mediaSource && this.mediaSource.readyState === 'open') {
      // Ensure duration is a valid number, not Infinity or NaN
      if (isFinite(duration) && duration > 0) {
        this.mediaSource.duration = duration
        console.log('✅ MediaSource duration set to:', duration)
      } else {
        console.warn('⚠️ Invalid duration provided:', duration)
      }
    }
  }

  getBufferedRanges(): BufferedRange[] {
    if (!this.sourceBuffer) return []
    
    const ranges: BufferedRange[] = []
    const buffered = this.sourceBuffer.buffered
    
    for (let i = 0; i < buffered.length; i++) {
      ranges.push({
        start: buffered.start(i),
        end: buffered.end(i)
      })
    }
    
    return ranges
  }

  evictOldSegments(beforeTime: number): void {
    if (!this.sourceBuffer || this.sourceBuffer.updating) return

    try {
      // Only remove if there's a significant amount to remove (> 30 seconds)
      if (beforeTime > 30) {
        const removeEnd = beforeTime - 10 // Keep 10 seconds buffer
        this.sourceBuffer.remove(0, removeEnd)
        console.log(`🧹 Evicted segments before ${removeEnd}s`)
      }
    } catch (error) {
      console.error('❌ Error evicting segments:', error)
    }
  }

  private handleSourceBufferError(): void {
    console.error('🚨 SourceBuffer error detected')
    this.errorCount++
    
    if (this.errorCount >= this.maxErrors) {
      console.log('🔄 Triggering recovery due to repeated errors...')
      this.recover()
    }
  }

  async recover(): Promise<void> {
    console.log('🔄 Starting BufferManager recovery...')
    
    try {
      // Reset error count
      this.errorCount = 0
      
      // Close current MediaSource
      if (this.mediaSource && this.mediaSource.readyState === 'open') {
        this.mediaSource.endOfStream()
      }
      
      // Clear video src
      this.video.src = ''
      this.video.load()
      
      // Reset state
      this.mediaSource = null
      this.sourceBuffer = null
      this.isInitialized = false
      
      // Wait a bit for cleanup
      await new Promise(resolve => setTimeout(resolve, 100))
      
      // Reinitialize
      await this.initialize()
      
      console.log('✅ BufferManager recovery completed')
    } catch (error) {
      console.error('❌ BufferManager recovery failed:', error)
      throw error
    }
  }

  dispose(): void {
    if (this.mediaSource && this.mediaSource.readyState === 'open') {
      this.mediaSource.endOfStream()
    }
    
    this.video.src = ''
    this.mediaSource = null
    this.sourceBuffer = null
    this.isInitialized = false
  }

  get ready(): boolean {
    return this.isInitialized && !!this.sourceBuffer && !this.sourceBuffer.updating
  }

  get hasErrors(): boolean {
    return !!this.video.error || this.errorCount > 0
  }
}
