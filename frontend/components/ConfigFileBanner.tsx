'use client'

import { LockClosedIcon, DocumentTextIcon } from '@heroicons/react/24/outline'
import { useConfigFile } from '@/hooks/useConfigFile'

// Tells the owner that their dashboard follows a config file
export default function ConfigFileBanner() {
  const configFile = useConfigFile()
  if (!configFile) return null

  const Icon = configFile.read_only ? LockClosedIcon : DocumentTextIcon
  return (
    <div
      role="status"
      className="glass-card border-card-border text-text-secondary mb-4 flex items-start gap-3 rounded-lg border p-3 text-sm"
    >
      <Icon className="text-primary mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
      <p>
        <span className="text-text-primary font-medium">Managed by config file.</span>{' '}
        {configFile.read_only
          ? 'This dashboard is read-only: its groups, services, widgets and integrations are set in the file NIMBUS_CONFIG_FILE points at. Change the file to change the dashboard.'
          : 'This dashboard follows the file NIMBUS_CONFIG_FILE points at. What you change here can be overwritten when the file changes or the server restarts.'}
        {configFile.preferences &&
          ' Theme, layout and wallpaper settings come from the file too: changes to them are not saved.'}
      </p>
    </div>
  )
}
