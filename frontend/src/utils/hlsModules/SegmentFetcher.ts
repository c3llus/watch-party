/**
 * SegmentFetcher - Handles fetching .ts segments with caching
 */

export interface SegmentCache {
  [key: string]: ArrayBuffer
}

export class SegmentFetcher {
  private cache: SegmentCache = {}
  private readonly maxCacheSize = 50 // Maximum segments to cache
  private cacheKeys: string[] = [] // LRU tracking

  async fetchSegment(url: string, segmentPath: string): Promise<ArrayBuffer> {
    // Check cache first
    if (this.cache[segmentPath]) {
      console.log(`📦 Segment ${segmentPath} served from cache`)
      return this.cache[segmentPath]
    }

    console.log(`🔄 Fetching segment ${segmentPath}`)
    
    try {
      const response = await fetch(url)
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const arrayBuffer = await response.arrayBuffer()
      console.log(`✅ Segment ${segmentPath} fetched, size: ${arrayBuffer.byteLength} bytes`)
      
      // Cache the segment
      this.cacheSegment(segmentPath, arrayBuffer)
      
      return arrayBuffer
    } catch (error) {
      console.error(`❌ Failed to fetch segment ${segmentPath}:`, error)
      throw error
    }
  }

  private cacheSegment(segmentPath: string, data: ArrayBuffer): void {
    // Remove from cache if already exists (for LRU update)
    if (this.cache[segmentPath]) {
      const index = this.cacheKeys.indexOf(segmentPath)
      if (index > -1) {
        this.cacheKeys.splice(index, 1)
      }
    }

    // Add to cache
    this.cache[segmentPath] = data
    this.cacheKeys.push(segmentPath)

    // Maintain cache size limit (LRU eviction)
    while (this.cacheKeys.length > this.maxCacheSize) {
      const oldestKey = this.cacheKeys.shift()!
      delete this.cache[oldestKey]
      console.log(`🗑️ Evicted segment ${oldestKey} from cache`)
    }
  }

  clearCache(): void {
    this.cache = {}
    this.cacheKeys = []
    console.log('🧹 Segment cache cleared')
  }

  getCacheSize(): number {
    return this.cacheKeys.length
  }

  getCacheInfo(): { size: number, keys: string[] } {
    return {
      size: this.cacheKeys.length,
      keys: [...this.cacheKeys]
    }
  }
}
