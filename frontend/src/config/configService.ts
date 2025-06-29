import { config } from './environment'

// local configuration that's immediately available
export const localConfig = {
  apiUrl: config.apiUrl,
  wsUrl: config.wsUrl,
  mode: config.mode
}

// remote configuration interface
interface RemoteConfig {
  apiUrl: string
  wsUrl: string
  features?: {
    guestAccess?: boolean
    videoUpload?: boolean
    emailInvites?: boolean
  }
}

// cache for remote config
let remoteConfigCache: RemoteConfig | null = null
let configPromise: Promise<RemoteConfig> | null = null

// get configuration from backend (for saas mode)
export async function getConfig(): Promise<RemoteConfig> {
  // return cached config if available
  if (remoteConfigCache) {
    return remoteConfigCache
  }

  // return existing promise if already fetching
  if (configPromise) {
    return configPromise
  }

  // only fetch remote config in saas mode
  if (config.mode !== 'production-saas') {
    return localConfig
  }

  configPromise = fetchRemoteConfig()
  
  try {
    remoteConfigCache = await configPromise
    return remoteConfigCache
  } catch (error) {
    console.warn('failed to fetch remote config, using local:', error)
    configPromise = null
    return localConfig
  }
}

async function fetchRemoteConfig(): Promise<RemoteConfig> {
  const response = await fetch(`${config.apiUrl}/api/v1/config`)
  
  if (!response.ok) {
    throw new Error(`config fetch failed: ${response.status}`)
  }
  
  const data = await response.json()
  
  return {
    apiUrl: data.api_url || config.apiUrl,
    wsUrl: data.ws_url || config.wsUrl,
    features: data.features || {}
  }
}

// invalidate cache (useful for development)
export function invalidateConfig() {
  remoteConfigCache = null
  configPromise = null
}
