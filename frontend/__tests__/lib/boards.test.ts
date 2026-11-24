import { beforeEach, describe, expect, it } from 'vitest'
import {
  boardHref,
  defaultBoard,
  findBoard,
  getLastBoard,
  homeRedirect,
  moveTargets,
  rememberBoard,
  showBoardSwitcher,
} from '@/lib/boards'
import type { Board } from '@/types'

const store: Record<string, string> = {}
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
  },
})

const board = (id: string, slug: string, extra: Partial<Board> = {}): Board => ({
  id,
  name: slug,
  slug,
  position: 0,
  is_default: false,
  is_shared: false,
  visibility: 'all',
  owned: true,
  can_edit: true,
  ...extra,
})

const home = board('b-home', 'home', { is_default: true })
const media = board('b-media', 'media')
// Someone else's shared board, with the slug of one of the user's own
const theirMedia = board('b-their-media', 'media', {
  owned: false,
  can_edit: false,
  is_shared: true,
})
const family = board('b-family', 'family', { owned: false, can_edit: false, is_shared: true })
const boards = [home, media, theirMedia, family]

describe('boards', () => {
  beforeEach(() => localStorage.removeItem('nimbus-last-board'))

  it('finds a board by ID, then own slug, then shared slug', () => {
    expect(findBoard(boards, 'b-their-media')).toBe(theirMedia)
    expect(findBoard(boards, 'media')).toBe(media)
    expect(findBoard(boards, 'family')).toBe(family)
    expect(findBoard(boards, 'nope')).toBeUndefined()
    expect(defaultBoard(boards)).toBe(home)
  })

  it('links the default board to /dashboard and a taken slug by ID', () => {
    expect(boardHref(home, boards)).toBe('/dashboard')
    expect(boardHref(media, boards)).toBe('/board/media')
    expect(boardHref(theirMedia, boards)).toBe('/board/b-their-media')
    expect(boardHref(family, boards, true)).toBe('/board/family?kiosk=1')
  })

  it('shows the switcher only when there is something to choose', () => {
    expect(showBoardSwitcher([])).toBe(false)
    expect(showBoardSwitcher([home])).toBe(false)
    expect(showBoardSwitcher([home, family])).toBe(true)
  })

  it("moves tiles only between the user's own boards", () => {
    expect(moveTargets([home, family], home.id)).toEqual([])
    expect(moveTargets(boards, home.id)).toEqual([media])
    expect(moveTargets(boards, media.id)).toEqual([home])
    expect(moveTargets(boards), 'no board ID is the default board').toEqual([media])
  })

  it('remembers the default board as /dashboard itself', () => {
    expect(getLastBoard()).toBeNull()
    rememberBoard(media)
    expect(getLastBoard()).toBe('b-media')
    rememberBoard(home)
    expect(getLastBoard()).toBe('')
  })

  it('sends /dashboard to the remembered board, or a new user to the shared one', () => {
    expect(homeRedirect(boards, 'b-media', false)).toBe(media)
    expect(homeRedirect(boards, '', true)).toBeUndefined()
    expect(homeRedirect(boards, 'b-home', false)).toBeUndefined()
    expect(homeRedirect(boards, 'b-gone', true)).toBeUndefined()
    // Never chose a board, nothing of their own yet: the first shared board
    expect(homeRedirect(boards, null, true)).toBe(theirMedia)
    expect(homeRedirect(boards, null, false)).toBeUndefined()
    expect(homeRedirect([home], null, true)).toBeUndefined()
  })
})
