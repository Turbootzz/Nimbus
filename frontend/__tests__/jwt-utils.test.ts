/**
 * JWT Utilities Tests
 *
 * Tests JWT token creation and validation utilities using the jose library.
 * These are helper functions used by middleware but tested in isolation.
 */

import { describe, it, expect } from 'vitest'
import { SignJWT, jwtVerify } from 'jose'

const TEST_SECRET = 'test-secret-key-at-least-32-characters-long-for-security'
const ENCODED_SECRET = new TextEncoder().encode(TEST_SECRET)

async function createTestToken(
  payload: Record<string, unknown>,
  secret: string,
  expiresIn = '1h'
): Promise<string> {
  const encoder = new TextEncoder()
  return await new SignJWT(payload)
    .setProtectedHeader({ alg: 'HS256' })
    .setExpirationTime(expiresIn)
    .setIssuedAt()
    .sign(encoder.encode(secret))
}

describe('JWT Token Utilities', () => {
  describe('Token Creation', () => {
    it('should create valid JWT token with HS256', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)

      expect(token).toBeTruthy()
      expect(typeof token).toBe('string')

      // Verify token structure (header.payload.signature)
      const parts = token.split('.')
      expect(parts).toHaveLength(3)
    })

    it('should include payload data in token', async () => {
      const payload = { user_id: '123', role: 'admin' }
      const token = await createTestToken(payload, TEST_SECRET)

      const { payload: decoded } = await jwtVerify(token, ENCODED_SECRET)

      expect(decoded.user_id).toBe('123')
      expect(decoded.role).toBe('admin')
    })

    it('should set expiration time', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET, '1h')

      const { payload } = await jwtVerify(token, ENCODED_SECRET)

      expect(payload.exp).toBeTruthy()
      expect(typeof payload.exp).toBe('number')
    })

    it('should set issued at time', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)

      const { payload } = await jwtVerify(token, ENCODED_SECRET)

      expect(payload.iat).toBeTruthy()
      expect(typeof payload.iat).toBe('number')
    })
  })

  describe('Token Validation', () => {
    it('should validate token with correct secret', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)

      const result = await jwtVerify(token, ENCODED_SECRET, { algorithms: ['HS256'] })

      expect(result.payload.user_id).toBe('123')
    })

    it('should reject token with wrong secret', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)
      const wrongSecret = new TextEncoder().encode('wrong-secret-at-least-32-characters-long')

      await expect(jwtVerify(token, wrongSecret, { algorithms: ['HS256'] })).rejects.toThrow()
    })

    it('should reject expired token', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET, '-1s')

      await expect(jwtVerify(token, ENCODED_SECRET, { algorithms: ['HS256'] })).rejects.toThrow()
    })

    it('should reject token with wrong algorithm', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)

      // Attempt validation with algorithm whitelist that excludes HS256
      await expect(jwtVerify(token, ENCODED_SECRET, { algorithms: ['HS512'] })).rejects.toThrow()
    })
  })

  describe('Algorithm Security', () => {
    it('should use HS256 algorithm', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)

      const { protectedHeader } = await jwtVerify(token, ENCODED_SECRET)

      expect(protectedHeader.alg).toBe('HS256')
    })

    it('should validate only with HS256 in whitelist', async () => {
      const token = await createTestToken({ user_id: '123' }, TEST_SECRET)

      // This should succeed with HS256 in whitelist
      await expect(
        jwtVerify(token, ENCODED_SECRET, { algorithms: ['HS256'] })
      ).resolves.toBeTruthy()

      // This should fail without HS256 in whitelist
      await expect(jwtVerify(token, ENCODED_SECRET, { algorithms: ['HS384'] })).rejects.toThrow()
    })
  })
})

describe('Secret Encoding', () => {
  it('should encode secret consistently', () => {
    const encoder1 = new TextEncoder()
    const encoder2 = new TextEncoder()

    const encoded1 = encoder1.encode(TEST_SECRET)
    const encoded2 = encoder2.encode(TEST_SECRET)

    expect(encoded1).toEqual(encoded2)
  })

  it('should meet minimum length requirement', () => {
    expect(TEST_SECRET.length).toBeGreaterThanOrEqual(32)
  })
})

export { createTestToken, TEST_SECRET }
