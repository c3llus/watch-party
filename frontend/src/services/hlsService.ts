// src/services/hlsService.ts

import { 
  getMasterPlaylist, 
  getMediaPlaylist
} from '../utils/hlsParser'

// set to true to enable detailed HLS streaming logs for debugging
const HLS_DEBUG_LOGGING = false 

// helper function for conditional logging
const hlsLog = (...args: unknown[]) => {
  if (HLS_DEBUG_LOGGING) {
    console.log(...args)
  }
}

// ===== TYPES AND INTERFACES =====

export interface HlsPlayerConfig {
  initialBufferSize: number;
  bufferAheadTime: number;
  cacheExpiryHours: number;
  onFatalError?: (error: string) => void;
}

export interface Segment {
  index: number;
  duration: number;
  url: string;
  startTime: number;
}

export interface SegmentCache {
  url: string;
  expiresAt: number;
}

export interface HlsPlayer {
  start(): Promise<void>;
  seekTo(time: number): Promise<void>;
  restart(targetTime?: number): Promise<void>;
  cleanup(): void;
  isInitialized(): boolean;
}

// segment task for queue processing
interface SegmentTask {
  index: number;
  priority: number; // lower number = higher priority
  generationId: number; // generation ID to prevent stale appends
}

// ===== DEFAULT CONFIGURATION =====

const DEFAULT_CONFIG: HlsPlayerConfig = {
  initialBufferSize: 3,
  bufferAheadTime: 10,
  cacheExpiryHours: 1.5
}

// ===== HLS INITIALIZATION MODULE =====

