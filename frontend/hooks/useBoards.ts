import { useEffect, useSyncExternalStore } from 'react'
import { api } from '@/lib/api'
import type { Board } from '@/types'

// The boards the user sees, loaded once and shared by the switcher, the
// dashboard, the modals and the command palette (a small external store,
// like the layout and dashboard stores). The server decides what is in the
// list; this only keeps it.

interface BoardsState {
  boards: Board[]
  // False until the first load ended, with or without boards
  loaded: boolean
}

let state: BoardsState = { boards: [], loaded: false }
let loading: Promise<void> | null = null
const listeners = new Set<() => void>()
const serverState: BoardsState = { boards: [], loaded: false }

function subscribe(onStoreChange: () => void) {
  listeners.add(onStoreChange)
  return () => {
    listeners.delete(onStoreChange)
  }
}

function set(next: BoardsState) {
  state = next
  listeners.forEach((l) => l())
}

// Loads the boards again, e.g. after one was created. Never throws: without
// a list there is one board, the default one.
export function refreshBoards(): Promise<void> {
  loading ??= (async () => {
    try {
      const response = await api.getBoards()
      set({ boards: response.data ?? state.boards, loaded: true })
    } catch {
      set({ ...state, loaded: true })
    } finally {
      loading = null
    }
  })()
  return loading
}

export function getBoardsSnapshot(): Board[] {
  return state.boards
}

// For tests: forget what was loaded
export function resetBoards() {
  state = { boards: [], loaded: false }
  loading = null
}

export function useBoards(): BoardsState {
  const current = useSyncExternalStore(
    subscribe,
    () => state,
    () => serverState
  )
  useEffect(() => {
    if (!state.loaded) refreshBoards()
  }, [])
  return current
}
