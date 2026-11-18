import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import InstallAppMenuItem from '@/components/InstallAppMenuItem'
import { API_CACHE, CLEAR_API_CACHE, clearApiCache } from '@/lib/pwa'

// What Chromium fires when the app can be installed
function fireInstallable() {
  const event = Object.assign(new Event('beforeinstallprompt', { cancelable: true }), {
    prompt: vi.fn().mockResolvedValue(undefined),
  })
  act(() => {
    window.dispatchEvent(event)
  })
  return event
}

function setStandalone(standalone: boolean) {
  window.matchMedia = vi.fn().mockReturnValue({ matches: standalone }) as never
}

afterEach(() => {
  act(() => {
    window.dispatchEvent(new Event('appinstalled'))
  })
  vi.unstubAllGlobals()
})

describe('InstallAppMenuItem', () => {
  it('renders nothing while the browser offers no install (Safari, Firefox)', () => {
    setStandalone(false)
    const { container } = render(<InstallAppMenuItem />)
    expect(container).toBeEmptyDOMElement()
  })

  it('offers the install captured before it mounted and prompts once', () => {
    setStandalone(false)
    const event = fireInstallable()
    expect(event.defaultPrevented).toBe(true)

    const onSelect = vi.fn()
    render(<InstallAppMenuItem onSelect={onSelect} />)
    fireEvent.click(screen.getByRole('button', { name: 'Install app' }))

    expect(event.prompt).toHaveBeenCalledOnce()
    expect(onSelect).toHaveBeenCalledOnce()
    // The event is spent, so the entry goes away
    expect(screen.queryByRole('button', { name: 'Install app' })).not.toBeInTheDocument()
  })

  it('hides once the app is installed', () => {
    setStandalone(false)
    fireInstallable()
    render(<InstallAppMenuItem />)
    expect(screen.getByRole('button', { name: 'Install app' })).toBeInTheDocument()

    act(() => {
      window.dispatchEvent(new Event('appinstalled'))
    })
    expect(screen.queryByRole('button', { name: 'Install app' })).not.toBeInTheDocument()
  })

  it('renders nothing when running as the installed app', () => {
    setStandalone(true)
    fireInstallable()
    const { container } = render(<InstallAppMenuItem />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('clearApiCache', () => {
  it('tells the worker and deletes the API cache', async () => {
    const postMessage = vi.fn()
    const deleteCache = vi.fn().mockResolvedValue(true)
    vi.stubGlobal('navigator', { serviceWorker: { controller: { postMessage } } })
    vi.stubGlobal('caches', { delete: deleteCache })

    await clearApiCache()

    expect(postMessage).toHaveBeenCalledWith({ type: CLEAR_API_CACHE })
    expect(deleteCache).toHaveBeenCalledWith(API_CACHE)
  })

  it('does not throw without a worker or when Cache Storage fails', async () => {
    vi.stubGlobal('navigator', {})
    vi.stubGlobal('caches', { delete: vi.fn().mockRejectedValue(new Error('denied')) })
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    await expect(clearApiCache()).resolves.toBeUndefined()
    warn.mockRestore()
  })
})
