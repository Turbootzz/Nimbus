import { expect, afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'
import * as matchers from '@testing-library/jest-dom/matchers'

// Extend Vitest's expect with jest-dom matchers
expect.extend(matchers)

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// Set environment variables for tests
process.env.JWT_SECRET = 'test-jwt-secret-key-for-tests-minimum-32-characters-required'
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080/api/v1'
