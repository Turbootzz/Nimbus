import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

/**
 * Next.js Middleware for Server-Side Authentication
 *
 * This middleware runs on the Edge before any page renders, providing:
 * - Server-side auth validation (no client-side localStorage checks)
 * - Protection against SSR/hydration mismatches
 * - Proper redirects before page load
 * - Validation of httpOnly cookies
 *
 * This only checks cookie PRESENCE for UX.
 * The backend Go API validates JWT signature, expiration, and claims for actual security.
 */

// Define public routes that don't require authentication
const publicPaths = ['/login', '/register']

// Define protected routes that require authentication
const protectedPaths = ['/dashboard', '/services', '/settings']

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Get auth token from httpOnly cookie
  const authToken = request.cookies.get('auth_token')?.value

  // Check if current path is protected
  const isProtectedPath = protectedPaths.some((path) => pathname.startsWith(path))
  const isPublicPath = publicPaths.some((path) => pathname.startsWith(path))

  // If accessing a protected path without auth token, redirect to login
  if (isProtectedPath && !authToken) {
    const loginUrl = new URL('/login', request.url)
    return NextResponse.redirect(loginUrl)
  }

  // If accessing public auth pages (login/register) with valid token, redirect to dashboard
  if (isPublicPath && authToken) {
    // Note: We only check cookie presence here. Backend validates the JWT.
    const dashboardUrl = new URL('/dashboard', request.url)
    return NextResponse.redirect(dashboardUrl)
  }

  // If accessing root path, redirect based on auth status
  if (pathname === '/') {
    if (authToken) {
      const dashboardUrl = new URL('/dashboard', request.url)
      return NextResponse.redirect(dashboardUrl)
    } else {
      const loginUrl = new URL('/login', request.url)
      return NextResponse.redirect(loginUrl)
    }
  }

  // Allow the request to continue
  return NextResponse.next()
}

// Configure which paths the middleware should run on
export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder files
     */
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
}
