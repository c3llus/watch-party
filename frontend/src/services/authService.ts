import { apiClient } from './apiClient'

interface LoginRequest {
  email: string
  password: string
}

interface RegisterRequest {
  email: string
  password: string
}

interface UserProfile {
  id: string
  email: string
  role: string
  created_at: string
}

interface LoginResponse {
  access_token: string
  refresh_token: string
  user: UserProfile
}

interface RegisterResponse {
  message: string
  user: UserProfile
}

export const authService = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>('/auth/login', credentials)
  },

  async register(userData: RegisterRequest): Promise<RegisterResponse> {
    return apiClient.post<RegisterResponse>('/users/register', userData)
  },

  logout(): void {
    // clear session data from localStorage (like clearing Redis cache)
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user')
  },

  getCurrentUser(): UserProfile | null {
    const userStr = localStorage.getItem('user')
    return userStr ? JSON.parse(userStr) : null
  },

  isAuthenticated(): boolean {
    return !!localStorage.getItem('token')
  },

  isAdmin(): boolean {
    const user = this.getCurrentUser()
    return user?.role === 'admin'
  }
}
