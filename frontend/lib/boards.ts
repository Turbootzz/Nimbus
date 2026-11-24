import type { Board } from '@/types'

// Pure rules for boards: which pages are a dashboard, how a board is found
// from its URL and linked to, and when the switcher shows.

// What the API answers for a board the user may not see (anymore)
export const BOARD_NOT_FOUND = 'Board not found'

// The pages that show a board: /dashboard (the default board) and
// /board/<slug>. They share the canvas layout and the kiosk view.
export function isDashboardPath(pathname: string): boolean {
  return pathname === '/dashboard' || pathname.startsWith('/board/')
}

// The board /dashboard shows
export function defaultBoard(boards: Board[]): Board | undefined {
  return boards.find((b) => b.owned && b.is_default)
}

// The board a /board/<param> URL means: an ID, the slug of one of the user's
// own boards, or the slug of a board shared with them, in that order. Slugs
// are unique per owner, so someone's shared board can have the slug of one
// of the user's own; that board is then reached by its ID.
export function findBoard(boards: Board[], param: string): Board | undefined {
  return (
    boards.find((b) => b.id === param) ??
    boards.find((b) => b.owned && b.slug === param) ??
    boards.find((b) => b.slug === param)
  )
}

// Where a board lives: /dashboard for the default board, else /board/<slug>,
// or /board/<id> when the slug would open another board
export function boardHref(board: Board, boards: Board[], kiosk = false): string {
  const param = findBoard(boards, board.slug)?.id === board.id ? board.slug : board.id
  const path = board.owned && board.is_default ? '/dashboard' : `/board/${param}`
  return kiosk ? `${path}?kiosk=1` : path
}

// The switcher is for choosing: with one board there is nothing to choose,
// and the dashboard looks as it did before boards existed
export function showBoardSwitcher(boards: Board[]): boolean {
  return boards.length > 1
}

// The boards a tile can move to: the user's own, except the one it is on
export function moveTargets(boards: Board[], currentId?: string): Board[] {
  const own = boards.filter((b) => b.owned)
  const current = currentId ?? defaultBoard(boards)?.id
  return own.length > 1 ? own.filter((b) => b.id !== current) : []
}

// The board the user opened last (a slug or an ID), so /dashboard can bring
// them back to it. '' means the default board was chosen; null: never chosen.
const LAST_BOARD_KEY = 'nimbus-last-board'

export function getLastBoard(): string | null {
  try {
    return localStorage.getItem(LAST_BOARD_KEY)
  } catch {
    return null
  }
}

export function setLastBoard(param: string) {
  try {
    localStorage.setItem(LAST_BOARD_KEY, param)
  } catch {
    // localStorage might be blocked; the board is then not remembered
  }
}

// Remembers a board the user chose ('' is the default board, /dashboard)
export function rememberBoard(board: Board) {
  setLastBoard(board.owned && board.is_default ? '' : board.id)
}

// Where /dashboard should go instead, if anywhere: the board the user opened
// last, or, for someone who never chose and has nothing on their own board
// yet, the board that is shared with them (what a new user sees first).
export function homeRedirect(
  boards: Board[],
  lastBoard: string | null,
  ownBoardIsEmpty: boolean
): Board | undefined {
  if (lastBoard) {
    const last = findBoard(boards, lastBoard)
    return last && !(last.owned && last.is_default) ? last : undefined
  }
  if (lastBoard === null && ownBoardIsEmpty) return boards.find((b) => !b.owned)
  return undefined
}
