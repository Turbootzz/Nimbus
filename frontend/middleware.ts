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

// fail-fast validation
const JWT_SECRET = process.env.JWT_SECRET
if (!JWT_SECRET) {
  throw new Error('JWT_SECRET must be set in environment variables')
}
if (JWT_SECRET.length < 32) {
  throw new Error('JWT_SECRET must be at least 32 characters for security')
}

// Cache the encoded secret to avoid encoding on every validation
const ENCODED_SECRET = new TextEncoder().encode(JWT_SECRET)

// JWT verification options for security
const JWT_VERIFY_OPTIONS = {
  algorithms: ['HS256'], // Only allow HS256 to prevent algorithm confusion attacks
}

/**
 * Validates JWT token locally without calling backend API
 * Much faster and reduces backend load
 *
 * Security features:
 * - Algorithm whitelist (HS256 only)
 * - Signature verification
 * - Expiration checking
 */
async function validateToken(token: string): Promise<boolean> {
  try {
    await jwtVerify(token, ENCODED_SECRET, JWT_VERIFY_OPTIONS)
    return true
  } catch (error) {
    // Log validation failure for observability (don't log token for security)
    console.warn(
      '[Middleware] Token validation failed:',
      error instanceof Error ? error.message : 'Unknown error'
    )
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

  // If accessing public auth pages (login/register) with token, clear invalid ones
  if (isPublicPath && authToken) {
    // Validate token signature - if invalid, clear it
    const isValid = await validateToken(authToken)
    if (!isValid) {
      // Clear invalid cookie
      const response = NextResponse.next()
      response.cookies.set('auth_token', '', { maxAge: 0, path: '/' })
      return response
    }
  }

  // If accessing root path, redirect based on authentication status
  if (pathname === '/') {
    if (authToken) {
      // Validate token signature before redirecting
      const isValid = await validateToken(authToken)
      if (isValid) {
        // Authenticated users go to dashboard
        return NextResponse.redirect(new URL('/dashboard', request.url))
      }
    }
    // Unauthenticated or invalid token - go to login
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
