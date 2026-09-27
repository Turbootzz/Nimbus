import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useDashboardStream } from '@/hooks/useDashboardStream'
import { api } from '@/lib/api'
import type { Snapshot } from '@/types'

vi.mock('@/lib/api', () => ({
  api: {
    getDashboardData: vi.fn(),
    getCurrentUser: vi.fn(),
  },
}))

// FakeEventSource records instances so tests can drive them
class FakeEventSource {
  static instances: FakeEventSource[] = []
  url: string
  withCredentials: boolean
  closed = false
  onerror: (() => void) | null = null
  private listeners: Record<string, ((e: MessageEvent) => void)[]> = {}

  constructor(url: string, init?: EventSourceInit) {
    this.url = url
    this.withCredentials = init?.withCredentials ?? false
    FakeEventSource.instances.push(this)
  }
  addEventListener(name: string, fn: (e: MessageEvent) => void) {
    ;(this.listeners[name] ??= []).push(fn)
  }
  close() {
    this.closed = true
  }
  emit(name: string, data: unknown) {
    act(() => {
      for (const fn of this.listeners[name] ?? []) {
        fn(new MessageEvent(name, { data: JSON.stringify(data) }))
      }
    })
  }
  fail() {
    act(() => this.onerror?.())
  }
  static latest() {
    return FakeEventSource.instances[FakeEventSource.instances.length - 1]
  }
}

const snapshot = (id: string, payload: unknown): Snapshot => ({
  source_kind: 'widget',
  source_id: id,
  payload,
  fetched_at: '2026-09-27T12:00:00Z',
  stale: false,
})

describe('useDashboardStream', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource)
    vi.mocked(api.getDashboardData).mockResolvedValue({ data: [snapshot('w1', { t: 1 })] })
    vi.mocked(api.getCurrentUser).mockResolvedValue({ data: undefined })
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  const setup = () => {
    const onServiceStatus = vi.fn()
    const onResync = vi.fn()
    const hook = renderHook(() => useDashboardStream({ onServiceStatus, onResync }))
    return { ...hook, onServiceStatus, onResync }
  }

  it('loads snapshots, then follows the stream', async () => {
    const { result, onServiceStatus } = setup()
    await waitFor(() => expect(result.current.snapshots['widget:w1']?.payload).toEqual({ t: 1 }))

    const source = FakeEventSource.latest()
    expect(source.url).toMatch(/\/api\/v1\/dashboard\/stream$/)
    expect(source.withCredentials).toBe(true)

    source.emit('hello', {})
    expect(result.current.connected).toBe(true)

    source.emit('snapshot', snapshot('w1', { t: 2 }))
    source.emit('snapshot', snapshot('w2', { t: 3 }))
    expect(result.current.snapshots['widget:w1'].payload).toEqual({ t: 2 })
    expect(result.current.snapshots['widget:w2'].payload).toEqual({ t: 3 })

    source.emit('service_status', { id: 's1', status: 'offline' })
    expect(onServiceStatus).toHaveBeenCalledWith({ id: 's1', status: 'offline' })
  })

  it('reconnects with backoff, polls meanwhile and resyncs after', async () => {
    vi.useFakeTimers()
    const { result, onResync } = setup()
    const first = FakeEventSource.latest()

    first.fail()
    expect(first.closed).toBe(true)
    expect(result.current.connected).toBe(false)

    // 1s later: second try, which fails too; the next wait is 2s
    act(() => vi.advanceTimersByTime(1000))
    expect(FakeEventSource.instances).toHaveLength(2)
    FakeEventSource.latest().fail()
    act(() => vi.advanceTimersByTime(1999))
    expect(FakeEventSource.instances).toHaveLength(2)
    act(() => vi.advanceTimersByTime(1))
    expect(FakeEventSource.instances).toHaveLength(3)

    // Third failure in a row checks the session (401 goes to login)
    expect(api.getCurrentUser).not.toHaveBeenCalled()
    FakeEventSource.latest().fail()
    expect(api.getCurrentUser).toHaveBeenCalledTimes(1)

    // While down, the dashboard is polled every 30s
    act(() => vi.advanceTimersByTime(30000))
    expect(onResync).toHaveBeenCalled()
    const polls = onResync.mock.calls.length

    // Back online: one catch-up resync, then no more polling
    const recovered = FakeEventSource.latest()
    recovered.emit('hello', {})
    expect(result.current.connected).toBe(true)
    expect(onResync).toHaveBeenCalledTimes(polls + 1)
    act(() => vi.advanceTimersByTime(60000))
    expect(onResync).toHaveBeenCalledTimes(polls + 1)
  })

  it('caps the retry delay at 30 seconds', () => {
    vi.useFakeTimers()
    setup()
    for (let i = 0; i < 8; i++) {
      FakeEventSource.latest().fail()
      act(() => vi.advanceTimersByTime(30000))
    }
    const count = FakeEventSource.instances.length
    FakeEventSource.latest().fail()
    act(() => vi.advanceTimersByTime(30000))
    expect(FakeEventSource.instances).toHaveLength(count + 1)
  })

  it('closes the stream on unmount', () => {
    vi.useFakeTimers()
    const { unmount } = setup()
    const source = FakeEventSource.latest()
    source.fail() // a retry is pending
    unmount()
    expect(source.closed).toBe(true)
    act(() => vi.advanceTimersByTime(60000))
    expect(FakeEventSource.instances).toHaveLength(1)
  })
})
