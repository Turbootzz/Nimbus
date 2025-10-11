import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  User,
  Service,
  ServiceCreateRequest,
  ServiceUpdateRequest,
  ApiResponse,
} from '@/types'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

class ApiClient {
  private token: string | null = null

  constructor() {
    // Load token from localStorage on initialization (client-side only)
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('auth_token')
    }
  }

  setToken(token: string) {
    this.token = token
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', token)
    }
  }

  clearToken() {
    this.token = null
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token')
    }
  }

  getToken(): string | null {
    return this.token
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    try {
      const response = await fetch(`${API_URL}${endpoint}`, {
        ...options,
        headers,
      })

      const data = await response.json()

      if (!response.ok) {
        return {
          error: {
            message: data.message || 'An error occurred',
            code: data.code,
            details: data.details,
          },
        }
      }

      return { data }
    } catch (error) {
      return {
        error: {
          message: error instanceof Error ? error.message : 'Network error',
        },
      }
    }
  }

  // ============================================
  // Authentication
  // ============================================

  async login(credentials: LoginRequest): Promise<ApiResponse<AuthResponse>> {
    const response = await this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    })

    if (response.data?.token) {
      this.setToken(response.data.token)
    }

    return response
  }

  async register(data: RegisterRequest): Promise<ApiResponse<AuthResponse>> {
    const response = await this.request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    })

    if (response.data?.token) {
      this.setToken(response.data.token)
    }

    return response
  }

  async logout(): Promise<void> {
    this.clearToken()
    // Optional: Call backend logout endpoint if needed
    // await this.request('/auth/logout', { method: 'POST' })
  }

  async getCurrentUser(): Promise<ApiResponse<User>> {
    return this.request<User>('/auth/me')
  }

  // ============================================
  // Services
  // ============================================

  async getServices(): Promise<ApiResponse<Service[]>> {
    return this.request<Service[]>('/services')
  }

  async getService(id: string): Promise<ApiResponse<Service>> {
    return this.request<Service>(`/services/${id}`)
  }

  async createService(data: ServiceCreateRequest): Promise<ApiResponse<Service>> {
    return this.request<Service>('/services', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateService(
    id: string,
    data: ServiceUpdateRequest
  ): Promise<ApiResponse<Service>> {
    return this.request<Service>(`/services/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteService(id: string): Promise<ApiResponse<void>> {
    return this.request<void>(`/services/${id}`, {
      method: 'DELETE',
    })
  }

  // ============================================
  // Health Checks
  // ============================================

  async getServiceHealth(id: string): Promise<ApiResponse<any>> {
    return this.request(`/health/services/${id}`)
  }

  async getAllServicesHealth(): Promise<ApiResponse<any>> {
    return this.request('/health/services')
  }
}

// Export singleton instance
export const api = new ApiClient()
