import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  User,
  Service,
  ServiceCreateRequest,
  ServiceUpdateRequest,
  ApiResponse,
  HealthCheck,
} from '@/types'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

/**
 * ApiClient - Secure API client using httpOnly cookies
 *
 * SECURITY NOTE: This client uses httpOnly cookies for authentication instead of
 * storing JWT tokens in localStorage/sessionStorage, which protects against XSS attacks.
 *
 * - All requests include credentials: 'include' to send httpOnly cookies
 * - Backend sets auth_token cookie with httpOnly, secure, and sameSite flags
 * - No token management in JavaScript - cookies are handled automatically by browser
 */
class ApiClient {
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }

    try {
      const response = await fetch(`${API_URL}${endpoint}`, {
        ...options,
        headers,
        credentials: 'include', // Always send httpOnly cookies with requests
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
    // Backend will set httpOnly cookie in response
    // No need to store token - it's handled automatically
    return this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    })
  }

  async register(data: RegisterRequest): Promise<ApiResponse<AuthResponse>> {
    // Backend will set httpOnly cookie in response
    // No need to store token - it's handled automatically
    return this.request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async logout(): Promise<void> {
    // Call backend to clear httpOnly cookie
    await this.request('/auth/logout', { method: 'POST' })
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

  async updateService(id: string, data: ServiceUpdateRequest): Promise<ApiResponse<Service>> {
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

  async getServiceHealth(id: string): Promise<ApiResponse<HealthCheck>> {
    return this.request<HealthCheck>(`/health/services/${id}`)
  }

  async getAllServicesHealth(): Promise<ApiResponse<HealthCheck[]>> {
    return this.request<HealthCheck[]>('/health/services')
  }
}

// Export singleton instance
export const api = new ApiClient()
