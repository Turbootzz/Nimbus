/**
 * Get the API base URL for client-side requests.
 * Determines the URL at runtime based on the browser's location.
 *
 * Priority:
 * 1. NEXT_PUBLIC_API_URL environment variable (full API URL)
 * 2. Runtime detection using window.location:
 *    - Production (custom domain): Same origin path-based routing → /api/v1
 *    - Development (localhost/IP): Different port → http://localhost:8080/api/v1
 *
 * @returns The full API base URL including /api/v1
 */
export const getApiUrl = (): string => {
  const defaultPort = '8080'
  const apiPath = '/api/v1'

  // Priority 1: Use environment variable if set, normalize to include /api/v1
  if (process.env.NEXT_PUBLIC_API_URL) {
    let url = process.env.NEXT_PUBLIC_API_URL.trim()
    url = url.replace(/\/$/, '')
    if (!url.endsWith(apiPath)) {
      url = `${url}${apiPath}`
    }
    return url
  }

  // Priority 2: Runtime detection (client-side only)
  if (typeof window !== 'undefined' && window.location) {
    const protocol = window.location.protocol
    const hostname = window.location.hostname
    const isLocalhost = hostname === 'localhost' || hostname === '127.0.0.1'
    const isIpAddress = /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(hostname)

    // For localhost or IP addresses (development), use different port
    if (isLocalhost || isIpAddress) {
      const backendPort = process.env.NEXT_PUBLIC_API_PORT || defaultPort
      return `${protocol}//${hostname}:${backendPort}${apiPath}`
    }

    // For production domains, use same-origin path-based routing
    return `${protocol}//${hostname}${apiPath}`
  }

  // Priority 3: Fallback for SSR or edge cases
  return `http://localhost:${defaultPort}${apiPath}`
}
