'use client'

import type { ReactNode } from 'react'
import { Bars3Icon } from '@heroicons/react/24/solid'
import type { CardSize } from '@/types'

interface EditOverlayProps {
  dragHandleProps?: Record<string, unknown>
  cardSize: CardSize
  showSizeBadge: boolean
  compact?: boolean
  // Extra buttons shown next to the size badge (e.g. edit, delete)
  actions?: ReactNode
}

// Edit mode overlay for dashboard tiles: drag handle, actions and size badge
export default function EditOverlay({
  dragHandleProps,
  cardSize,
  showSizeBadge,
  compact = false,
  actions,
}: EditOverlayProps) {
  const position = compact ? 'top-1 right-1 left-1' : 'top-2 right-2 left-2'
  const iconSize = compact ? 'h-4 w-4' : 'h-5 w-5'

  return (
    <div className={`absolute ${position} z-10 flex items-center justify-between`}>
      <div
        {...dragHandleProps}
        className="bg-card/90 cursor-grab touch-none rounded p-1 active:cursor-grabbing"
        onClick={(e) => e.stopPropagation()}
        onTouchStart={(e) => e.stopPropagation()}
      >
        <Bars3Icon className={`text-text-muted ${iconSize}`} />
      </div>
      <div className="flex items-center gap-1">
        {actions}
        {showSizeBadge && (
          <span className="bg-primary rounded px-1.5 py-0.5 text-xs text-white">{cardSize}</span>
        )}
      </div>
    </div>
  )
}
