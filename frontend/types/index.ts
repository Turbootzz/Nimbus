// User types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user'
  last_activity_at?: string
  created_at: string
  updated_at?: string
}

// Auth types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  name: string
  email: string
  password: string
}

export interface AuthResponse {
  token: string
  user: User
}

// Service types
export interface Service {
  id: string
  name: string
  url: string
  icon?: string
  description?: string
  status: 'online' | 'offline' | 'unknown'
  response_time?: number
  created_at: string
  updated_at?: string
}

export interface ServiceCreateRequest {
  name: string
  url: string
  icon?: string
  description?: string
}

export interface ServiceUpdateRequest {
  name?: string
  url?: string
  icon?: string
  description?: string
}

// Health check types
export interface HealthCheck {
  service_id: string
  status: 'online' | 'offline'
  response_time?: number
  timestamp: string
  error?: string
}

// Theme types
export interface Theme {
  mode: 'light' | 'dark'
  background?: string
  accent_color?: string
}

export interface UserPreferences {
  theme_mode: 'light' | 'dark'
  theme_background?: string
  theme_accent_color?: string
  updated_at?: string
}

export interface PreferencesUpdateRequest {
  theme_mode: 'light' | 'dark'
  theme_background?: string
  theme_accent_color?: string
}

// API response types
export interface ApiError {
  message: string
  code?: string
  details?: unknown
}

export interface ApiResponse<T> {
  data?: T
  error?: ApiError
  message?: string
}

// Paginated response for admin user list
export interface PaginatedUsersResponse {
  users: User[]
  total: number
  page: number
  total_pages: number
  limit: number
}

// Query params for user filtering
export interface UserFilterParams {
  search?: string
  role?: 'admin' | 'user' | ''
  page?: number
  limit?: number
}
