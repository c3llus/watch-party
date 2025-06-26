import { localConfig, getConfig } from '../config/configService'

class ApiClient {
  private baseUrl = `${localConfig.apiUrl}/api/v1`
  private configInitialized = false
  
  // initialize with remote config if available
  private async initializeConfig() {
    if (this.configInitialized) return
    
    try {
      const config = await getConfig()
      this.baseUrl = `${config.apiUrl}/api/v1`
      this.configInitialized = true
    } catch {
      console.warn('failed to load remote config, using local config')
      this.configInitialized = true
    }
  }
  
  private getAuthHeaders(): Record<string, string> {
    const token = localStorage.getItem('token')
    return token ? { Authorization: `Bearer ${token}` } : {}
  }

  async post<T>(endpoint: string, body: unknown): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
      },
      body: JSON.stringify(body),
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || `request failed with status ${response.status}`)
    }

    return data as T
  }

  async get<T>(endpoint: string): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      headers: {
        ...this.getAuthHeaders(),
      },
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || `request failed with status ${response.status}`)
    }

    return data as T
  }

  async put<T>(endpoint: string, body: unknown): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
      },
      body: JSON.stringify(body),
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || `request failed with status ${response.status}`)
    }

    return data as T
  }

  async delete<T>(endpoint: string): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: 'DELETE',
      headers: {
        ...this.getAuthHeaders(),
      },
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || `request failed with status ${response.status}`)
    }

    return data as T
  }
}

export const apiClient = new ApiClient()
