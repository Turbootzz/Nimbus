import type { Group, Service } from '@/types'

export type GroupMonitoringMap = Map<string, boolean>

export function buildGroupMonitoringMap(groups: Group[] | undefined | null): GroupMonitoringMap {
  const map: GroupMonitoringMap = new Map()
  if (!groups) return map
  for (const group of groups) {
    map.set(group.id, group.monitoring_enabled)
  }
  return map
}

// A service is effectively monitored only when:
// - its own monitoring_enabled flag is true, AND
// - it either has no group, or its group's monitoring_enabled is true.
//
// The backend health-checker enforces the same rule (see GetAllForMonitoring),
// so a service with monitoring_enabled=true inside a non-monitored group sits
// at status="unknown" forever — surface that as "not monitored" in the UI.
//
// If the service references a group_id not present in the map (e.g. groups
// haven't loaded yet, or a race where the service arrived first), we default
// to "monitored". The status will correct itself once the groups list is in.
export function isServiceEffectivelyMonitored(
  service: Pick<Service, 'monitoring_enabled' | 'group_id'>,
  groupMonitoringMap?: GroupMonitoringMap | null
): boolean {
  if (!service.monitoring_enabled) return false
  if (!service.group_id) return true
  if (!groupMonitoringMap) return true
  return groupMonitoringMap.get(service.group_id) !== false
}
