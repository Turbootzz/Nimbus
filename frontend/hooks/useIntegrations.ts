'use client'

import { useCallback, useEffect, useState } from 'react'
import { api } from '@/lib/api'
import type { Integration, IntegrationKindMeta } from '@/types'

/**
 * The user's integrations and the kinds the server supports
 */
export function useIntegrations() {
  const [integrations, setIntegrations] = useState<Integration[]>([])
  const [kinds, setKinds] = useState<IntegrationKindMeta[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    try {
      const [integrationsResponse, kindsResponse] = await Promise.all([
        api.getIntegrations(),
        api.getIntegrationKinds(),
      ])
      const failed = integrationsResponse.error || kindsResponse.error
      setError(failed ? failed.message : null)
      if (integrationsResponse.data) setIntegrations(integrationsResponse.data)
      if (kindsResponse.data) setKinds(kindsResponse.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load integrations')
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    reload()
  }, [reload])

  return { integrations, setIntegrations, kinds, isLoading, error, reload }
}

// Name of a kind, falling back to its id for kinds this server doesn't know
export function kindName(kinds: IntegrationKindMeta[], kind: string): string {
  return kinds.find((k) => k.kind === kind)?.name ?? kind
}
