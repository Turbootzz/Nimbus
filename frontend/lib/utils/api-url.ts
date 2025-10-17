/**
 * Get the API base URL for client-side requests.
 * Determines the URL at runtime based on the browser's location.
 *
 * Priority:
 * 1. NEXT_PUBLIC_API_URL environment variable (base URL with or without /api/v1)
 * 2. Runtime detection using window.location
 * 3. Fallback to localhost:8080
 *
 * @returns The full API base URL including /api/v1
 */
export const getApiUrl = (): string => {
  const defaultPort = '8080'
  const apiPath = '/api/v1'

  // Priority 1: Use environment variable if set, normalize to include /api/v1
  if (process.env.NEXT_PUBLIC_API_URL) {
    let url = process.env.NEXT_PUBLIC_API_URL.trim()

    // Remove trailing slash if present
    url = url.replace(/\/$/, '')

    // If URL doesn't already end with /api/v1, append it
    if (!url.endsWith(apiPath)) {
      url = `${url}${apiPath}`
    }

    return url
  }

  // Priority 2: Runtime detection (client-side only)
  if (typeof window !== 'undefined' && window.location) {
    const protocol = window.location.protocol // http: or https:
    const hostname = window.location.hostname // e.g., 192.168.1.100 or localhost

    // Check if we should add a port
    const isCustomDomain = hostname.includes('.') && hostname !== 'localhost'
    const currentPort = window.location.port
    const isStandardPort = !currentPort || currentPort === '80' || currentPort === '443'

    // For custom domains with reverse proxy (like Nginx Proxy Manager)
    // Use subdomain: api.domain.com instead of domain.com:8080
    if (isCustomDomain && isStandardPort) {
      // Replace first subdomain with 'api' or prepend 'api.'
      const parts = hostname.split('.')
      if (parts.length >= 2) {
        // If hostname is already api.domain.com, keep it
        if (parts[0] === 'api') {
          return `${protocol}//${hostname}${apiPath}`
        }
        // Otherwise, replace first part with 'api' (e.g., nimbus.domain.com -> api.domain.com)
        parts[0] = 'api'
        return `${protocol}//${parts.join('.')}${apiPath}`
      }
    }

    // For localhost, IPs, or custom port scenarios, add the backend port
    const backendPort = process.env.NEXT_PUBLIC_API_PORT || defaultPort
    return `${protocol}//${hostname}:${backendPort}${apiPath}`
  }

  // Priority 3: Fallback for SSR or edge cases
  return `http://localhost:${defaultPort}${apiPath}`
}
