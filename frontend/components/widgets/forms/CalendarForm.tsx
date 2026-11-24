'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
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

export default function CalendarForm({
  config,
  onChange,
  disabled,
  meta,
}: WidgetFormProps<CalendarWidgetConfig>) {
  const t = useTranslations('widgets')
  const { integrations, isLoading, error } = useIntegrations()
  const kinds = meta?.config_integration_kinds ?? ['sonarr', 'radarr']
  const usable = integrations.filter((i) => kinds.includes(i.kind))
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
        <legend className={labelClass}>{t('calendar.form.sources')}</legend>
        {isLoading ? (
          <p className="text-text-muted text-sm">{t('calendar.form.loadingIntegrations')}</p>
        ) : error ? (
          <p className="text-error text-sm">{t('common.integrationsError', { error })}</p>
        ) : usable.length === 0 ? (
          <p className="text-text-muted text-sm">
            {t.rich('calendar.form.noIntegrations', {
              link: (chunks) => (
                <Link href="/settings/integrations" className="text-primary underline">
                  {chunks}
                </Link>
              ),
            })}
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
        <span className={labelClass}>{t('calendar.form.feeds')}</span>
        <div className="space-y-2">
          {config.ical_urls.map((url, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={t('calendar.form.feedUrl', { n: index + 1 })}
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
                aria-label={t('calendar.form.removeFeed', { n: index + 1 })}
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
            {t('calendar.form.addFeed')}
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">{t('calendar.form.feedsHint')}</p>
      </div>

      <div>
        <label htmlFor="calendar-days" className={labelClass}>
          {t('calendar.form.days')}
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
          label={t('common.verifyTls')}
          description={t('calendar.form.verifyTlsHint')}
          disabled={disabled}
        />
      )}
    </div>
  )
}
