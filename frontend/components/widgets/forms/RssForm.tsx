'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import type { RssWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { Toggle } from '@/components/ui/Toggle'
import {
  addRowClass,
  inputClass,
  labelClass,
  removeRowClass,
} from '@/components/widgets/forms/fieldStyles'

// Match the limits in the backend
const MAX_FEEDS = 3
const MAX_ITEMS = 50

export default function RssForm({ config, onChange, disabled }: WidgetFormProps<RssWidgetConfig>) {
  const setFeeds = (feeds: string[]) => onChange({ ...config, feeds })

  return (
    <div className="space-y-4">
      <div>
        <span className={labelClass}>Feeds</span>
        <div className="space-y-2">
          {config.feeds.map((feed, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={`Feed ${index + 1} URL`}
                type="url"
                value={feed}
                onChange={(e) =>
                  setFeeds(config.feeds.map((f, i) => (i === index ? e.target.value : f)))
                }
                placeholder="https://example.com/feed.xml"
                className={inputClass}
                disabled={disabled}
              />
              {config.feeds.length > 1 && (
                <button
                  type="button"
                  onClick={() => setFeeds(config.feeds.filter((_, i) => i !== index))}
                  className={removeRowClass}
                  aria-label={`Remove feed ${index + 1}`}
                  disabled={disabled}
                >
                  <TrashIcon className="h-4 w-4" />
                </button>
              )}
            </div>
          ))}
        </div>
        {config.feeds.length < MAX_FEEDS && (
          <button
            type="button"
            onClick={() => setFeeds([...config.feeds, ''])}
            className={addRowClass}
            disabled={disabled}
          >
            <PlusIcon className="mr-1 h-4 w-4" />
            Add feed
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">
          RSS, Atom and JSON Feed work. Items from all feeds are merged, newest first.
        </p>
      </div>

      <div>
        <label htmlFor="rss-limit" className={labelClass}>
          Number of items
        </label>
        <input
          id="rss-limit"
          type="number"
          min={1}
          max={MAX_ITEMS}
          value={config.limit}
          onChange={(e) => onChange({ ...config, limit: Number(e.target.value) })}
          className={`${inputClass} w-24!`}
          disabled={disabled}
        />
      </div>

      <Toggle
        id="rss-verify-tls"
        enabled={config.verify_tls}
        onChange={(verify_tls) => onChange({ ...config, verify_tls })}
        label="Verify TLS certificate"
        description="Turn off for feeds on your LAN with a self-signed certificate"
        disabled={disabled}
      />
    </div>
  )
}
