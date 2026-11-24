// A condition's message key (weather.<id>, see useWeatherLabel)
export type WeatherCondition =
  | 'clear'
  | 'partlyCloudy'
  | 'overcast'
  | 'fog'
  | 'drizzle'
  | 'rain'
  | 'snow'
  | 'thunderstorm'
  | 'unknown'

// WMO weather codes as used by Open-Meteo, grouped into what a tile shows
const conditions: {
  id: WeatherCondition
  codes: number[]
  label: string
  day: string
  night: string
}[] = [
  { id: 'clear', codes: [0], label: 'Clear', day: '☀️', night: '🌙' },
  { id: 'partlyCloudy', codes: [1, 2], label: 'Partly cloudy', day: '⛅', night: '☁️' },
  { id: 'overcast', codes: [3], label: 'Overcast', day: '☁️', night: '☁️' },
  { id: 'fog', codes: [45, 48], label: 'Fog', day: '🌫️', night: '🌫️' },
  { id: 'drizzle', codes: [51, 53, 55, 56, 57], label: 'Drizzle', day: '🌦️', night: '🌧️' },
  {
    id: 'rain',
    codes: [61, 63, 65, 66, 67, 80, 81, 82],
    label: 'Rain',
    day: '🌧️',
    night: '🌧️',
  },
  { id: 'snow', codes: [71, 73, 75, 77, 85, 86], label: 'Snow', day: '🌨️', night: '🌨️' },
  { id: 'thunderstorm', codes: [95, 96, 99], label: 'Thunderstorm', day: '⛈️', night: '⛈️' },
]

export function describeWeather(code: number, isDay = true): { label: string; icon: string } {
  const condition = conditions.find((c) => c.codes.includes(code))
  if (!condition) return { label: 'Unknown', icon: '🌡️' }
  return { label: condition.label, icon: isDay ? condition.day : condition.night }
}

// The condition of a code as a message key: clear, rain, ... or unknown
export function weatherConditionId(code: number): WeatherCondition {
  return conditions.find((c) => c.codes.includes(code))?.id ?? 'unknown'
}

// Short weekday for a YYYY-MM-DD date, without timezone shifts; in the
// given locale, else the browser's
export function weekday(date: string, locale?: string): string {
  const [year, month, day] = date.split('-').map(Number)
  return new Intl.DateTimeFormat(locale, { weekday: 'short', timeZone: 'UTC' }).format(
    new Date(Date.UTC(year, month - 1, day))
  )
}

export interface GeocodeResult {
  name: string
  latitude: number
  longitude: number
  label: string // name with region and country
}

// City search through Open-Meteo's free geocoding API
export async function searchLocations(query: string): Promise<GeocodeResult[]> {
  const url = `https://geocoding-api.open-meteo.com/v1/search?name=${encodeURIComponent(query)}&count=5&format=json`
  const response = await fetch(url)
  if (!response.ok) throw new Error(`Location search failed (${response.status})`)
  const data: {
    results?: {
      name: string
      latitude: number
      longitude: number
      admin1?: string
      country?: string
    }[]
  } = await response.json()
  return (data.results ?? []).map((r) => ({
    name: r.name,
    latitude: r.latitude,
    longitude: r.longitude,
    label: [r.name, r.admin1, r.country].filter(Boolean).join(', '),
  }))
}
