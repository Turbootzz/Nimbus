'use client'

import Link from 'next/link'
import type { Integration, IntegrationKindMeta } from '@/types'
import { kindName } from '@/hooks/useIntegrations'

interface IntegrationSelectorProps {
  value: string
  onChange: (value: string) => void
  integrations: Integration[]
  kinds: IntegrationKindMeta[]
  isLoading?: boolean
  disabled?: boolean
  label?: string
  description?: string
  emptyOption?: string
}

// Links a service or widget to an integration, whose data then shows on the tile
export default function IntegrationSelector({
  value,
  onChange,
  integrations,
  kinds,
  isLoading = false,
  disabled = false,
  label = 'Live numbers',
  description = "Show numbers from an app, like Sonarr's queue, on this tile.",
  emptyOption = 'None',
}: IntegrationSelectorProps) {
  return (
    <div>
      <label
        htmlFor="integration_id"
        className="text-text-secondary mb-2 block text-sm font-medium"
      >
        {label}
      </label>
      <select
        id="integration_id"
        name="integration_id"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="border-card-border focus:border-primary w-full rounded-md border px-4 py-2 transition focus:ring-2 focus:outline-none"
        style={{ backgroundColor: 'var(--color-background)', color: 'var(--color-text-primary)' }}
        disabled={disabled || isLoading}
      >
        <option value="">{isLoading ? 'Loading integrations...' : emptyOption}</option>
        {integrations.map((integration) => (
          <option key={integration.id} value={integration.id}>
            {integration.name} ({kindName(kinds, integration.kind)})
          </option>
        ))}
      </select>
      <p className="text-text-muted mt-1 text-xs">
        {description}{' '}
        <Link href="/settings/integrations" className="text-primary underline">
          Manage integrations
        </Link>
      </p>
    </div>
  )
}
