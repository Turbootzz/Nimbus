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
  UserPreferences,
  PreferencesUpdateRequest,
  PaginatedUsersResponse,
  UserFilterParams,
} from '@/types'

const getApiUrl = (): string | undefined => {
  const defaultPort = '8080'

  // Server-side: use internal Docker network (faster)
  if (typeof window === 'undefined') {
    return (
      process.env.INTERNAL_API_URL ||
      process.env.NEXT_PUBLIC_API_URL ||
      `http://localhost:${defaultPort}/api/v1`
    )
  }

  // Client-side: determine API URL at runtime
  // Priority: 1) Full URL env var, 2) Runtime detection with configurable port, 3) localhost fallback
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL
  }

  // Runtime: Use same host as frontend but with configurable backend port
  if (typeof window !== 'undefined' && window.location) {
    const protocol = window.location.protocol // http: or https:
    const hostname = window.location.hostname // e.g., 192.168.1.100 or localhost
    const backendPort = process.env.NEXT_PUBLIC_API_PORT || defaultPort
    return `${protocol}//${hostname}:${backendPort}/api/v1`
  }

  return `http://localhost:${defaultPort}/api/v1`
}

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
    // Validate API URL at request time (not import time)
    const apiUrl = getApiUrl()
    if (!apiUrl) {
      const errorMsg =
        'API URL not configured. Please set NEXT_PUBLIC_API_URL environment variable.'
      console.error('[API Client]', errorMsg)
      return {
        error: {
          message: errorMsg,
        },
      }
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }

    try {
      const response = await fetch(`${apiUrl}${endpoint}`, {
        ...options,
        headers,
        credentials: 'include', // Always send httpOnly cookies with requests
      })

      const data = await response.json()

      if (!response.ok) {
        // Handle 401 Unauthorized - token is invalid or user doesn't exist
        if (response.status === 401 && typeof window !== 'undefined') {
          // Redirect to login unless already on login/register page
          if (
            !window.location.pathname.startsWith('/login') &&
            !window.location.pathname.startsWith('/register')
          ) {
            window.location.href = '/login'
          }
        }

        return {
          error: {
            // Backend returns {error: "message"} or {message: "message"}
            message: data.error || data.message || 'An error occurred',
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

  // ============================================
  // User Preferences
  // ============================================

  async getPreferences(): Promise<ApiResponse<UserPreferences>> {
    return this.request<UserPreferences>('/users/me/preferences')
  }

  async updatePreferences(data: PreferencesUpdateRequest): Promise<ApiResponse<UserPreferences>> {
    return this.request<UserPreferences>('/users/me/preferences', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  // ============================================
  // Admin User Management
  // ============================================

  async getAllUsers(params?: UserFilterParams): Promise<ApiResponse<PaginatedUsersResponse>> {
    const query = new URLSearchParams()

    if (params?.search) query.append('search', params.search)
    if (params?.role) query.append('role', params.role)
    if (params?.page) query.append('page', params.page.toString())
    if (params?.limit) query.append('limit', params.limit.toString())

    const queryString = query.toString()
    const url = queryString ? `/admin/users?${queryString}` : '/admin/users'

    return this.request<PaginatedUsersResponse>(url)
  }

  async getUserStats(): Promise<ApiResponse<{ total: number; admins: number; users: number }>> {
    return this.request<{ total: number; admins: number; users: number }>('/admin/users/stats')
  }

  async updateUserRole(userId: string, role: 'admin' | 'user'): Promise<ApiResponse<User>> {
    return this.request<User>(`/admin/users/${userId}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role }),
    })
  }

  async deleteUser(userId: string): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>(`/admin/users/${userId}`, {
      method: 'DELETE',
    })
  }
}

// Export singleton instance
export const api = new ApiClient()
