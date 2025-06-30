// src/utils/hlsParser.ts

export interface MasterPlaylistVariant {
  bandwidth: number;
  resolution?: string;
  codecs?: string;
  url: string;
}

export interface MasterPlaylist {
  variants: MasterPlaylistVariant[];
}

export interface MediaSegment {
  duration: number;
  url: string;
  title?: string;
}

export interface MediaPlaylist {
  segments: MediaSegment[];
  targetDuration: number;
}

// Parse master playlist (.m3u8)
export function parseMasterPlaylist(text: string, baseUrl: string): MasterPlaylist {
  const lines = text.split(/\r?\n/);
  const variants: MasterPlaylistVariant[] = [];
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line.startsWith('#EXT-X-STREAM-INF')) {
      // extract bandwidth
      const bwMatch = line.match(/BANDWIDTH=(\d+)/);
      const bandwidth = bwMatch ? parseInt(bwMatch[1], 10) : 0;
      
      // extract resolution
      const resMatch = line.match(/RESOLUTION=(\d+x\d+)/);
      const resolution = resMatch ? resMatch[1] : undefined;
      
      // extract codecs - handle both quoted and unquoted formats
      const codecMatch = line.match(/CODECS=(?:"([^"]+)"|([^,\s]+))/);
      const codecs = codecMatch ? (codecMatch[1] || codecMatch[2]) : undefined;
      
      // get the URL from next non-comment line
      const url = lines[i + 1] && !lines[i + 1].startsWith('#') ? 
        new URL(lines[i + 1], baseUrl).toString() : '';
      
      if (url) {
        variants.push({ bandwidth, resolution, codecs, url });
      }
    }
  }
  
  return { variants };
}

// Parse media playlist (.m3u8) segments
export function parseMediaPlaylist(text: string, baseUrl: string): MediaPlaylist {
  const lines = text.split(/\r?\n/);
  const segments: MediaSegment[] = [];
  let targetDuration = 0;
  let duration = 0;
  let title = undefined;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line.startsWith('#EXT-X-TARGETDURATION')) {
      const match = line.match(/#EXT-X-TARGETDURATION:(\d+)/);
      if (match) targetDuration = parseInt(match[1], 10);
    }
    if (line.startsWith('#EXTINF')) {
      const match = line.match(/#EXTINF:([\d.]+)(?:,(.*))?/);
      if (match) {
        duration = parseFloat(match[1]);
        title = match[2] || undefined;
      }
      const url = lines[i + 1] && !lines[i + 1].startsWith('#') ? new URL(lines[i + 1], baseUrl).toString() : '';
      if (url) {
        segments.push({ duration, url, title });
      }
    }
  }
  return { segments, targetDuration };
}

import { apiClient } from '../services/apiClient'

// Fetch with auth headers and handle signed URLs
export async function fetchWithAuth(url: string, token?: string): Promise<Response> {
  // Check if this is a signed GCS URL - these don't need credentials
  if (url.includes('storage.googleapis.com') && url.includes('X-Goog-Signature')) {
    console.log('fetchWithAuth: detected signed GCS URL, skipping credentials for:', url.substring(0, 100) + '...');
    return fetch(url); // No credentials needed for signed URLs
  }
  
  console.log('fetchWithAuth: using credentials for:', url.substring(0, 100) + '...');
  return fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
  });
}

// get signed URL for a file path from backend
export async function getSignedUrl(movieId: string, filePath: string, guestToken?: string): Promise<string> {
  try {
    const data = await apiClient.postWithGuestToken<{
      movie_id: string;
      file_urls: Record<string, string>;
      expires_at: string;
      cdn_info: {
        cache_duration: string;
        cacheable: boolean;
      };
    }>(`/videos/${movieId}/urls`, { 
      files: [filePath] 
    }, guestToken);
    const result = data.file_urls[filePath];
    return result;
  } catch (error) {
    console.error('getSignedUrl error:', error)
    throw error;
  }
}

export async function getSignedUrls(movieId: string, filePath: string, guestToken?: string): Promise<{
  movie_id: string;
  file_urls: Record<string, string>;
  expires_at: string;
  cdn_info: {
    cache_duration: string;
    cacheable: boolean;
  };
}> {
  try {
    const data = await apiClient.postWithGuestToken<{
      movie_id: string;
      file_urls: Record<string, string>;
      expires_at: string;
      cdn_info: {
        cache_duration: string;
        cacheable: boolean;
      };
    }>(`/videos/${movieId}/urls`, { 
      files: [filePath] 
    }, guestToken);
    return data;
  } catch (error) {
    console.error('getSignedUrls error:', error)
    throw error;
  }
}

// get and parse master playlist  
export async function getMasterPlaylist(url: string, guestToken?: string): Promise<MasterPlaylist> {
  const res = await fetchWithAuth(url, guestToken);
  if (!res.ok) throw new Error('failed to fetch master playlist');
  const text = await res.text();
  return parseMasterPlaylist(text, url);
}

// get and parse media playlist
export async function getMediaPlaylist(url: string, guestToken?: string): Promise<MediaPlaylist> {
  const res = await fetchWithAuth(url, guestToken);
  if (!res.ok) throw new Error('failed to fetch media playlist');
  const text = await res.text();
  return parseMediaPlaylist(text, url);
}

// Get segment as blob URL
export async function getSegmentBlobUrl(url: string, token?: string): Promise<string> {
  const res = await fetchWithAuth(url, token);
  if (!res.ok) throw new Error('Failed to fetch segment');
  const blob = await res.blob();
  return URL.createObjectURL(blob);
}

// Segment info returned by backend seek API
export interface SegmentInfo {
  index: number;
  filename: string;
  duration: number;
  start_time: number;
}

// Response from backend seek API
export interface SeekResponse {
  movie_id: string;
  target_time: number;
  target_segment_index: number;
  segment_start_time: number;
  total_duration: number;
  quality: string;
  file_urls: Record<string, string>;
  segments: SegmentInfo[];
  expires_at: string;
}

// get segments for a specific time position (for seeking)
export async function getSegmentsForTime(
  movieId: string, 
  time: number, 
  quality?: string, 
  preloadCount?: number,
  guestToken?: string
): Promise<SeekResponse> {
  try {
    const data = await apiClient.postWithGuestToken<SeekResponse>(
      `/videos/${movieId}/seek`, 
      { 
        time,
        quality: quality || undefined,
        preload_count: preloadCount || 3
      }, 
      guestToken
    );
    return data;
  } catch (error) {
    console.error('getSegmentsForTime error:', error)
    throw error;
  }
}
