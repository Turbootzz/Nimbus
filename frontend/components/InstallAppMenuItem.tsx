'use client'

import { ArrowDownTrayIcon } from '@heroicons/react/24/outline'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import { promptInstall, useCanInstall } from '@/lib/pwa'

interface InstallAppMenuItemProps {
  // Called once the install dialog is requested, to close the menu
  onSelect?: () => void
}

// User menu entry that installs Nimbus as an app. Renders nothing when the
// browser offers no install (Safari, Firefox) or the app is already installed.
export default function InstallAppMenuItem({ onSelect }: InstallAppMenuItemProps) {
  const canInstall = useCanInstall()
  const menuItemHover = useHoverStyle(hoverStyles.menuItem)

  if (!canInstall) return null

  return (
    <button
      type="button"
      onClick={() => {
        onSelect?.()
        promptInstall().catch((error) => console.warn('Install prompt failed:', error))
      }}
      className="flex w-full items-center px-4 py-2 text-sm transition-colors"
      style={{ color: 'var(--color-text-primary)' }}
      {...menuItemHover}
    >
      <ArrowDownTrayIcon className="mr-3 h-5 w-5" style={{ color: 'var(--color-text-muted)' }} />
      Install app
    </button>
  )
}
