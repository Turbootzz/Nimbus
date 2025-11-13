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
export type IconType = 'emoji' | 'image_upload' | 'image_url'

export interface Service {
  id: string
  name: string
  url: string
  icon?: string
  icon_type: IconType
  icon_image_path?: string
  description?: string
  status: 'online' | 'offline' | 'unknown'
  response_time?: number
  position: number
  created_at: string
  updated_at?: string
}

export interface ServiceCreateRequest {
  name: string
  url: string
  icon?: string
  icon_type?: IconType
  icon_image_path?: string
  description?: string
}

export interface ServiceUpdateRequest {
  name?: string
  url?: string
  icon?: string
  icon_type?: IconType
  icon_image_path?: string
  description?: string
}

export interface ServicePosition {
  id: string
  position: number
}

export interface ServiceReorderRequest {
  services: ServicePosition[]
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
  open_in_new_tab: boolean
  updated_at?: string
}

export interface PreferencesUpdateRequest {
  theme_mode?: 'light' | 'dark'
  theme_background?: string | null
  theme_accent_color?: string | null
  open_in_new_tab?: boolean
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

// Metrics and monitoring types
export interface StatusLog {
  id: string
  service_id: string
  status: 'online' | 'offline' | 'unknown'
  response_time?: number
  error_message?: string
  checked_at: string
}

export interface MetricDataPoint {
  timestamp: string
  check_count: number
  online_count: number
  uptime_percentage: number
  avg_response_time: number
}

export interface TimeRange {
  start: string
  end: string
}

export interface MetricsResponse {
  service_id: string
  time_range: TimeRange
  uptime_percentage: number
  total_checks: number
  online_count: number
  offline_count: number
  avg_response_time: number
  min_response_time: number
  max_response_time: number
  data_points: MetricDataPoint[]
}

export type TimeRangeOption = '1h' | '6h' | '24h' | '7d' | '30d'
