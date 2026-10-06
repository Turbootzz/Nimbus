import { describe, expect, it } from 'vitest'
import { bestScore, fuzzyScore } from '@/lib/fuzzy'

describe('fuzzyScore', () => {
  it('needs the characters in order', () => {
    expect(fuzzyScore('plx', 'Plex')).not.toBeNull()
    expect(fuzzyScore('xpl', 'Plex')).toBeNull()
    expect(fuzzyScore('', 'Plex')).toBe(0)
  })

  it('prefers consecutive characters and word starts', () => {
    const rank = (query: string, names: string[]) =>
      [...names].sort((a, b) => (fuzzyScore(query, b) ?? -1) - (fuzzyScore(query, a) ?? -1))
    expect(rank('son', ['Jellyfin Server', 'Sonarr', 'Home Assistant'])[0]).toBe('Sonarr')
    expect(rank('ha', ['Pihole Admin', 'Home Assistant'])[0]).toBe('Home Assistant')
    expect(fuzzyScore('pve', 'Proxmox VE') ?? 0).toBeGreaterThan(
      fuzzyScore('pve', 'apt-get vendor') ?? 0
    )
  })

  it('weighs fields', () => {
    const byName = bestScore('nas', [{ text: 'NAS', weight: 3 }])
    const byUrl = bestScore('nas', [{ text: 'http://nas.lan', weight: 1 }])
    expect(byName!).toBeGreaterThan(byUrl!)
    expect(
      bestScore('zzz', [
        { text: 'NAS', weight: 3 },
        { text: undefined, weight: 1 },
      ])
    ).toBeNull()
  })
})

describe('fuzzyScore alignment', () => {
  it('finds a whole word later in the text', () => {
    // Taking the first a and d (in "Radarr") scored 10.7; the word scores about 15
    expect(fuzzyScore('admin', 'Radarr Admin Panel')!).toBeGreaterThan(14)
  })

  it('stays positive, so a heavier field always counts more', () => {
    expect(fuzzyScore('v', 'Proxmox Server with a long name')!).toBeGreaterThan(0)
    const name = bestScore('v', [{ text: 'Proxmox Server', weight: 3 }])!
    const url = bestScore('v', [{ text: 'http://pve.lan', weight: 1 }])!
    expect(name).toBeGreaterThan(url)
  })
})
