'use client'

import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import { openCommandPalette } from '@/components/CommandPalette'

// Opens the command palette; the only way in without a keyboard
export default function SearchButton() {
  return (
    <button
      type="button"
      onClick={openCommandPalette}
      className="text-text-secondary hover:text-text-primary hover:bg-card-border flex items-center gap-2 rounded-md p-2 text-sm transition-colors"
      aria-label="Search services"
      title="Search services (/ or Ctrl+K)"
    >
      <MagnifyingGlassIcon className="h-5 w-5" />
      <kbd className="border-card-border text-text-muted hidden rounded border px-1.5 text-xs lg:inline">
        /
      </kbd>
    </button>
  )
}
