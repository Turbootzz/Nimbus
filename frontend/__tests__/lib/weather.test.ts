import { afterEach, describe, expect, it, vi } from 'vitest'
import { describeWeather, searchLocations, weekday } from '@/lib/weather'

describe('describeWeather', () => {
  it('maps WMO codes to a label and a day or night icon', () => {
    expect(describeWeather(0, true)).toEqual({ label: 'Clear', icon: '☀️' })
    expect(describeWeather(0, false)).toEqual({ label: 'Clear', icon: '🌙' })
    expect(describeWeather(63).label).toBe('Rain')
    expect(describeWeather(81).label).toBe('Rain')
    expect(describeWeather(75).label).toBe('Snow')
    expect(describeWeather(99).label).toBe('Thunderstorm')
    expect(describeWeather(1234)).toEqual({ label: 'Unknown', icon: '🌡️' })
  })
})

describe('weekday', () => {
  it('reads the date without timezone shifts', () => {
    // 2026-09-27 is a Sunday everywhere
    expect(weekday('2026-09-27')).toBe(
      new Intl.DateTimeFormat(undefined, { weekday: 'short' }).format(new Date(2026, 8, 27))
    )
  })
})

describe('searchLocations', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('returns places with a readable label', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        results: [
          {
            name: 'Amsterdam',
            latitude: 52.37,
            longitude: 4.89,
            admin1: 'North Holland',
            country: 'Netherlands',
          },
          { name: 'Amsterdam', latitude: 42.94, longitude: -74.19, country: 'United States' },
        ],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const results = await searchLocations('Amster dam')
    expect(fetchMock.mock.calls[0][0]).toContain('name=Amster%20dam')
    expect(results[0]).toEqual({
      name: 'Amsterdam',
      latitude: 52.37,
      longitude: 4.89,
      label: 'Amsterdam, North Holland, Netherlands',
    })
    expect(results[1].label).toBe('Amsterdam, United States')
  })

  it('returns nothing for no matches and throws on errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }))
    expect(await searchLocations('zzzz')).toEqual([])

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 503 }))
    await expect(searchLocations('x')).rejects.toThrow('503')
  })
})
