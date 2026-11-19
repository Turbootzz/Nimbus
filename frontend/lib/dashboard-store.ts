import { useSyncExternalStore } from 'react'
import type { WeatherPayload } from '@/types'

// What the dashboard page shares with the canvas top bar, which sits in the
// layout above it: edit mode (the user menu toggles it) and the weather of
// the user's weather widget (the page already follows the stream). Read with
// useSyncExternalStore, like the layout and sidebar stores; not persisted.

export interface TopBarWeather {
  location: string
  weather: WeatherPayload
}

let state: { editMode: boolean; weather: TopBarWeather | null } = {
  editMode: false,
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

export function useEditMode(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => state.editMode,
    () => false
  )
}

export function useTopBarWeather(): TopBarWeather | null {
  return useSyncExternalStore(
    subscribe,
    () => state.weather,
    () => null
  )
}
