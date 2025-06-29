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

  // shared response handler
  private async handleResponse<T>(response: Response): Promise<T> {
    // check if response is JSON
    const contentType = response.headers.get('content-type')
    if (!contentType || !contentType.includes('application/json')) {
      const text = await response.text()
      throw new Error(`server returned non-JSON response: ${text.substring(0, 200)}`)
    }

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || `request failed with status ${response.status}`)
    }

    return data as T
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

    return this.handleResponse<T>(response)
  }

  // post method that accepts guest token
  async postWithGuestToken<T>(endpoint: string, body: unknown, guestToken?: string): Promise<T> {
    await this.initializeConfig()
    
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    
    if (guestToken) {
      headers['X-Guest-Token'] = guestToken
    } else {
      Object.assign(headers, this.getAuthHeaders())
    }
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    })

    return this.handleResponse<T>(response)
  }

  async get<T>(endpoint: string): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      headers: {
        ...this.getAuthHeaders(),
      },
    })

    return this.handleResponse<T>(response)
  }

  // public get method without auth headers for guest requests
  async publicGet<T>(endpoint: string): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return this.handleResponse<T>(response)
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

    return this.handleResponse<T>(response)
  }

  async delete<T>(endpoint: string): Promise<T> {
    await this.initializeConfig()
    
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: 'DELETE',
      headers: {
        ...this.getAuthHeaders(),
      },
    })

    return this.handleResponse<T>(response)
  }
}

export const apiClient = new ApiClient()
