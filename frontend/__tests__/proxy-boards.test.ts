import { beforeAll, describe, expect, it } from 'vitest'
import { NextRequest } from 'next/server'
import { SignJWT } from 'jose'

const SECRET = 'test-jwt-secret-key-for-middleware-minimum-32-characters'

beforeAll(() => {
  process.env.JWT_SECRET = SECRET
})

async function request(path: string, token?: string) {
  const { default: proxy } = await import('@/proxy')
  const req = new NextRequest(`http://localhost${path}`)
  if (token) req.cookies.set('auth_token', token)
  return proxy(req)
}

describe('proxy on board pages', () => {
  it('sends a visitor without a session to the login page, kiosk or not', async () => {
    for (const path of ['/dashboard', '/board/family', '/board/family?kiosk=1']) {
      const response = await request(path)
      expect(response.headers.get('location'), path).toBe('http://localhost/login')
    }
  })

  it('clears a bad token on a board page too', async () => {
    const response = await request('/board/family?kiosk=1', 'not-a-jwt')
    expect(response.headers.get('location')).toBe('http://localhost/login')
    expect(response.headers.get('set-cookie')).toContain('auth_token=;')
  })

  it('lets a signed-in user through', async () => {
    const token = await new SignJWT({ sub: 'user-1' })
      .setProtectedHeader({ alg: 'HS256' })
      .setExpirationTime('1h')
      .sign(new TextEncoder().encode(SECRET))
    const response = await request('/board/family?kiosk=1', token)
    expect(response.headers.get('location')).toBeNull()
  })
})
