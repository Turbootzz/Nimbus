// User types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user'
  provider: 'local' | 'google' | 'github' | 'discord' | 'oidc'
  avatar_url?: string
  email_verified: boolean
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
export type CardSize = '1x1' | '2x1' | '2x2'
export type CardScale = 'small' | 'medium' | 'large'
export type ViewMode = 'grid' | 'list'

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
  card_size: CardSize
  group_id?: string
  monitoring_enabled: boolean
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
  card_size?: CardSize
  group_id?: string
  monitoring_enabled?: boolean
}

export interface ServiceUpdateRequest {
  name?: string
  url?: string
  icon?: string
  icon_type?: IconType
  icon_image_path?: string
  description?: string
  card_size?: CardSize
  group_id?: string
  monitoring_enabled?: boolean
}

export interface ServicePosition {
  id: string
  position: number
}

export interface ServiceReorderRequest {
  services: ServicePosition[]
}

// Group types
export interface Group {
  id: string
  name: string
  color: string
  position: number
  is_default: boolean
  monitoring_enabled: boolean
  created_at: string
  updated_at?: string
}

export interface GroupCreateRequest {
  name: string
  color?: string
  monitoring_enabled?: boolean
}

export interface GroupUpdateRequest {
  name?: string
  color?: string
  monitoring_enabled?: boolean
}

export interface GroupPosition {
  id: string
  position: number
}

export interface GroupReorderRequest {
  groups: GroupPosition[]
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
  mode: 'light' | 'dark' | 'auto'
  background?: string
  accent_color?: string
}

export interface UserPreferences {
  theme_mode: 'light' | 'dark' | 'auto'
  theme_background?: string
  theme_accent_color?: string
  open_in_new_tab: boolean
  enable_card_resizing: boolean
  enable_service_grouping: boolean
  card_scale: CardScale
  view_mode: ViewMode
  updated_at?: string
}

export interface PreferencesUpdateRequest {
  theme_mode?: 'light' | 'dark' | 'auto'
  theme_background?: string | null
  theme_accent_color?: string | null
  open_in_new_tab?: boolean
  enable_card_resizing?: boolean
  enable_service_grouping?: boolean
  card_scale?: CardScale
  view_mode?: ViewMode
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

// OAuth types
export type OAuthProvider = 'google' | 'github' | 'discord' | 'oidc'

export interface OAuthProviderConfig {
  name: OAuthProvider
  enabled: boolean
  displayName: string
  icon: string
}

export interface OAuthProviderStatus {
  name: string
  enabled: boolean
  configure: boolean
}

// Webhook types
export type WebhookFormat = 'generic' | 'discord' | 'slack'

export interface WebhookTriggers {
  on_offline: boolean
  on_online: boolean
}

export interface Webhook {
  id: string
  name: string
  url: string
  enabled: boolean
  triggers: WebhookTriggers
  format: WebhookFormat
  retry_count: number
  retry_delay_seconds: number
  last_triggered_at?: string
  last_success_at?: string
  consecutive_failures: number
  total_sent: number
  total_failed: number
  created_at: string
  updated_at: string
}

export interface WebhookCreateRequest {
  name: string
  url: string
  enabled?: boolean
  triggers?: WebhookTriggers
  format?: WebhookFormat
  retry_count?: number
  retry_delay_seconds?: number
}

export interface WebhookUpdateRequest {
  name?: string
  url?: string
  enabled?: boolean
  triggers?: WebhookTriggers
  format?: WebhookFormat
  retry_count?: number
  retry_delay_seconds?: number
}

export interface WebhookLog {
  id: string
  webhook_id: string
  service_id: string
  service_name: string
  old_status: string
  new_status: string
  success: boolean
  status_code?: number
  error_message?: string
  response_time_ms?: number
  created_at: string
}

export interface WebhookTestResult {
  success: boolean
  message?: string
  status_code?: number
  response_time_ms?: number
  error?: string
}

// Password reset types
export interface ForgotPasswordRequest {
  email: string
}

export interface ResetPasswordRequest {
  token: string
  new_password: string
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

export interface SMTPStatusResponse {
  configured: boolean
  source: 'env' | 'database' | 'none'
}

export interface UpdateSMTPSettingsRequest {
  smtp_host: string
  smtp_port: string
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_enabled: string
}

// Setup types
export interface SetupStatusResponse {
  needs_setup: boolean
}

export interface RegistrationStatusResponse {
  enabled: boolean
}

// System settings types
export interface SystemSetting {
  key: string
  value: string
  updated_at: string
  updated_by?: string
}

export interface SystemSettingsResponse {
  settings: SystemSetting[]
}

export interface UpdateSettingRequest {
  value: string
}
