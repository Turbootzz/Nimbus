import { useEffect, useSyncExternalStore } from 'react'
import { api } from '@/lib/api'
import type { Board } from '@/types'

// The boards the user sees, loaded once and shared by the switcher, the
// dashboard, the modals and the command palette (a small external store,
// like the layout and dashboard stores). The server decides what is in the
// list; this only keeps it.

interface BoardsState {
  boards: Board[]
  // False until a load succeeded
  loaded: boolean
  // The last load failed; the boards are those of the load before, if any
  failed: boolean
}

let state: BoardsState = { boards: [], loaded: false, failed: false }
let loading: Promise<void> | null = null
// A refresh asked for while a load was in flight: it runs after that one
let queued: Promise<void> | null = null
const listeners = new Set<() => void>()
const serverState: BoardsState = { boards: [], loaded: false, failed: false }

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

async function load() {
  try {
    const response = await api.getBoards()
    if (!response.data) throw new Error(response.error?.message)
    set({ boards: response.data, loaded: true, failed: false })
  } catch {
    set({ ...state, failed: true })
  }
}

// Loads the boards, or joins the load in flight
function loadBoards(): Promise<void> {
  loading ??= load().finally(() => {
    loading = null
  })
  return loading
}

// Loads the boards again, e.g. after one was created. A load in flight may
// have been answered before that change, so another one follows it. Never
// throws: without a list there is one board, the default one.
export function refreshBoards(): Promise<void> {
  if (!loading) return loadBoards()
  queued ??= loading.then(() => {
    queued = null
    return loadBoards()
  })
  return queued
}

export function getBoardsSnapshot(): Board[] {
  return state.boards
}

// For tests: forget what was loaded
export function resetBoards() {
  state = { boards: [], loaded: false, failed: false }
  loading = null
  queued = null
}

export function useBoards(): BoardsState {
  const current = useSyncExternalStore(
    subscribe,
    () => state,
    () => serverState
  )
  // Until a load succeeded, every newly shown user of the list tries again
  useEffect(() => {
    if (!state.loaded) loadBoards()
  }, [])
  return current
}
