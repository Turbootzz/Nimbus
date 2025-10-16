import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

/**
 * Next.js Middleware for Server-Side Authentication
 *
 * This middleware runs on the Edge before any page renders, providing:
 * - Server-side auth validation by calling backend API
 * - Protection against SSR/hydration mismatches
 * - Proper redirects before page load
 * - Automatic cleanup of invalid/expired tokens
 *
 * For protected routes, validates the token with the backend API.
 * Invalid tokens are cleared and the user is redirected to login.
 */

// Define public routes that don't require authentication
const publicPaths = ['/login', '/register']

// Define protected routes that require authentication
const protectedPaths = ['/dashboard', '/services', '/settings', '/admin']

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Get auth token from httpOnly cookie
  const authToken = request.cookies.get('auth_token')?.value

  // Check if current path is protected
  const isProtectedPath = protectedPaths.some((path) => pathname.startsWith(path))
  const isPublicPath = publicPaths.some((path) => pathname.startsWith(path))

  // If accessing a protected path, validate the token
  if (isProtectedPath) {
    if (!authToken) {
      // No token at all - redirect to login
      const loginUrl = new URL('/login', request.url)
      return NextResponse.redirect(loginUrl)
    }

    // Validate token with backend
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'
      const response = await fetch(`${apiUrl}/auth/me`, {
        headers: {
          Cookie: `auth_token=${authToken}`,
        },
      })

      // If token is invalid (401, 403, etc), clear cookie and redirect to login
      if (!response.ok) {
        const loginUrl = new URL('/login', request.url)
        const redirectResponse = NextResponse.redirect(loginUrl)
        // Clear the invalid cookie
        redirectResponse.cookies.set('auth_token', '', {
          maxAge: 0,
          path: '/',
        })
        return redirectResponse
      }
    } catch (error) {
      // Network error or backend down - redirect to login
      console.error('[Middleware] Failed to validate token:', error)
      const loginUrl = new URL('/login', request.url)
      return NextResponse.redirect(loginUrl)
    }
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
