'use client'

import { useState } from 'react'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import type { WeatherWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'
import { searchLocations, type GeocodeResult } from '@/lib/weather'

export default function WeatherForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<WeatherWidgetConfig>) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<GeocodeResult[] | null>(null)
  const [searchError, setSearchError] = useState<string | null>(null)
  const [isSearching, setIsSearching] = useState(false)

  const search = async () => {
    if (!query.trim()) return
    setIsSearching(true)
    setSearchError(null)
    try {
      setResults(await searchLocations(query.trim()))
    } catch (err) {
      setSearchError(err instanceof Error ? err.message : 'Location search failed')
      setResults(null)
    } finally {
      setIsSearching(false)
    }
  }

  const pick = (result: GeocodeResult) => {
    onChange({
      ...config,
      latitude: result.latitude,
      longitude: result.longitude,
      location: result.name,
    })
    setResults(null)
    setQuery('')
  }

  const setCoordinate = (field: 'latitude' | 'longitude', value: string) =>
    onChange({ ...config, [field]: value === '' ? undefined : Number(value) })

  return (
    <div className="space-y-4">
      <div>
        <label htmlFor="weather-search" className={labelClass}>
          Find a place
        </label>
        <div className="flex gap-2">
          <input
            id="weather-search"
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              // Search instead of submitting the whole form
              if (e.key === 'Enter') {
                e.preventDefault()
                search()
              }
            }}
            placeholder="City name"
            className={inputClass}
            disabled={disabled}
          />
          <button
            type="button"
            onClick={search}
            disabled={disabled || isSearching}
            className="border-card-border text-text-primary hover:bg-card-hover shrink-0 rounded-md border px-3 text-sm"
            aria-label="Search places"
          >
            <MagnifyingGlassIcon className="h-4 w-4" />
          </button>
        </div>
        {searchError && <p className="text-error mt-1 text-xs">{searchError}</p>}
        {results && results.length === 0 && (
          <p className="text-text-muted mt-1 text-xs">No places found</p>
        )}
        {results && results.length > 0 && (
          <ul className="border-card-border mt-2 divide-y rounded-md border">
            {results.map((result) => (
              <li key={`${result.latitude},${result.longitude}`}>
                <button
                  type="button"
                  onClick={() => pick(result)}
                  className="text-text-primary hover:bg-card-hover w-full px-3 py-2 text-left text-sm"
                >
                  {result.label}
                </button>
              </li>
            ))}
          </ul>
        )}
        <p className="text-text-muted mt-1 text-xs">
          The search goes to Open-Meteo from your browser. You can also enter coordinates below.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <label htmlFor="weather-latitude" className={labelClass}>
            Latitude
          </label>
          <input
            id="weather-latitude"
            type="number"
            step="any"
            min={-90}
            max={90}
            value={config.latitude ?? ''}
            onChange={(e) => setCoordinate('latitude', e.target.value)}
            className={inputClass}
            disabled={disabled}
          />
        </div>
        <div>
          <label htmlFor="weather-longitude" className={labelClass}>
            Longitude
          </label>
          <input
            id="weather-longitude"
            type="number"
            step="any"
            min={-180}
            max={180}
            value={config.longitude ?? ''}
            onChange={(e) => setCoordinate('longitude', e.target.value)}
            className={inputClass}
            disabled={disabled}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <label htmlFor="weather-location" className={labelClass}>
            Name on the tile
          </label>
          <input
            id="weather-location"
            type="text"
            value={config.location}
            onChange={(e) => onChange({ ...config, location: e.target.value })}
            maxLength={100}
            className={inputClass}
            disabled={disabled}
          />
        </div>
        <div>
          <label htmlFor="weather-units" className={labelClass}>
            Units
          </label>
          <select
            id="weather-units"
            value={config.units}
            onChange={(e) =>
              onChange({ ...config, units: e.target.value as WeatherWidgetConfig['units'] })
            }
            className={inputClass}
            disabled={disabled}
          >
            <option value="metric">°C, km/h</option>
            <option value="imperial">°F, mph</option>
          </select>
        </div>
      </div>
    </div>
  )
}