export function createHlsPlayer(
  videoEl: HTMLVideoElement, 
  masterUrlResolver: () => Promise<string>,
  getSegmentUrl: (uri: string) => Promise<string>,
  config: Partial<HlsPlayerConfig> = {}
): HlsPlayer {
  const finalConfig = { ...DEFAULT_CONFIG, ...config }
  let isInitialized = false
  let isInitializing = false
  let mediaSource: MediaSource | null = null
  let sourceBuffer: SourceBuffer | null = null
  let segments: Segment[] = []
  let currentSegmentIndex = 0
  let hasStartedPlayback = false
  let isUnmounted = false
  let isAppending = false
  let revokeUrls: string[] = []
  let programmaticSeekTime: number | null = null // track programmatic seeks more reliably
  const loadedSegments = new Set<number>() // track which segments have been loaded
  let appendLock = false // prevent concurrent append operations
  
  // state machine and queue processing
  let playerState = 'idle'
  let abortController = new AbortController()
  const segmentQueue: SegmentTask[] = []
  let isProcessingQueue = false
  let currentGenerationId = 0 // generation ID to prevent stale operations
  
  // error recovery tracking
  let errorRecoveryCount = 0
  let lastErrorRecoveryTime = 0
  let lastErrorEventTime = 0 // prevent cascading error events
  const ERROR_RECOVERY_COOLDOWN_MS = 5000 // 5 seconds between error recovery attempts
  const MAX_ERROR_RECOVERY_ATTEMPTS = 3 // max attempts before giving up
  const ERROR_EVENT_DEBOUNCE_MS = 1000 // 1 second debounce for error events
  
  // url cache for segments with expiry tracking
  const segmentUrlCache = new Map<number, SegmentCache>()
  let totalDuration = 0

  // ===== CODEC DETECTION AND MIME TYPE =====

  function buildMimeType(codecs?: string): string {
    if (!codecs) {
      hlsLog('no codecs detected, falling back to H.264/AAC')
      return 'video/mp2t; codecs="avc1.42E01E,mp4a.40.2"'
    }

    const hasVideo = codecs.toLowerCase().includes('avc1') || codecs.toLowerCase().includes('h264')
    const hasAudio = codecs.toLowerCase().includes('mp4a') || codecs.toLowerCase().includes('aac')

    if (hasVideo && hasAudio) {
      hlsLog('detected video + audio codecs:', codecs)
      return `video/mp2t; codecs="${codecs}"`
    } else if (hasVideo) {
      hlsLog('detected video-only codecs, adding audio:', codecs)
      const videoCodec = codecs.split(',').find(c => c.trim().toLowerCase().includes('avc1'))?.trim() || 'avc1.42E01E'
      return `video/mp2t; codecs="${videoCodec},mp4a.40.2"`
    } else {
      hlsLog('unknown codec format, falling back to H.264/AAC:', codecs)
      return 'video/mp2t; codecs="avc1.42E01E,mp4a.40.2"'
    }
  }

  // ===== MEDIASOURCE AND SOURCEBUFFER MANAGEMENT =====

  function setupMediaSource(mimeType: string): Promise<{ mediaSource: MediaSource; sourceBuffer: SourceBuffer }> {
    return new Promise((resolve, reject) => {
      if (!('MediaSource' in window)) {
        reject(new Error('MediaSource not supported'))
        return
      }

      // Clean up any existing MediaSource first
      if (videoEl.src && videoEl.src.startsWith('blob:')) {
        hlsLog('revoking existing blob URL:', videoEl.src)
        URL.revokeObjectURL(videoEl.src)
        videoEl.src = ''
        videoEl.load() // force video element to reset
      }

      hlsLog('creating new MediaSource...')
      const ms = new MediaSource()
      hlsLog('mediaSource created, initial readyState:', ms.readyState)
      
      const timeoutId = setTimeout(() => {
        hlsLog('MediaSource timeout, cleaning up...')
        if (ms.readyState !== 'closed') {
          try {
            ms.endOfStream()
          } catch {
            // ignore errors during cleanup
          }
        }
        reject(new Error('MediaSource sourceopen event timeout'))
      }, 15000)

      const cleanup = () => {
        clearTimeout(timeoutId)
        ms.removeEventListener('sourceopen', onSourceOpen)
        ms.removeEventListener('error', onError)
      }

      const onSourceOpen = () => {
        cleanup()
        hlsLog('mediaSource sourceopen event fired, readyState:', ms.readyState)
        
        // double-check MediaSource is still open
        if (ms.readyState !== 'open') {
          reject(new Error(`MediaSource readyState is ${ms.readyState}, expected 'open'`))
          return
        }
        
        try {
          const sb = ms.addSourceBuffer(mimeType)
          hlsLog('sourceBuffer created successfully with mime type:', mimeType)
          
          // set duration if we have it
          if (totalDuration > 0) {
            try {
              ms.duration = totalDuration
              hlsLog('mediaSource duration set to:', totalDuration, 'seconds')
            } catch (error) {
              console.warn('failed to set MediaSource duration:', error)
            }
          }

          resolve({ mediaSource: ms, sourceBuffer: sb })
        } catch (error) {
          console.error('failed to create SourceBuffer:', error, 'MediaSource readyState:', ms.readyState)
          reject(error)
        }
      }

      const onError = (e: Event) => {
        cleanup()
        console.error('MediaSource error event:', e)
        reject(new Error('MediaSource error'))
      }

      ms.addEventListener('sourceopen', onSourceOpen)
      ms.addEventListener('error', onError)
      
      // Set video source and trigger load
      videoEl.src = URL.createObjectURL(ms)
      revokeUrls.push(videoEl.src)
      hlsLog('video src set, waiting for sourceopen event...')
      
      // force load to trigger sourceopen event
      videoEl.load()
    })
  }

  // ===== PLAYLIST AND SEGMENT PARSING =====

  async function parsePlaylistAndSegments(masterUrl: string): Promise<{ segments: Segment[]; codecs?: string }> {
    // get master playlist
    hlsLog('fetching master playlist from:', masterUrl)
    const master = await getMasterPlaylist(masterUrl)
    
    if (!master.variants.length) {
      throw new Error('no variants in master playlist')
    }

    // pick highest bandwidth variant
    const bestVariant = master.variants.reduce((a, b) => 
      a.bandwidth > b.bandwidth ? a : b
    )
    hlsLog('selected variant:', bestVariant)

    // extract codecs from the best variant
    const detectedCodecs = bestVariant.codecs
    hlsLog('detected codecs from master playlist:', detectedCodecs)

    // extract the relative path from the variant URL and get a signed URL for it
    const variantUrl = new URL(bestVariant.url)
    const variantPath = variantUrl.pathname.split('/').slice(-2).join('/') // e.g., "1080p/playlist.m3u8"
    hlsLog('getting signed URL for media playlist path:', variantPath)
    const signedMediaPlaylistUrl = await getSegmentUrl(variantPath)
    hlsLog('signed media playlist URL:', signedMediaPlaylistUrl)

    // get media playlist using signed URL
    hlsLog('fetching media playlist from signed URL')
    const media = await getMediaPlaylist(signedMediaPlaylistUrl)
    
    if (!media.segments.length) {
      throw new Error('no segments in media playlist')
    }

    // convert to internal segment format with calculated start times
    let currentTime = 0
    const processedSegments: Segment[] = media.segments.map((segment, index) => {
      const seg: Segment = {
        index,
        duration: segment.duration,
        url: segment.url,
        startTime: currentTime
      }
      currentTime += segment.duration
      return seg
    })

    totalDuration = currentTime
    hlsLog('processed', processedSegments.length, 'segments, total duration:', totalDuration)
    
    return { segments: processedSegments, codecs: detectedCodecs }
  }

  // ===== SEGMENT FETCHING =====

  async function fetchSegmentBuffer(segmentIndex: number): Promise<ArrayBuffer> {
    const segment = segments[segmentIndex]
    if (!segment) {
      throw new Error(`segment ${segmentIndex} not found`)
    }

    // check cache first
    const cached = segmentUrlCache.get(segmentIndex)
    const now = Date.now()
    
    let segmentUrl: string

    if (cached && cached.expiresAt > now) {
      hlsLog(`using cached URL for segment ${segmentIndex}`)
      segmentUrl = cached.url
    } else {
      hlsLog(`fetching new signed URL for segment ${segmentIndex}`)
      segmentUrl = await getSegmentUrl(segment.url)
      
      // cache the URL
      segmentUrlCache.set(segmentIndex, {
        url: segmentUrl,
        expiresAt: now + (finalConfig.cacheExpiryHours * 60 * 60 * 1000)
      })
    }

    const response = await fetch(segmentUrl, {
      signal: abortController.signal
    })
    if (!response.ok) {
      throw new Error(`failed to fetch segment ${segmentIndex}: ${response.status}`)
    }

    return await response.arrayBuffer()
  }

  // ===== SEEK LOGIC =====

  function getSegmentForTime(targetTime: number): Segment | null {
    if (!segments.length) return null

    // binary search for efficiency
    let left = 0
    let right = segments.length - 1

    while (left <= right) {
      const mid = Math.floor((left + right) / 2)
      const segment = segments[mid]
      const segmentEnd = segment.startTime + segment.duration

      if (targetTime >= segment.startTime && targetTime < segmentEnd) {
        return segment
      } else if (targetTime < segment.startTime) {
        right = mid - 1
      } else {
        left = mid + 1
      }
    }

    // if not found exactly, return last segment
    return segments[segments.length - 1]
  }

  function getBufferStatus() {
    // Check if sourceBuffer is valid and not removed from MediaSource
    if (!sourceBuffer || !mediaSource || mediaSource.readyState === 'closed') {
      return { bufferedEnd: 0, bufferAhead: 0, needsData: true }
    }
    
    try {
      if (sourceBuffer.buffered.length === 0) {
        return { bufferedEnd: 0, bufferAhead: 0, needsData: true }
      }
    } catch (error) {
      console.warn('sourceBuffer.buffered access failed, likely removed from MediaSource:', error)
      return { bufferedEnd: 0, bufferAhead: 0, needsData: true }
    }
    
    const currentTime = videoEl.currentTime
    let bufferedEnd = 0
    
    // Debug: log all buffer ranges
    hlsLog(`Buffer ranges: ${sourceBuffer.buffered.length} ranges at time ${currentTime.toFixed(2)}s`)
    for (let i = 0; i < sourceBuffer.buffered.length; i++) {
      const start = sourceBuffer.buffered.start(i)
      const end = sourceBuffer.buffered.end(i)
      hlsLog(`  Range ${i}: ${start.toFixed(2)}s - ${end.toFixed(2)}s`)
    }
    
    // find the buffered range that contains current time
    for (let i = 0; i < sourceBuffer.buffered.length; i++) {
      const start = sourceBuffer.buffered.start(i)
      const end = sourceBuffer.buffered.end(i)
      if (currentTime >= start && currentTime <= end) {
        bufferedEnd = end
        break
      }
      if (start > currentTime) {
        bufferedEnd = i > 0 ? sourceBuffer.buffered.end(i - 1) : 0
        break
      }
    }
    
    const bufferAhead = bufferedEnd - currentTime
    const needsData = bufferAhead < finalConfig.bufferAheadTime
    
    hlsLog(`Buffer status: bufferedEnd=${bufferedEnd.toFixed(2)}s, bufferAhead=${bufferAhead.toFixed(2)}s, needsData=${needsData}`)
    
    return { bufferedEnd, bufferAhead, needsData }
  }

  // ===== SERIALIZED QUEUE PROCESSING =====

  function addSegmentToQueue(segmentIndex: number, priority: number = 0): void {
    // avoid duplicates in queue
    if (segmentQueue.some(task => task.index === segmentIndex)) {
      return
    }
    
    segmentQueue.push({ index: segmentIndex, priority, generationId: currentGenerationId })
    
    // sort by priority (lower number = higher priority)
    segmentQueue.sort((a, b) => a.priority - b.priority)
    
    // trigger processing if not already running
    if (!isProcessingQueue && playerState === 'idle') {
      processSegmentQueue()
    }
  }

  async function processSegmentQueue(): Promise<void> {
    // prevent concurrent processing
    if (isProcessingQueue || playerState !== 'idle') {
      return
    }
    
    // check if queue is empty
    if (segmentQueue.length === 0) {
      return
    }
    
    isProcessingQueue = true
    
    try {
      while (segmentQueue.length > 0 && playerState === 'idle') {
        const task = segmentQueue.shift()
        if (!task) break
        
        // check generation ID to prevent stale segment loads
        if (task.generationId !== currentGenerationId) {
          hlsLog(`skipping stale segment ${task.index} (generation ${task.generationId} vs current ${currentGenerationId})`)
          continue
        }
        
        // check if segment already loaded
        if (loadedSegments.has(task.index)) {
          continue
        }
        
        // check if we're still in the right state
        if (playerState !== 'idle') {
          hlsLog('queue processing stopped, player state changed to:', playerState)
          break
        }
        
        // double-check generation before proceeding (race condition protection)
        if (task.generationId !== currentGenerationId) {
          hlsLog(`generation changed during processing for segment ${task.index}`)
          continue
        }
        
        try {
          await loadSegmentSequential(task.index, task.generationId)
        } catch (error) {
          console.error('failed to load segment', task.index, 'from queue:', error)
          // continue with next segment rather than stopping entirely
        }
      }
    } finally {
      isProcessingQueue = false
    }
  }

  async function loadSegmentSequential(segmentIndex: number, generationId: number): Promise<void> {
    if (segmentIndex >= segments.length || !sourceBuffer || playerState !== 'idle') {
      return
    }
    
    // check generation ID to prevent stale appends
    if (generationId !== currentGenerationId) {
      hlsLog(`skipping stale segment append ${segmentIndex} (generation ${generationId} vs current ${currentGenerationId})`)
      return
    }
    
    // check if already loaded
    if (loadedSegments.has(segmentIndex)) {
      return
    }
    
    // check append lock to prevent concurrent appends
    if (appendLock) {
      hlsLog(`skipping segment ${segmentIndex} append - another append in progress`)
      return
    }
    
    try {
      // enter loading state
      playerState = 'loading'
      hlsLog('loading segment', segmentIndex + 1, 'of', segments.length, `(generation ${generationId})`)
      
      const buffer = await fetchSegmentBuffer(segmentIndex)
      hlsLog('segment', segmentIndex, 'loaded, size:', buffer.byteLength, 'bytes')
      
      // check state and generation again before appending
      if (playerState !== 'loading' || generationId !== currentGenerationId) {
        hlsLog(`skipping append for segment ${segmentIndex} due to state change or stale generation`)
        playerState = 'idle'
        return
      }
      
      // acquire append lock
      appendLock = true
      
      // enter appending state
      playerState = 'appending'
      
      return new Promise<void>((resolve, reject) => {
        const handleUpdateEnd = () => {
          if (!sourceBuffer) return
          sourceBuffer.removeEventListener('updateend', handleUpdateEnd)
          
          // release append lock
          appendLock = false
          
          // check generation one more time to ensure we're not adding stale data
          if (generationId !== currentGenerationId) {
            hlsLog(`segment ${segmentIndex} appended but generation is stale, ignoring`)
            playerState = 'idle'
            resolve()
            return
          }
          
          loadedSegments.add(segmentIndex)
          currentSegmentIndex = Math.max(currentSegmentIndex, segmentIndex + 1)
          hlsLog('segment', segmentIndex, 'appended successfully', `(generation ${generationId})`)
          
          // try to start playback after initial segments
          if (!hasStartedPlayback && currentSegmentIndex >= finalConfig.initialBufferSize && videoEl.readyState >= 2) {
            startInitialPlayback()
          }
          
          // return to idle state
          playerState = 'idle'
          resolve()
        }
        
        const handleError = (e: Event) => {
          if (!sourceBuffer) return
          sourceBuffer.removeEventListener('error', handleError)
          
          // release append lock
          appendLock = false
          
          console.error('sourceBuffer error during segment', segmentIndex, 'append:', e)
          
          // check if video element has entered error state
          if (videoEl.error) {
            console.error('video element error detected:', videoEl.error)
            playerState = 'error'
            // trigger error recovery with cooldown
            scheduleErrorRecovery()
          } else {
            // return to idle state for non-fatal errors
            playerState = 'idle'
          }
          
          reject(new Error(`SourceBuffer error during segment ${segmentIndex} append`))
        }
        
        if (!sourceBuffer) {
          appendLock = false
          playerState = 'idle'
          reject(new Error('SourceBuffer is null'))
          return
        }
        
        // final generation check before actual append
        if (generationId !== currentGenerationId) {
          appendLock = false
          playerState = 'idle'
          hlsLog(`final generation check failed for segment ${segmentIndex}`)
          resolve()
          return
        }
        
        sourceBuffer.addEventListener('updateend', handleUpdateEnd, { once: true })
        sourceBuffer.addEventListener('error', handleError, { once: true })
        
        // safety check before appendBuffer
        if (!sourceBuffer || !mediaSource || mediaSource.readyState === 'closed') {
          appendLock = false
          playerState = 'idle'
          reject(new Error('SourceBuffer or MediaSource is no longer valid'))
          return
        }
        
        // check if video element is in error state
        if (videoEl.error) {
          appendLock = false
          console.warn('video element is in error state, cannot append buffer. Error:', videoEl.error)
          playerState = 'error'
          reject(new Error(`Video element error: ${videoEl.error.message || 'Unknown error'}`))
          return
        }
        
        sourceBuffer.appendBuffer(buffer)
      })
      
    } catch (error) {
      appendLock = false
      console.error('error loading segment', segmentIndex, ':', error)
      playerState = 'idle'
      throw error
    }
  }

  // ===== ERROR RECOVERY =====

  function scheduleErrorRecovery(): void {
    const now = Date.now()
    
    // check if we're in cooldown period
    if (now - lastErrorRecoveryTime < ERROR_RECOVERY_COOLDOWN_MS) {
      hlsLog(`error recovery in cooldown, waiting ${ERROR_RECOVERY_COOLDOWN_MS - (now - lastErrorRecoveryTime)}ms`)
      return
    }
    
    // check if we've exceeded max attempts
    if (errorRecoveryCount >= MAX_ERROR_RECOVERY_ATTEMPTS) {
      console.error(`max error recovery attempts (${MAX_ERROR_RECOVERY_ATTEMPTS}) exceeded, giving up`)
      playerState = 'error'
      
      // trigger fatal error callback to initiate full restart
      if (finalConfig.onFatalError) {
        const errorMsg = `HLS player failed after ${MAX_ERROR_RECOVERY_ATTEMPTS} recovery attempts: ${videoEl.error?.message || 'unknown error'}`
        hlsLog('triggering fatal error callback for automatic restart:', errorMsg)
        finalConfig.onFatalError(errorMsg)
      }
      
      return
    }
    
    errorRecoveryCount++
    lastErrorRecoveryTime = now
    
    hlsLog(`scheduling error recovery attempt ${errorRecoveryCount}/${MAX_ERROR_RECOVERY_ATTEMPTS}`)
    setTimeout(() => attemptErrorRecovery(), 100)
  }

  async function attemptErrorRecovery(): Promise<void> {
    if (!videoEl.error || playerState === 'seeking') {
      return
    }
    
    hlsLog(`attempting error recovery for video error: ${videoEl.error.message || 'Unknown error'} (attempt ${errorRecoveryCount}/${MAX_ERROR_RECOVERY_ATTEMPTS})`)
    
    try {
      const currentTime = videoEl.currentTime
      
      // for critical errors (like DTS sequence errors), perform aggressive reset
      if (shouldPerformAggressiveReset(videoEl.error)) {
        hlsLog('performing aggressive MediaSource reset for critical error')
        await performAggressiveReset(currentTime)
      } else {
        hlsLog('using seek-based recovery to time:', currentTime)
        // use our robust seek function for recovery
        await seekTo(currentTime)
      }
      
      // reset error recovery count on successful recovery
      if (!videoEl.error && playerState === 'idle') {
        hlsLog('error recovery successful, resetting error count')
        errorRecoveryCount = 0
        lastErrorRecoveryTime = 0 // reset cooldown timer as well
      }
      
    } catch (error) {
      console.error('error recovery failed:', error)
      playerState = 'error'
    }
  }

  function shouldPerformAggressiveReset(error: MediaError): boolean {
    // perform aggressive reset for media decode errors and unknown errors
    // these are typically unrecoverable without a complete reset
    return error.code === MediaError.MEDIA_ERR_DECODE || 
           error.code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED ||
           error.code === 4 || // MEDIA_ELEMENT_ERROR
           (error.message && (
             error.message.toLowerCase().includes('dts') ||
             error.message.toLowerCase().includes('empty src') ||
             error.message.toLowerCase().includes('chunk_demuxer_error')
           )) ||
           false
  }

  async function performAggressiveReset(targetTime: number): Promise<void> {
    hlsLog('performing aggressive MediaSource/SourceBuffer reset')
    
    try {
      // clear video element error state first
      if (videoEl.error) {
        hlsLog('clearing video element error state')
        // clear the src to reset error state
        videoEl.src = ''
        videoEl.load() // force reload to clear error
        // small delay to ensure error is cleared
        await new Promise(resolve => setTimeout(resolve, 50))
      }
      
      // increment generation to invalidate all pending operations
      currentGenerationId++
      hlsLog(`incremented generation ID to ${currentGenerationId}`)
      
      // cancel all operations
      abortController.abort('aggressive reset')
      abortController = new AbortController()
      
      // clear queue and state
      segmentQueue.length = 0
      isProcessingQueue = false
      appendLock = false // release any locks
      playerState = 'seeking'
      
      // remove and recreate MediaSource if needed
      if (mediaSource && sourceBuffer) {
        try {
          // try to remove the source buffer
          if (mediaSource.sourceBuffers.length > 0) {
            mediaSource.removeSourceBuffer(sourceBuffer)
          }
        } catch (e) {
          console.warn('failed to remove sourceBuffer, will recreate MediaSource:', e)
        }
      }
      
      // recreate MediaSource and SourceBuffer
      const masterUrl = await masterUrlResolver()
      const parseResult = await parsePlaylistAndSegments(masterUrl)
      segments = parseResult.segments
      const detectedCodecs = parseResult.codecs
      const mimeType = buildMimeType(detectedCodecs)
      const result = await setupMediaSource(mimeType)
      mediaSource = result.mediaSource
      sourceBuffer = result.sourceBuffer
      
      // reset all state
      currentSegmentIndex = 0
      hasStartedPlayback = false
      isAppending = false
      loadedSegments.clear()
      
      // preserve target time as much as possible - only reset to 0 as last resort
      let safeTargetTime = targetTime
      
      // only reset to 0 if target time is completely invalid
      if (!isFinite(targetTime) || targetTime < 0) {
        hlsLog(`target time ${targetTime} is invalid (negative or non-finite), using 0`)
        safeTargetTime = 0
      } else if (totalDuration && totalDuration > 0 && targetTime > totalDuration) {
        // if target exceeds duration, clamp to last few seconds instead of resetting to 0
        safeTargetTime = Math.max(0, totalDuration - 5)
        hlsLog(`target time ${targetTime} exceeds duration ${totalDuration}, clamping to ${safeTargetTime}`)
      } else if (!totalDuration || totalDuration <= 0) {
        // if duration is unknown but target time seems reasonable, trust it
        if (targetTime > 0 && targetTime < 7200) { // reasonable for videos under 2 hours
          hlsLog(`total duration not available (${totalDuration}), but target time ${targetTime} seems reasonable, keeping it`)
          safeTargetTime = targetTime
        } else {
          hlsLog(`total duration not available and target time ${targetTime} seems unreasonable, using 0`)
          safeTargetTime = 0
        }
      }
      
      // if no segments available, still try to preserve the target time
      if (segments.length === 0) {
        hlsLog(`no segments available during aggressive reset, but preserving target time ${safeTargetTime}`)
        // do not reset to 0 here - segments will be loaded after MediaSource is recreated
      }
      
      // find target segment and seek
      const targetSegment = getSegmentForTime(safeTargetTime)
      if (targetSegment) {
        currentSegmentIndex = targetSegment.index
        hlsLog(`reset complete, seeking to segment ${targetSegment.index} at time ${safeTargetTime}`)
      } else if (segments.length > 0) {
        // if no exact segment found but we have segments, estimate the index
        const estimatedIndex = Math.floor((safeTargetTime / totalDuration) * segments.length)
        currentSegmentIndex = Math.max(0, Math.min(estimatedIndex, segments.length - 1))
        hlsLog(`no exact segment found for time ${safeTargetTime}, estimated index ${currentSegmentIndex}`)
      } else {
        // no segments available yet, start from beginning but remember target time
        currentSegmentIndex = 0
        hlsLog(`no segments available during aggressive reset, starting from beginning but will seek to ${safeTargetTime} when ready`)
      }
      
      // update video current time to target regardless of segment availability
      programmaticSeekTime = safeTargetTime
      videoEl.currentTime = safeTargetTime
      requestAnimationFrame(() => {
        programmaticSeekTime = null
      })
      
      // exit seeking state and restart buffer management
      playerState = 'idle'
      
      // reset error recovery count since we successfully recovered
      errorRecoveryCount = 0
      hlsLog('aggressive reset completed successfully, state reset to idle')
      
      // manually trigger buffer management
      setTimeout(() => {
        if (playerState === 'idle') {
          processSegmentQueue()
        }
      }, 100) // slightly longer delay to ensure video element is ready
      
    } catch (error) {
      console.error('aggressive reset failed:', error)
      playerState = 'error'
      throw error
    }
  }

  // ===== PLAYBACK ORCHESTRATION =====

  function startInitialPlayback() {
    if (hasStartedPlayback) return
    
    hasStartedPlayback = true
    hlsLog('starting initial video playback')
    
    // ensure currentTime is within buffered range
    if (videoEl.buffered.length > 0) {
      const firstBufferStart = videoEl.buffered.start(0)
      if (videoEl.currentTime < firstBufferStart) {
        hlsLog(`adjusting currentTime from ${videoEl.currentTime} to ${firstBufferStart}`)
        programmaticSeekTime = firstBufferStart
        videoEl.currentTime = firstBufferStart
        // reset flag after a small delay to ensure event is processed
        requestAnimationFrame(() => {
          programmaticSeekTime = null
        })
      }
    }
    
    videoEl.play().catch(e => {
      console.error('failed to start playback:', e)
      hasStartedPlayback = false
    })
  }

  async function manageBuffer(): Promise<void> {
    if (isUnmounted || !videoEl || !sourceBuffer) return
    
    // check if player is in correct state for buffer management
    if (playerState !== 'idle') {
      // if we've been in error state for too long, try a recovery
      if (playerState === 'error') {
        hlsLog('manageBuffer: player state is error, checking for recovery opportunity')
        // if no error recovery is running and we're past cooldown, force a state reset
        if (errorRecoveryCount >= MAX_ERROR_RECOVERY_ATTEMPTS) {
          const timeSinceLastRecovery = Date.now() - lastErrorRecoveryTime
          if (timeSinceLastRecovery > ERROR_RECOVERY_COOLDOWN_MS * 2) {
            hlsLog('forcing player state reset after prolonged error state')
            
            // before resetting, try calling fatal error callback for automatic restart
            if (finalConfig.onFatalError && timeSinceLastRecovery > ERROR_RECOVERY_COOLDOWN_MS * 3) {
              const errorMsg = `HLS player stuck in error state for ${Math.round(timeSinceLastRecovery / 1000)}s after ${MAX_ERROR_RECOVERY_ATTEMPTS} recovery attempts`
              hlsLog('triggering fatal error callback for stuck error state:', errorMsg)
              finalConfig.onFatalError(errorMsg)
              return // let the callback handle the restart
            }
            
            playerState = 'idle'
            errorRecoveryCount = 0
            lastErrorRecoveryTime = 0
            lastErrorEventTime = 0
          }
        }
      }
      
      if (playerState !== 'idle') {
        hlsLog('manageBuffer: player state is', playerState, ', skipping buffer management')
        setTimeout(manageBuffer, 1000) // retry after delay
        return
      }
    }
    
    // check if video element is in error state
    if (videoEl.error) {
      console.warn('manageBuffer: video element is in error state, attempting recovery')
      attemptErrorRecovery()
      return
    }
    
    const { bufferedEnd, bufferAhead, needsData } = getBufferStatus()
    
    // load initial segments sequentially for quick start
    if (currentSegmentIndex < finalConfig.initialBufferSize) {
      hlsLog('loading initial segment', currentSegmentIndex)
      addSegmentToQueue(currentSegmentIndex, 0) // highest priority
      setTimeout(manageBuffer, 100)
      return
    }
    
    // fallback playback start
    if (!hasStartedPlayback && currentSegmentIndex >= finalConfig.initialBufferSize && videoEl.readyState >= 1) {
      startInitialPlayback()
    }
    
    // find next segment to load using intelligent logic
    if (needsData && playerState === 'idle') {
      const nextSegmentIndex = findNextSegmentToLoad()
      
      hlsLog(`loaded segments: [${Array.from(loadedSegments).sort((a, b) => a - b).join(', ')}]`)
      
      if (nextSegmentIndex !== null && nextSegmentIndex < segments.length) {
        const reason = getLoadReason(nextSegmentIndex)
        hlsLog(`${reason}: current=${videoEl.currentTime.toFixed(1)}s, buffered=${bufferedEnd.toFixed(1)}s, ahead=${bufferAhead?.toFixed(1)}s - queueing segment ${nextSegmentIndex}`)
        addSegmentToQueue(nextSegmentIndex, 1) // normal priority
      }
    }
    
    // check if we've loaded all segments
    if (currentSegmentIndex >= segments.length && mediaSource && mediaSource.readyState === 'open') {
      hlsLog('all segments loaded, ending MediaSource stream...')
      try {
        mediaSource.endOfStream()
      } catch (error) {
        console.warn('failed to end MediaSource stream:', error)
      }
      return
    }
    
    // continue buffer management
    setTimeout(manageBuffer, needsData ? 500 : 2000)
  }

  /**
   * Finds the next segment that should be loaded based on current playhead position,
   * buffer gaps, and buffer ahead requirements.
   * Priority: 1) Fill gaps around current time, 2) Maintain buffer ahead
   */
  function findNextSegmentToLoad(): number | null {
    if (!sourceBuffer || !videoEl || segments.length === 0) return null
    
    const currentTime = videoEl.currentTime
    const currentSegment = getSegmentForTime(currentTime)
    if (!currentSegment) return null
    
    // 1. Check for gaps around current playhead (high priority)
    const gapSegment = findGapAroundCurrentTime(currentTime)
    if (gapSegment !== null) {
      return gapSegment
    }
    
    // 2. Find segments needed to maintain buffer ahead
    const targetEndTime = currentTime + finalConfig.bufferAheadTime
    const segmentsNeeded = findSegmentsNeededForBufferAhead(currentTime, targetEndTime)
    
    if (segmentsNeeded.length > 0) {
      // return the earliest unloaded segment
      return segmentsNeeded[0]
    }
    
    return null
  }

  /**
   * Finds gaps in the buffer around the current playhead position.
   * Returns the index of the first missing segment that creates a gap.
   */
  function findGapAroundCurrentTime(currentTime: number): number | null {
    const searchRange = 30 // search 30 seconds around current time
    const startTime = Math.max(0, currentTime - 10)
    const endTime = currentTime + searchRange
    
    const startSegment = getSegmentForTime(startTime)
    const endSegment = getSegmentForTime(endTime)
    
    if (!startSegment || !endSegment) return null
    
    // check for missing segments in the range
    for (let i = startSegment.index; i <= endSegment.index; i++) {
      if (!loadedSegments.has(i)) {
        hlsLog(`Found gap: segment ${i} missing around current time ${currentTime.toFixed(1)}s`)
        return i
      }
    }
    
    return null
  }

  /**
   * Finds segments needed to maintain the required buffer ahead time.
   * Returns array of segment indices that need to be loaded, sorted by priority.
   */
  function findSegmentsNeededForBufferAhead(currentTime: number, targetEndTime: number): number[] {
    const startSegment = getSegmentForTime(currentTime)
    const endSegment = getSegmentForTime(targetEndTime)
    
    if (!startSegment || !endSegment) return []
    
    const neededSegments: number[] = []
    
    for (let i = startSegment.index; i <= endSegment.index && i < segments.length; i++) {
      if (!loadedSegments.has(i)) {
        neededSegments.push(i)
      }
    }
    
    return neededSegments
  }

  /**
   * Gets a human-readable reason for why a segment is being loaded.
   */
  function getLoadReason(segmentIndex: number): string {
    const currentTime = videoEl.currentTime
    const segment = segments[segmentIndex]
    
    if (!segment) return 'unknown reason'
    
    const segmentStart = segment.startTime
    const segmentEnd = segment.startTime + segment.duration
    
    // check if it's filling a gap
    if (Math.abs(currentTime - segmentStart) < 30 || Math.abs(currentTime - segmentEnd) < 30) {
      if (currentTime >= segmentStart && currentTime <= segmentEnd) {
        return 'filling gap at current position'
      } else {
        return 'filling gap near current position'
      }
    }
    
    // check if it's for buffer ahead
    if (segmentStart > currentTime) {
      return 'maintaining buffer ahead'
    }
    
    return 'buffer management'
  }

  async function startPlayback(): Promise<void> {
    if (isInitialized) {
      hlsLog('player already initialized, skipping')
      return
    }

    if (isInitializing) {
      hlsLog('player already initializing, skipping')
      return
    }

    try {
      hlsLog('starting HLS playback process')
      isInitializing = true
      
      // step 1: get master playlist URL
      const masterUrl = await masterUrlResolver()
      hlsLog('master playlist URL resolved:', masterUrl)
      
      // step 2: parse playlists and segments
      const parseResult = await parsePlaylistAndSegments(masterUrl)
      segments = parseResult.segments
      const detectedCodecs = parseResult.codecs
      
      // step 3: build mime type from detected codecs
      const mimeType = buildMimeType(detectedCodecs)
      
      // step 4: setup MediaSource
      const result = await setupMediaSource(mimeType)
      mediaSource = result.mediaSource
      sourceBuffer = result.sourceBuffer
      
      // step 5: setup event listeners
      setupVideoEventListeners()
      
      // step 6: start buffer management
      manageBuffer()
      
      isInitialized = true
      isInitializing = false
      hlsLog('HLS player initialized successfully')
      
    } catch (error) {
      console.error('failed to start HLS playback:', error)
      isInitializing = false
      throw error
    }
  }

  async function seekTo(time: number): Promise<void> {
    if (!isInitialized || !sourceBuffer || !mediaSource) {
      console.warn('cannot seek: player not initialized')
      return
    }

    hlsLog('seeking to time:', time)
    
    try {
      // step 1: increment generation ID to invalidate all pending operations
      currentGenerationId++
      hlsLog(`seek: incremented generation ID to ${currentGenerationId}`)
      
      // step 2: enter seeking state immediately
      playerState = 'seeking'
      hlsLog('entered seeking state, cancelling all operations')
      
      // step 3: cancel all in-flight operations aggressively
      abortController.abort('seek operation started')
      abortController = new AbortController() // create new controller for new operations
      
      // step 4: clear the segment queue and remove stale tasks
      const queueSizeBefore = segmentQueue.length
      segmentQueue.length = 0
      isProcessingQueue = false
      hlsLog(`cleared ${queueSizeBefore} queued segments during seek`)
      
      // step 5: wait for sourceBuffer to be ready
      if (sourceBuffer.updating) {
        hlsLog('waiting for sourceBuffer to finish updating')
        await new Promise<void>((resolve) => {
          sourceBuffer!.addEventListener('updateend', () => resolve(), { once: true })
        })
      }
      
      // step 6: unconditionally clear all buffer ranges (aggressive approach)
      hlsLog('clearing all buffer ranges unconditionally')
      if (sourceBuffer.buffered.length > 0) {
        // clear everything - most robust approach
        sourceBuffer.remove(0, Number.MAX_SAFE_INTEGER)
        
        await new Promise<void>((resolve) => {
          sourceBuffer!.addEventListener('updateend', () => resolve(), { once: true })
        })
      }
      
      // step 7: reset all state for new position
      const targetSegment = getSegmentForTime(time)
      if (!targetSegment) {
        console.error('no segment found for time:', time)
        playerState = 'idle'
        return
      }
      
      hlsLog('resetting state for new position, target segment:', targetSegment.index, `(generation ${currentGenerationId})`)
      currentSegmentIndex = targetSegment.index
      hasStartedPlayback = false
      isAppending = false
      loadedSegments.clear()
      
      // step 8: update video current time
      programmaticSeekTime = time
      videoEl.currentTime = time
      requestAnimationFrame(() => {
        programmaticSeekTime = null
      })
      
      // step 9: exit seeking state and restart buffer management
      playerState = 'idle'
      hlsLog('seek complete, starting buffer management from segment', targetSegment.index, `(generation ${currentGenerationId})`)
      
      // manually trigger buffer management to start loading new segments
      setTimeout(() => {
        if (playerState === 'idle') {
          processSegmentQueue()
        }
      }, 50)
      
    } catch (error) {
      console.error('seek operation failed:', error)
      playerState = 'error'
      
      // attempt to recover by clearing error state after delay
      setTimeout(() => {
        if (playerState === 'error') {
          hlsLog('attempting to recover from seek error')
          playerState = 'idle'
        }
      }, 1000)
    }
  }

  function setupVideoEventListeners(): void {
    // time update for buffer management
    videoEl.addEventListener('timeupdate', () => {
      if (!isAppending) {
        manageBuffer()
      }
    })

    // seeking event - trigger immediate buffer check
    videoEl.addEventListener('seeking', () => {
      const currentTime = videoEl.currentTime;
      hlsLog('video seeking event triggered for time:', currentTime)
      
      // ignore programmatic seeks to prevent infinite loops
      if (programmaticSeekTime !== null && Math.abs(currentTime - programmaticSeekTime) < 0.1) {
        hlsLog('ignoring programmatic seek event to', currentTime)
        return
      }
      
      hlsLog('user-initiated seek detected to', currentTime)
      if (!isAppending) {
        manageBuffer()
      }
    })

    // seeked event - handle seek completion
    videoEl.addEventListener('seeked', async () => {
      hlsLog('video seeked event triggered for time:', videoEl.currentTime)
      
      // check if we need to load segments for the new position
      const { needsData } = getBufferStatus()
      if (needsData) {
        hlsLog('seek completed but buffer needs data, triggering load')
        manageBuffer()
      }
    })

    // play event
    videoEl.addEventListener('play', () => {
      hlsLog('video play event triggered')
      if (!isAppending) {
        manageBuffer()
      }
    })

    // debug events
    videoEl.addEventListener('loadedmetadata', () => {
      hlsLog('video: loadedmetadata, duration:', videoEl.duration)
    })

    videoEl.addEventListener('durationchange', () => {
      hlsLog('video: durationchange, new duration:', videoEl.duration)
    })

    // error recovery
    videoEl.addEventListener('error', () => {
      const now = Date.now()
      console.error('video element error event triggered:', videoEl.error)
      
      // debounce error events to prevent cascading
      if (now - lastErrorEventTime < ERROR_EVENT_DEBOUNCE_MS) {
        hlsLog('ignoring cascading error event')
        return
      }
      lastErrorEventTime = now
      
      if (videoEl.error && playerState !== 'seeking') {
        playerState = 'error'
        scheduleErrorRecovery()
      }
    })
  }

  function cleanup(): void {
    hlsLog('cleaning up HLS player')
    isUnmounted = true
    isInitialized = false
    isInitializing = false
    
    // revoke all object URLs
    revokeUrls.forEach(url => {
      try {
        URL.revokeObjectURL(url)
      } catch (e) {
        console.warn('failed to revoke URL:', url, e)
      }
    })
    revokeUrls = []
    
    // clear cache
    segmentUrlCache.clear()
    loadedSegments.clear()
    
    // cancel any in-flight operations
    abortController.abort('cleanup')
    abortController = new AbortController()
    
    // properly close MediaSource
    if (mediaSource && mediaSource.readyState !== 'closed') {
      try {
        mediaSource.endOfStream()
      } catch (e) {
        console.warn('failed to end MediaSource stream:', e)
      }
    }
    
    // clear video src to fully disconnect MediaSource
    if (videoEl.src && videoEl.src.startsWith('blob:')) {
      hlsLog('revoking existing blob URL:', videoEl.src)
      const oldSrc = videoEl.src
      videoEl.src = ''
      videoEl.load()
      
      // revoke the blob URL to free memory
      try {
        URL.revokeObjectURL(oldSrc)
      } catch (e) {
        console.warn('failed to revoke blob URL during cleanup:', e)
      }
    }
    
    // pause the video to ensure it stops completely
    if (!videoEl.paused) {
      videoEl.pause()
    }
    
    // reset state machine
    playerState = 'idle'
    segmentQueue.length = 0
    isProcessingQueue = false
    currentGenerationId = 0
    appendLock = false
    
    // reset error recovery state
    errorRecoveryCount = 0
    lastErrorRecoveryTime = 0
    lastErrorEventTime = 0
    
    // reset state
    mediaSource = null
    sourceBuffer = null
    segments = []
    currentSegmentIndex = 0
    hasStartedPlayback = false
    isAppending = false
    totalDuration = 0
  }

  // ===== PUBLIC API =====

  return {
    async start() {
      await startPlayback()
    },
    
    async seekTo(time: number) {
      await seekTo(time)
    },
    
    async restart(targetTime: number = 0) {
      hlsLog('restarting HLS player completely, target time:', targetTime)
      
      // complete cleanup first
      cleanup()
      
      // wait a bit for cleanup to complete
      await new Promise(resolve => setTimeout(resolve, 100))
      
      // restart playback
      await startPlayback()
      
      // seek to target time if specified
      if (targetTime > 0) {
        hlsLog('seeking to target time after restart:', targetTime)
        await seekTo(targetTime)
      }
    },
    
    cleanup() {
      cleanup()
    },
    
    isInitialized() {
      return isInitialized
    }
  }
}
