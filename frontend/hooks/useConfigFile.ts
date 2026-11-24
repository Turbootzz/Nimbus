import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import type { ConfigFileStatus } from '@/types'

// Tells whether the user's dashboard comes from a config file
// (NIMBUS_CONFIG_FILE) and what that locks; null when it doesn't. The server
// enforces the lock, this only lets the UI explain it.
export function useConfigFile(): ConfigFileStatus | null {
  const [status, setStatus] = useState<ConfigFileStatus | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .getCurrentUser()
      .then((response) => {
        if (!cancelled) setStatus(response.data?.config_file ?? null)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  return status
}
