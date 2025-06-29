/**
 * SeekController - Handles seeking logic and segment selection
 */

import type { SegmentInfo } from '../hlsParser'

export interface SeekResult {
  targetSegmentIndex: number
  segmentStartTime: number
  preloadSegments: number[]
  shouldClearBuffer: boolean
}

export class SeekController {
  private segments: SegmentInfo[] = []
  
  setSegments(segments: SegmentInfo[]): void {
    this.segments = segments
  }

  /**
   * Strategy: Find closest segment for seek time
   * This can be easily replaced with other strategies like keyframe-based seeking
   */
  seekUsingNearestSegment(targetTime: number, preloadCount: number = 3): SeekResult {
    if (this.segments.length === 0) {
      throw new Error('No segments available for seeking')
    }

    // Find target segment using accumulated time
    let targetSegmentIndex = 0
    let accumulatedTime = 0
    
    for (let i = 0; i < this.segments.length; i++) {
      if (targetTime <= accumulatedTime + this.segments[i].duration) {
        targetSegmentIndex = i
        break
      }
      accumulatedTime += this.segments[i].duration
      
      // If we've gone past all segments, use the last one
      if (i === this.segments.length - 1) {
        targetSegmentIndex = i
      }
    }

    // Calculate actual start time of the target segment
    let segmentStartTime = 0
    for (let i = 0; i < targetSegmentIndex; i++) {
      segmentStartTime += this.segments[i].duration
    }

    // Calculate preload segments
    const preloadSegments: number[] = []
    const endIndex = Math.min(targetSegmentIndex + preloadCount, this.segments.length)
    
    for (let i = targetSegmentIndex; i < endIndex; i++) {
      preloadSegments.push(i)
    }

    // Determine if we should clear buffer (if seeking far from current position)
    // For now, always clear buffer on seek for simplicity
    const shouldClearBuffer = true

    console.log(`🎯 Seek strategy result:`, {
      targetTime,
      targetSegmentIndex,
      segmentStartTime,
      preloadSegments,
      shouldClearBuffer
    })

    return {
      targetSegmentIndex,
      segmentStartTime,
      preloadSegments,
      shouldClearBuffer
    }
  }

  /**
   * Alternative strategy: Seek using keyframes (placeholder for future implementation)
   */
  seekUsingNearestKeyframe(targetTime: number, preloadCount: number = 3): SeekResult {
    // For now, fallback to nearest segment strategy
    // In the future, this could analyze segment metadata to find keyframes
    console.log('🔑 Using keyframe-based seeking (fallback to nearest segment)')
    return this.seekUsingNearestSegment(targetTime, preloadCount)
  }

  /**
   * Get segment timing info
   */
  getSegmentTiming(segmentIndex: number): { start: number, end: number, duration: number } | null {
    if (segmentIndex < 0 || segmentIndex >= this.segments.length) {
      return null
    }

    let start = 0
    for (let i = 0; i < segmentIndex; i++) {
      start += this.segments[i].duration
    }

    const duration = this.segments[segmentIndex].duration
    const end = start + duration

    return { start, end, duration }
  }

  /**
   * Check if a time is within a buffered range
   */
  isTimeInBufferedRange(time: number, bufferedRanges: { start: number, end: number }[]): boolean {
    return bufferedRanges.some(range => time >= range.start && time <= range.end)
  }

  /**
   * Find the closest buffered range to a target time
   */
  findClosestBufferedRange(time: number, bufferedRanges: { start: number, end: number }[]): {
    range: { start: number, end: number } | null
    distance: number
    adjustedTime: number | null
  } {
    if (bufferedRanges.length === 0) {
      return { range: null, distance: Infinity, adjustedTime: null }
    }

    let closestRange: { start: number, end: number } | null = null
    let minDistance = Infinity
    let adjustedTime: number | null = null

    for (const range of bufferedRanges) {
      let distance: number
      let suggestedTime: number

      if (time < range.start) {
        // Time is before this range
        distance = range.start - time
        suggestedTime = range.start
      } else if (time > range.end) {
        // Time is after this range
        distance = time - range.end
        suggestedTime = range.end
      } else {
        // Time is within this range
        distance = 0
        suggestedTime = time
      }

      if (distance < minDistance) {
        minDistance = distance
        closestRange = range
        adjustedTime = suggestedTime
      }
    }

    return { range: closestRange, distance: minDistance, adjustedTime }
  }
}
