import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { jwtVerify } from 'jose'

/**
 * Next.js Middleware for Server-Side Authentication
 *
 * This middleware runs on the Edge before any page renders, providing:
 * - Fast JWT validation without backend calls (improves performance)
 * - Protection against SSR/hydration mismatches
 * - Proper redirects before page load
 * - Automatic cleanup of invalid/expired tokens
 *
 * Performance: Validates JWT tokens locally by checking signature and expiration
 * instead of making backend API calls on every request.
 */

// Define public routes that don't require authentication
const publicPaths = ['/login', '/register']

// Define protected routes that require authentication
const protectedPaths = ['/dashboard', '/services', '/settings', '/admin']

// Validate JWT_SECRET at module load time for fail-fast behavior
const JWT_SECRET = process.env.JWT_SECRET
if (!JWT_SECRET) {
  console.error('[Middleware] CRITICAL: JWT_SECRET not set in environment')
  if (process.env.NODE_ENV === 'production') {
    throw new Error('JWT_SECRET must be set in production environment')
  }
}

/**
 * Validates JWT token locally without calling backend API
 * Much faster and reduces backend load
 */
async function validateToken(token: string): Promise<boolean> {
  if (!JWT_SECRET) {
    return false
  }

  try {
    // Verify JWT signature and expiration
    const secret = new TextEncoder().encode(JWT_SECRET)
    await jwtVerify(token, secret)
    return true
  } catch {
    // Token is invalid, expired, or malformed
    return false
  }
}

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

    // Validate token locally (fast, no backend call)
    const isValid = await validateToken(authToken)
    if (!isValid) {
      // Token is invalid or expired - clear cookie and redirect to login
      const loginUrl = new URL('/login', request.url)
      const redirectResponse = NextResponse.redirect(loginUrl)
      // Clear the invalid cookie
      redirectResponse.cookies.set('auth_token', '', {
        maxAge: 0,
        path: '/',
      })
      return redirectResponse
    }
  }

  // If accessing public auth pages (login/register) with valid token, redirect to dashboard
  if (isPublicPath && authToken) {
    // Validate token before redirecting
    const isValid = await validateToken(authToken)
    if (isValid) {
      const dashboardUrl = new URL('/dashboard', request.url)
      return NextResponse.redirect(dashboardUrl)
    } else {
      // Clear invalid cookie
      const response = NextResponse.next()
      response.cookies.set('auth_token', '', { maxAge: 0, path: '/' })
      return response
    }
  }

  // If accessing root path, redirect based on auth status
  if (pathname === '/') {
    if (authToken) {
      // Validate token before redirecting to dashboard
      const isValid = await validateToken(authToken)
      if (isValid) {
        const dashboardUrl = new URL('/dashboard', request.url)
        return NextResponse.redirect(dashboardUrl)
      } else {
        // Clear invalid cookie and redirect to login
        const loginUrl = new URL('/login', request.url)
        const redirectResponse = NextResponse.redirect(loginUrl)
        redirectResponse.cookies.set('auth_token', '', { maxAge: 0, path: '/' })
        return redirectResponse
      }
    }
    // No token - redirect to login
    const loginUrl = new URL('/login', request.url)
    return NextResponse.redirect(loginUrl)
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
