import { describe, it, expect } from 'vitest'
import { isValidUrl } from '@/lib/utils/url'

describe('isValidUrl', () => {
  it.each([
    ['https://example.com', true],
    ['http://localhost:3000', true],
    ['https://github.com/owner/repo', true],
    ['http://192.168.1.5:8080/path', true],
  ])('accepts %s', (input, expected) => {
    expect(isValidUrl(input)).toBe(expected)
  })

  it.each([
    ['', false],
    ['not a url', false],
    ['example.com', false], // no scheme
    ['ftp://files.example.com', false], // not http(s)
    ['mailto:hi@example.com', false],
    ['file:///etc/passwd', false],
    ['javascript:alert(1)', false],
  ])('rejects %s', (input, expected) => {
    expect(isValidUrl(input)).toBe(expected)
  })
})
