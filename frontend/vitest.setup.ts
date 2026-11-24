import { expect, afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import * as matchers from '@testing-library/jest-dom/matchers'

// Every render gets the English messages, like the root layout gives the
// app; a missing message or a broken one fails the test instead of
// rendering its key. A test's own wrapper goes inside.
vi.mock('@testing-library/react', async (importOriginal) => {
  const rtl = await importOriginal<typeof import('@testing-library/react')>()
  const { createElement } = await import('react')
  const { NextIntlClientProvider } = await import('next-intl')
  const messages = (await import('./messages/en.json')).default
  type Wrapper = React.JSXElementConstructor<{ children: React.ReactNode }>
  const withIntl =
    (Inner?: Wrapper) =>
    ({ children }: { children: React.ReactNode }) =>
      createElement(NextIntlClientProvider, {
        locale: 'en',
        messages,
        timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        onError: (error: Error) => {
          throw error
        },
        children: Inner ? createElement(Inner, null, children) : children,
      })
  return {
    ...rtl,
    render: ((ui: React.ReactNode, options?: { wrapper?: Wrapper }) =>
      rtl.render(ui, { ...options, wrapper: withIntl(options?.wrapper) })) as typeof rtl.render,
    renderHook: ((callback: (props: unknown) => unknown, options?: { wrapper?: Wrapper }) =>
      rtl.renderHook(callback, {
        ...options,
        wrapper: withIntl(options?.wrapper),
      })) as typeof rtl.renderHook,
  }
})

// Extend Vitest's expect with jest-dom matchers
expect.extend(matchers)

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// Set environment variables for tests
process.env.JWT_SECRET = 'test-jwt-secret-key-for-tests-minimum-32-characters-required'
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080/api/v1'
