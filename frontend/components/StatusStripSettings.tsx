'use client'

import { useEffect, useMemo, useState } from 'react'
import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import { useTheme } from '@/contexts/ThemeContext'
import { useIntegrations } from '@/hooks/useIntegrations'
import { api } from '@/lib/api'
import { chipKey, chipOptions } from '@/lib/status-strip'
import { Toggle } from '@/components/ui/Toggle'
import type { StatusChip, Widget } from '@/types'

// Matches the backend limit
const MAX_CHIPS = 6

const selectClass =
  'bg-background border-card-border text-text-primary focus:ring-primary min-w-0 flex-1 rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none'

export default function StatusStripSettings() {
  const { statusStrip, setStatusStrip } = useTheme()
  const { integrations, kinds, isLoading, error } = useIntegrations()
  const [widgets, setWidgets] = useState<Widget[] | null>(null)
  const [widgetError, setWidgetError] = useState<string | null>(null)

  useEffect(() => {
    api.getWidgets().then((response) => {
      setWidgets(response.data ?? [])
      if (response.error) setWidgetError(response.error.message)
    })
  }, [])

  const byKey = useMemo(
    () => chipOptions(integrations, kinds, widgets ?? []),
    [integrations, kinds, widgets]
  )
  const options = [...byKey.values()]
  // Until everything is loaded a saved chip would look removed
  const loading = isLoading || widgets === null
  const loadError = error ?? widgetError

  const setChips = (chips: StatusChip[]) => setStatusStrip({ ...statusStrip, chips })
  const unused = options.filter(
    (o) => !statusStrip.chips.some((c) => chipKey(c) === chipKey(o.chip))
  )

  return (
    <div className="bg-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-2 text-xl font-semibold">Status Strip</h2>
      <p className="text-text-secondary mb-4 text-sm">
        A row of key numbers at the top of the dashboard, with whether all services are up.
      </p>
      <Toggle
        id="status-strip-enabled"
        enabled={statusStrip.enabled}
        onChange={(enabled) => setStatusStrip({ ...statusStrip, enabled })}
        label="Show the status strip"
      />

      {statusStrip.enabled && loading && (
        <p className="text-text-muted mt-4 text-sm">Loading your integrations and widgets...</p>
      )}
      {statusStrip.enabled && !loading && (
        <div className="mt-4 space-y-2">
          {statusStrip.chips.map((chip, index) => (
            // Keyed by position: a changed chip keeps its row (and focus)
            <div key={index} className="flex items-center gap-2">
              <select
                aria-label={`Number ${index + 1}`}
                value={chipKey(chip)}
                onChange={(e) => {
                  const next = byKey.get(e.target.value)
                  if (next) setChips(statusStrip.chips.map((c, i) => (i === index ? next.chip : c)))
                }}
                className={selectClass}
              >
                {!byKey.has(chipKey(chip)) && (
                  <option value={chipKey(chip)}>Removed integration or widget</option>
                )}
                {[byKey.get(chipKey(chip)), ...unused].flatMap((option) =>
                  option
                    ? [
                        <option key={chipKey(option.chip)} value={chipKey(option.chip)}>
                          {option.label}
                        </option>,
                      ]
                    : []
                )}
              </select>
              <button
                type="button"
                onClick={() => setChips(statusStrip.chips.filter((_, i) => i !== index))}
                className="text-text-muted hover:text-error shrink-0 p-1 transition-colors"
                aria-label={`Remove number ${index + 1}`}
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
          ))}
          {statusStrip.chips.length < MAX_CHIPS && unused.length > 0 && (
            <button
              type="button"
              onClick={() => setChips([...statusStrip.chips, unused[0].chip])}
              className="text-primary hover:text-primary-hover inline-flex items-center text-sm font-medium"
            >
              <PlusIcon className="mr-1 h-4 w-4" />
              Add number
            </button>
          )}
          <p className="text-text-muted text-xs">
            {loadError
              ? `Could not load your integrations or widgets: ${loadError}`
              : options.length === 0
                ? 'Add an integration or a custom API widget to pick numbers from.'
                : 'Integrations show numbers once a service or widget uses them.'}
          </p>
        </div>
      )}
    </div>
  )
}
