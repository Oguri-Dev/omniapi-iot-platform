import api from './api'

export interface LoginCredentials {
  username: string
  password: string
}

export interface User {
  id: string
  username: string
  email: string
  full_name: string
  role: string
  avatar?: string
}

export interface LoginResponse {
  success: boolean
  message: string
  data: {
    token: string
    user: User
  }
}

class AuthService {
  async login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await api.post<LoginResponse>('/api/auth/login', credentials)

    if (response.data.success && response.data.data.token) {
      localStorage.setItem('token', response.data.data.token)
      localStorage.setItem('user', JSON.stringify(response.data.data.user))
    }

    return response.data
  }

  logout(): void {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  getCurrentUser(): User | null {
    const userStr = localStorage.getItem('user')
    if (userStr) {
      try {
        return JSON.parse(userStr)
      } catch {
        return null
      }
    }
    return null
  }

  getToken(): string | null {
    return localStorage.getItem('token')
  }

  isAuthenticated(): boolean {
    return !!this.getToken()
  }
}

export default new AuthService()
