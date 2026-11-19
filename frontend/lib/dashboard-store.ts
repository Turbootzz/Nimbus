import { useSyncExternalStore } from 'react'
import type { WeatherPayload } from '@/types'

// What the dashboard page shares with the canvas top bar, which sits in the
// layout above it: edit mode (the user menu toggles it), whether the board
// can be edited at all, and the weather of the board's weather widget (the
// page already follows the stream). Read with useSyncExternalStore, like the
// layout and sidebar stores; not persisted.

export interface TopBarWeather {
  location: string
  weather: WeatherPayload
}

let state: {
  editMode: boolean
  readOnly: boolean
  // The board on screen: its ID, '' for the default board, null for none
  boardOnScreen: string | null
  weather: TopBarWeather | null
} = {
  editMode: false,
  readOnly: false,
  boardOnScreen: null,
  weather: null,
}
const listeners = new Set<() => void>()

function update(change: Partial<typeof state>) {
  state = { ...state, ...change }
  listeners.forEach((l) => l())
}

function subscribe(onStoreChange: () => void) {
  listeners.add(onStoreChange)
  return () => {
    listeners.delete(onStoreChange)
  }
}

export const setEditMode = (editMode: boolean) => update({ editMode })
export const toggleEditMode = () => update({ editMode: !state.editMode })
export const setTopBarWeather = (weather: TopBarWeather | null) => update({ weather })
// A board someone else shares, or a kiosk view: nothing to edit
export const setBoardReadOnly = (readOnly: boolean) => update({ readOnly })
export const setBoardOnScreen = (boardOnScreen: string | null) => update({ boardOnScreen })

export function useEditMode(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => state.editMode,
    () => false
  )
}

export function useBoardReadOnly(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => state.readOnly,
    () => false
  )
}

export function useBoardOnScreen(): string | null {
  return useSyncExternalStore(
    subscribe,
    () => state.boardOnScreen,
    () => null
  )
}

export function useTopBarWeather(): TopBarWeather | null {
  return useSyncExternalStore(
    subscribe,
    () => state.weather,
    () => null
  )
}
