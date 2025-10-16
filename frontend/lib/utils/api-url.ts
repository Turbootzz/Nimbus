/**
 * Get the API base URL for client-side requests.
 * Determines the URL at runtime based on the browser's location.
 *
 * Priority:
 * 1. NEXT_PUBLIC_API_URL environment variable (if set)
 * 2. Runtime detection using window.location + configurable port
 * 3. Fallback to localhost:8080
 *
 * @returns The full API base URL including /api/v1
 */
export const getApiUrl = (): string => {
  const defaultPort = '8080'
  const apiPath = '/api/v1'

  // Priority 1: Use environment variable if set
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL
  }

  // Priority 2: Runtime detection (client-side only)
  if (typeof window !== 'undefined' && window.location) {
    const protocol = window.location.protocol // http: or https:
    const hostname = window.location.hostname // e.g., 192.168.1.100 or localhost
    const backendPort = process.env.NEXT_PUBLIC_API_PORT || defaultPort
    return `${protocol}//${hostname}:${backendPort}${apiPath}`
  }

  // Priority 3: Fallback for SSR or edge cases
  return `http://localhost:${defaultPort}${apiPath}`
}
