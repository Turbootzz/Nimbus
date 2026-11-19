import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import OfflineServices from '@/components/OfflineServices'
import { API_CACHE, CACHED_AT_HEADER } from '@/lib/pwa'

const SYNCED_AT = '2026-11-18T10:30:00.000Z'

// Cache Storage holding what the worker stored for GET /services
function stubCaches(response?: Response) {
  const match = vi.fn().mockResolvedValue(response)
  vi.stubGlobal('caches', {
    has: vi.fn(async (name: string) => name === API_CACHE && response !== undefined),
    open: vi.fn().mockResolvedValue({ match }),
  })
  return match
}

afterEach(() => vi.unstubAllGlobals())

describe('OfflineServices', () => {
  it('lists the last synced services read-only, with the time of the sync', async () => {
    const services = [
      { id: '1', name: 'Jellyfin', url: 'http://jellyfin.lan' },
      { id: '2', name: 'Pi-hole', url: 'http://pihole.lan/admin' },
    ]
    const match = stubCaches(
      new Response(JSON.stringify(services), { headers: { [CACHED_AT_HEADER]: SYNCED_AT } })
    )

    render(<OfflineServices />)

    expect(screen.getByRole('heading', { name: 'You are offline' })).toBeInTheDocument()
    expect(await screen.findByText('Jellyfin')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'http://pihole.lan/admin' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent(new Date(SYNCED_AT).toLocaleString())
    expect(match).toHaveBeenCalledWith('http://localhost:8080/api/v1/services')
    // Nothing to edit or add while offline
    expect(screen.getAllByRole('button')).toHaveLength(1)
  })

  it('says so when nothing was saved on this device', async () => {
    stubCaches()
    render(<OfflineServices />)
    expect(await screen.findByText(/No services were saved/)).toBeInTheDocument()
  })

  it('still renders when Cache Storage is unavailable', async () => {
    vi.stubGlobal('caches', { has: vi.fn().mockRejectedValue(new Error('denied')) })
    render(<OfflineServices />)
    expect(await screen.findByText(/No services were saved/)).toBeInTheDocument()
  })
})
