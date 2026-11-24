import type { ConfigFileStatus } from '@/types'

// Whether the dashboard can be edited, from what useConfigFile gives: only
// once the server said no config file makes it read-only. While that loads,
// or when it could not be loaded, edit controls stay hidden.
export function canEdit(status: ConfigFileStatus | null | undefined): boolean {
  return status === null || status?.read_only === false
}
