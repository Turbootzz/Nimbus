'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import { useTranslations } from 'next-intl'
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
  const t = useTranslations('widgets')
  const setFeeds = (feeds: string[]) => onChange({ ...config, feeds })

  return (
    <div className="space-y-4">
      <div>
        <span className={labelClass}>{t('rss.form.feeds')}</span>
        <div className="space-y-2">
          {config.feeds.map((feed, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={t('rss.form.feedUrl', { n: index + 1 })}
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
                  aria-label={t('rss.form.removeFeed', { n: index + 1 })}
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
            {t('rss.form.addFeed')}
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">{t('rss.form.feedsHint')}</p>
      </div>

      <div>
        <label htmlFor="rss-limit" className={labelClass}>
          {t('rss.form.limit')}
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
        label={t('common.verifyTls')}
        description={t('rss.form.verifyTlsHint')}
        disabled={disabled}
      />
    </div>
  )
}
