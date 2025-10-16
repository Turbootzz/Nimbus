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

// Validate API_URL is configured properly
const getApiUrl = () => {
  const url = process.env.NEXT_PUBLIC_API_URL

  // In production, API_URL must be explicitly set
  if (process.env.NODE_ENV === 'production' && !url) {
    console.error('[API Client] NEXT_PUBLIC_API_URL not configured in production')
    throw new Error('API URL not configured. Please set NEXT_PUBLIC_API_URL environment variable.')
  }

  // In development, fallback to localhost
  return url || 'http://localhost:8080/api/v1'
}

const API_URL = getApiUrl()

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
