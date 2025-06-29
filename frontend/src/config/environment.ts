// environment configuration for different deployment modes
interface EnvironmentConfig {
  apiUrl: string
  wsUrl: string
  mode: 'development' | 'production-saas' | 'production-selfhost'
}

// get configuration based on environment variables
function getEnvironmentConfig(): EnvironmentConfig {
  // vite exposes env vars prefixed with VITE_
  const apiUrl = import.meta.env.VITE_API_URL
  const wsUrl = import.meta.env.VITE_WS_URL
  const mode = import.meta.env.VITE_MODE || import.meta.env.MODE || 'development'

  // if environment variables are set, use them directly
  if (apiUrl && wsUrl) {
    return {
      apiUrl: apiUrl.replace(/\/$/, ''), // remove trailing slash
      wsUrl: wsUrl.replace(/\/$/, ''),
      mode: mode as EnvironmentConfig['mode']
    }
  }

  // fallback configuration based on mode
  switch (mode) {
    case 'production-saas':
      // saas mode - uses configurable hosted service URLs
      return {
        apiUrl: apiUrl || 'https://api.example.com',
        wsUrl: wsUrl || 'wss://sync.example.com',
        mode: 'production-saas'
      }

    case 'production-selfhost': {
      // self-hosted mode - auto-detect from current hostname
      const hostname = window.location.hostname
      const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      
      return {
        apiUrl: `${protocol}//${hostname}:8080`,
        wsUrl: `${wsProtocol}//${hostname}:8081`,
        mode: 'production-selfhost'
      }
    }

    case 'development':
    default:
      return {
        apiUrl: 'http://localhost:8080',
        wsUrl: 'ws://localhost:8081',
        mode: 'development'
      }
  }
}

export const config = getEnvironmentConfig()

// helper to determine deployment type
export const isDevelopment = config.mode === 'development'
export const isSaasMode = config.mode === 'production-saas'
export const isSelfHosted = config.mode === 'production-selfhost'