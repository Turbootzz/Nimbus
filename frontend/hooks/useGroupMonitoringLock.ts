'use client'

import { useEffect } from 'react'
import type { Group } from '@/types'

const DEFAULT_MONITORING_DESCRIPTION =
  "When disabled, this service won't be health-checked and won't appear in metrics or trigger webhooks"

interface UseGroupMonitoringLockArgs {
  groups: Group[]
  selectedGroupId: string
  enableServiceGrouping: boolean
  monitoringEnabled: boolean
  setMonitoringEnabled: (next: boolean) => void
}

interface UseGroupMonitoringLockResult {
  selectedGroup: Group | undefined
  // True when the selected group has opted out of monitoring, in which case
  // the form's monitoring toggle must be locked off.
  groupMonitoringDisabled: boolean
  // Description to feed into the monitoring Toggle's `description` prop.
  // Switches to a context-aware message when the group has locked monitoring off.
  monitoringDescription: string
}

// Keeps the per-service "Enable Monitoring" toggle in sync with the parent
// group's monitoring flag. The backend skips health checks for any service
// inside a non-monitored group (see GetAllForMonitoring), so letting a user
// flip the per-service flag on in that situation would just produce a stuck
// "unknown" status.
export function useGroupMonitoringLock({
  groups,
  selectedGroupId,
  enableServiceGrouping,
  monitoringEnabled,
  setMonitoringEnabled,
}: UseGroupMonitoringLockArgs): UseGroupMonitoringLockResult {
  const selectedGroup = enableServiceGrouping
    ? groups.find((g) => g.id === selectedGroupId)
    : undefined
  const groupMonitoringDisabled = selectedGroup?.monitoring_enabled === false

  useEffect(() => {
    if (groupMonitoringDisabled && monitoringEnabled) {
      setMonitoringEnabled(false)
    }
  }, [groupMonitoringDisabled, monitoringEnabled, setMonitoringEnabled])

  const monitoringDescription = groupMonitoringDisabled
    ? `Monitoring is disabled for the "${selectedGroup?.name}" group, so services in it are never health-checked`
    : DEFAULT_MONITORING_DESCRIPTION

  return { selectedGroup, groupMonitoringDisabled, monitoringDescription }
}
