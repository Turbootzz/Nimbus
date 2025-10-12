// User types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user'
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
  accentColor?: string
}

export interface UserPreferences {
  user_id: string
  theme: Theme
  created_at: string
  updated_at?: string
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
