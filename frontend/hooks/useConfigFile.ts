import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import type { ConfigFileStatus } from '@/types'

// As often as the server checks the config file for changes
const REFRESH_MS = 60_000

// Tells whether the user's dashboard comes from a config file
// (NIMBUS_CONFIG_FILE) and what that locks: null when it doesn't, undefined
// until the server has said so (see canEdit in lib/config-file). The file can
// change while the page is open, so the status is asked again every minute
// and when the tab is shown again; a request that fails keeps what was known.
// The server enforces the lock, this only lets the UI explain it.
export function useConfigFile(): ConfigFileStatus | null | undefined {
  const [status, setStatus] = useState<ConfigFileStatus | null | undefined>(undefined)

  useEffect(() => {
    let cancelled = false
    const load = () => {
      api
        .getCurrentUser()
        .then((response) => {
          if (!cancelled && response.data) setStatus(response.data.config_file ?? null)
        })
        .catch(() => {})
    }
    const refresh = () => {
      if (document.visibilityState === 'visible') load()
    }
    load()
    const timer = setInterval(refresh, REFRESH_MS)
    document.addEventListener('visibilitychange', refresh)
    return () => {
      cancelled = true
      clearInterval(timer)
      document.removeEventListener('visibilitychange', refresh)
    }
  }, [])

  return status
}
