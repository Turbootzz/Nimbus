import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  User,
  Service,
  ServiceCreateRequest,
  ServiceUpdateRequest,
  ServiceReorderRequest,
  Group,
  GroupCreateRequest,
  GroupUpdateRequest,
  GroupReorderRequest,
  ApiResponse,
  HealthCheck,
  UserPreferences,
  PreferencesUpdateRequest,
  PaginatedUsersResponse,
  UserFilterParams,
  OAuthProvider,
  OAuthProviderStatus,
  Webhook,
  WebhookCreateRequest,
  WebhookUpdateRequest,
  WebhookLog,
  WebhookTestResult,
  SetupStatusResponse,
  RegistrationStatusResponse,
  SystemSetting,
  SystemSettingsResponse,
  UpdateSettingRequest,
  ForgotPasswordRequest,
  ResetPasswordRequest,
  ChangePasswordRequest,
  SMTPStatusResponse,
  UpdateSMTPSettingsRequest,
} from '@/types'
import { getApiUrl as getClientApiUrl } from '@/lib/utils/api-url'

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

  // Client-side: use shared utility
  return getClientApiUrl()
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

      // Handle 204 No Content responses (empty body)
      if (response.status === 204) {
        return { data: undefined as T }
      }

      // Parse response as text first to handle non-JSON responses gracefully
      const text = await response.text()

      // Handle empty successful responses
      if (!text && response.ok) {
        return { data: undefined as T }
      }

      let data
      try {
        data = JSON.parse(text)
      } catch {
        // Check for common HTML error responses (reverse proxy errors, 404 pages, etc.)
        if (text.includes('<!DOCTYPE') || text.includes('<html')) {
          console.error('[API Client] Received HTML instead of JSON. API URL may be misconfigured.')
          return {
            error: {
              message:
                'Cannot reach API server. If using Docker, ensure NEXT_PUBLIC_API_URL is set to your server IP (e.g., http://192.168.1.100:8080), not "http://backend:8080".',
            },
          }
        }
        console.error('[API Client] Invalid JSON response:', text.substring(0, 200))
        return {
          error: {
            message: 'API returned an invalid response. Check server logs for details.',
          },
        }
      }

      if (!response.ok) {
        // Handle 401 Unauthorized - token is invalid or user doesn't exist
        if (response.status === 401 && typeof window !== 'undefined') {
          // Don't redirect for preferences endpoint - it's okay if user isn't logged in yet
          const isPreferencesEndpoint = endpoint.includes('/users/me/preferences')

          // Redirect to login unless already on login/register page or accessing preferences
          if (
            !isPreferencesEndpoint &&
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

  async forgotPassword(data: ForgotPasswordRequest): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async resetPassword(data: ResetPasswordRequest): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/auth/reset-password', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async changePassword(data: ChangePasswordRequest): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/auth/change-password', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteAccount(): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/auth/me', {
      method: 'DELETE',
    })
  }

  async uploadAvatar(formData: FormData): Promise<ApiResponse<User>> {
    const apiUrl = getApiUrl()
    if (!apiUrl) {
      return {
        error: { message: 'API URL not configured' },
      }
    }

    try {
      const response = await fetch(`${apiUrl}/users/me/avatar`, {
        method: 'PUT',
        credentials: 'include',
        body: formData,
      })

      const text = await response.text()
      let data
      try {
        data = JSON.parse(text)
      } catch {
        if (text.includes('<!DOCTYPE') || text.includes('<html')) {
          return {
            error: {
              message: 'Cannot reach API server. Check NEXT_PUBLIC_API_URL configuration.',
            },
          }
        }
        return { error: { message: 'API returned an invalid response' } }
      }

      if (!response.ok) {
        return {
          error: { message: data.error || data.message || 'Failed to upload avatar' },
        }
      }

      return {
        data: data.user,
      }
    } catch (error) {
      return {
        error: { message: error instanceof Error ? error.message : 'Failed to upload avatar' },
      }
    }
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

  async reorderServices(data: ServiceReorderRequest): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/services/reorder', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async fetchServiceFavicon(
    url: string
  ): Promise<ApiResponse<{ icon_image_path: string; message: string }>> {
    return this.request<{ icon_image_path: string; message: string }>(
      `/services/favicon?url=${encodeURIComponent(url)}`
    )
  }

  async uploadServiceIcon(
    file: File
  ): Promise<ApiResponse<{ icon_image_path: string; message: string }>> {
    const apiUrl = getApiUrl()
    if (!apiUrl) {
      return {
        error: {
          message: 'API URL not configured',
        },
      }
    }

    const formData = new FormData()
    formData.append('icon', file)

    try {
      const response = await fetch(`${apiUrl}/uploads/service-icon`, {
        method: 'POST',
        body: formData,
        credentials: 'include', // Send httpOnly cookies
      })

      const text = await response.text()
      let data
      try {
        data = JSON.parse(text)
      } catch {
        if (text.includes('<!DOCTYPE') || text.includes('<html')) {
          return {
            error: {
              message: 'Cannot reach API server. Check NEXT_PUBLIC_API_URL configuration.',
            },
          }
        }
        return { error: { message: 'API returned an invalid response' } }
      }

      if (!response.ok) {
        return {
          error: {
            message: data.error || data.message || 'Upload failed',
          },
        }
      }

      return { data }
    } catch (error) {
      return {
        error: {
          message: error instanceof Error ? error.message : 'Upload failed',
        },
      }
    }
  }

  // ============================================
  // Groups
  // ============================================

  async getGroups(): Promise<ApiResponse<Group[]>> {
    return this.request<Group[]>('/groups')
  }

  async getGroup(id: string): Promise<ApiResponse<Group>> {
    return this.request<Group>(`/groups/${id}`)
  }

  async createGroup(data: GroupCreateRequest): Promise<ApiResponse<Group>> {
    return this.request<Group>('/groups', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateGroup(id: string, data: GroupUpdateRequest): Promise<ApiResponse<Group>> {
    return this.request<Group>(`/groups/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteGroup(
    id: string,
    deleteServices: boolean = false
  ): Promise<ApiResponse<{ message: string }>> {
    const query = deleteServices ? '?delete_services=true' : ''
    return this.request<{ message: string }>(`/groups/${id}${query}`, {
      method: 'DELETE',
    })
  }

  async reorderGroups(data: GroupReorderRequest): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/groups/reorder', {
      method: 'PUT',
      body: JSON.stringify(data),
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

  // OAuth methods
  /**
   * Get OAuth provider status (which providers are configured)
   */
  async getOAuthProviders(): Promise<ApiResponse<{ providers: OAuthProviderStatus[] }>> {
    return this.request<{ providers: OAuthProviderStatus[] }>('/auth/oauth/providers')
  }

  /**
   * Initiate OAuth login flow - redirects to provider
   */
  initiateOAuth(provider: OAuthProvider, redirectTo?: string, rememberMe?: boolean): void {
    const apiUrl = getApiUrl()
    if (!apiUrl) {
      console.error('API URL not configured for OAuth')
      return
    }

    const params = new URLSearchParams()
    if (redirectTo) params.set('redirect', redirectTo)
    if (rememberMe) params.set('remember_me', 'true')
    const queryString = params.toString()
    window.location.href = `${apiUrl}/auth/oauth/${provider}${queryString ? '?' + queryString : ''}`
  }

  /**
   * Unlink an OAuth provider from the current user's account
   */
  async unlinkOAuthProvider(
    provider: OAuthProvider
  ): Promise<ApiResponse<{ message: string; user: User }>> {
    return this.request<{ message: string; user: User }>(`/auth/oauth/unlink/${provider}`, {
      method: 'DELETE',
    })
  }

  // ============================================
  // Webhooks
  // ============================================

  async getWebhooks(): Promise<ApiResponse<Webhook[]>> {
    return this.request<Webhook[]>('/webhooks')
  }

  async getWebhook(id: string): Promise<ApiResponse<Webhook>> {
    return this.request<Webhook>(`/webhooks/${id}`)
  }

  async createWebhook(data: WebhookCreateRequest): Promise<ApiResponse<Webhook>> {
    return this.request<Webhook>('/webhooks', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateWebhook(id: string, data: WebhookUpdateRequest): Promise<ApiResponse<Webhook>> {
    return this.request<Webhook>(`/webhooks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteWebhook(id: string): Promise<ApiResponse<void>> {
    return this.request<void>(`/webhooks/${id}`, {
      method: 'DELETE',
    })
  }

  async testWebhook(id: string): Promise<ApiResponse<WebhookTestResult>> {
    return this.request<WebhookTestResult>(`/webhooks/${id}/test`, {
      method: 'POST',
    })
  }

  async getWebhookLogs(id: string, limit?: number): Promise<ApiResponse<WebhookLog[]>> {
    const query = limit ? `?limit=${limit}` : ''
    return this.request<WebhookLog[]>(`/webhooks/${id}/logs${query}`)
  }

  // ============================================
  // Setup (First-time installation)
  // ============================================

  async getSetupStatus(): Promise<ApiResponse<SetupStatusResponse>> {
    return this.request<SetupStatusResponse>('/setup/status')
  }

  async getRegistrationStatus(): Promise<ApiResponse<RegistrationStatusResponse>> {
    return this.request<RegistrationStatusResponse>('/setup/registration-status')
  }

  async createInitialAdmin(data: RegisterRequest): Promise<ApiResponse<AuthResponse>> {
    return this.request<AuthResponse>('/setup/admin', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // ============================================
  // System Settings (Admin only)
  // ============================================

  async getSystemSettings(): Promise<ApiResponse<SystemSettingsResponse>> {
    return this.request<SystemSettingsResponse>('/admin/settings')
  }

  async getSystemSetting(key: string): Promise<ApiResponse<SystemSetting>> {
    return this.request<SystemSetting>(`/admin/settings/${key}`)
  }

  async updateSystemSetting(
    key: string,
    data: UpdateSettingRequest
  ): Promise<ApiResponse<SystemSetting>> {
    return this.request<SystemSetting>(`/admin/settings/${key}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async updateSMTPSettings(
    data: UpdateSMTPSettingsRequest
  ): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>('/admin/settings/smtp', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async getSMTPStatus(): Promise<ApiResponse<SMTPStatusResponse>> {
    return this.request<SMTPStatusResponse>('/admin/settings/smtp/status')
  }

  async testSMTPConnection(
    data?: UpdateSMTPSettingsRequest
  ): Promise<ApiResponse<{ success: boolean; message: string }>> {
    return this.request<{ success: boolean; message: string }>('/admin/settings/smtp/test', {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    })
  }
}

// Export singleton instance
export const api = new ApiClient()
