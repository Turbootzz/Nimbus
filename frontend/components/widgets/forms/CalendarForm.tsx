'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import Link from 'next/link'
import type { CalendarWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import {
  addRowClass,
  inputClass,
  labelClass,
  removeRowClass,
} from '@/components/widgets/forms/fieldStyles'
import { Toggle } from '@/components/ui/Toggle'
import { useIntegrations } from '@/hooks/useIntegrations'

// Match the limits in the backend
const MAX_INTEGRATIONS = 4
const MAX_FEEDS = 5
const CALENDAR_KINDS = ['sonarr', 'radarr']

export default function CalendarForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<CalendarWidgetConfig>) {
  const { integrations, isLoading, error } = useIntegrations()
  const usable = integrations.filter((i) => CALENDAR_KINDS.includes(i.kind))
  const setFeeds = (ical_urls: string[]) => onChange({ ...config, ical_urls })

  const toggleIntegration = (id: string, on: boolean) =>
    onChange({
      ...config,
      integrations: on
        ? [...config.integrations, id]
        : config.integrations.filter((other) => other !== id),
    })

  return (
    <div className="space-y-4">
      <fieldset>
        <legend className={labelClass}>Sonarr and Radarr</legend>
        {isLoading ? (
          <p className="text-text-muted text-sm">Loading integrations...</p>
        ) : error ? (
          <p className="text-error text-sm">Could not load your integrations: {error}</p>
        ) : usable.length === 0 ? (
          <p className="text-text-muted text-sm">
            No Sonarr or Radarr yet.{' '}
            <Link href="/settings/integrations" className="text-primary underline">
              Add one
            </Link>{' '}
            to see what is coming.
          </p>
        ) : (
          <div className="space-y-1">
            {usable.map((integration) => {
              const checked = config.integrations.includes(integration.id)
              return (
                <label
                  key={integration.id}
                  className="text-text-primary flex items-center gap-2 text-sm"
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={(e) => toggleIntegration(integration.id, e.target.checked)}
                    disabled={
                      disabled || (!checked && config.integrations.length >= MAX_INTEGRATIONS)
                    }
                    className="accent-primary"
                  />
                  {integration.name}
                </label>
              )
            })}
          </div>
        )}
      </fieldset>

      <div>
        <span className={labelClass}>Calendar feeds (iCal)</span>
        <div className="space-y-2">
          {config.ical_urls.map((url, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={`Calendar ${index + 1} URL`}
                type="url"
                value={url}
                onChange={(e) =>
                  setFeeds(config.ical_urls.map((u, i) => (i === index ? e.target.value : u)))
                }
                placeholder="https://calendar.example.com/basic.ics"
                className={inputClass}
                disabled={disabled}
              />
              <button
                type="button"
                onClick={() => setFeeds(config.ical_urls.filter((_, i) => i !== index))}
                className={removeRowClass}
                aria-label={`Remove calendar ${index + 1}`}
                disabled={disabled}
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
        {config.ical_urls.length < MAX_FEEDS && (
          <button
            type="button"
            onClick={() => setFeeds([...config.ical_urls, ''])}
            className={addRowClass}
            disabled={disabled}
          >
            <PlusIcon className="mr-1 h-4 w-4" />
            Add calendar
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">
          The secret iCal address of a Google, Outlook or Nextcloud calendar works. Repeating events
          only show on their first date.
        </p>
      </div>

      <div>
        <label htmlFor="calendar-days" className={labelClass}>
          Days ahead in the agenda
        </label>
        <input
          id="calendar-days"
          type="number"
          min={1}
          max={60}
          value={config.days}
          onChange={(e) => onChange({ ...config, days: Number(e.target.value) })}
          className={`${inputClass} w-24!`}
          disabled={disabled}
        />
      </div>

      {config.ical_urls.length > 0 && (
        <Toggle
          id="calendar-verify-tls"
          enabled={config.verify_tls}
          onChange={(verify_tls) => onChange({ ...config, verify_tls })}
          label="Verify TLS certificate"
          description="Turn off for calendars on your LAN with a self-signed certificate"
          disabled={disabled}
        />
      )}
    </div>
  )
}
