'use client'

import {
  ArrowPathIcon,
  ExclamationTriangleIcon,
  PencilIcon,
  TrashIcon,
} from '@heroicons/react/24/outline'
import type { CardScale, CardSize, Snapshot, Widget, WidgetTypeMeta } from '@/types'
import { getNextAllowedSize, sizeToGridSpan } from '@/lib/card-utils'
import EditOverlay from '@/components/EditOverlay'
import { getWidgetDefinition, widgetConfig } from '@/components/widgets/registry'

interface WidgetCardProps {
  widget: Widget
  // From GET /widgets/types; limits the sizes a click cycles through
  meta?: WidgetTypeMeta
  // Latest data, for widgets the backend polls
  snapshot?: Snapshot
  openInNewTab: boolean
  cardScale?: CardScale
  enableCardResizing?: boolean
  isEditMode?: boolean
  isDragging?: boolean
  dragHandleProps?: Record<string, unknown>
  onSizeChange?: (widget: Widget, size: CardSize) => void
  onEdit?: (widget: Widget) => void
  onDelete?: (widget: Widget) => void
  onRefresh?: (widget: Widget) => void
}

const actionClass =
  'bg-card/90 text-text-muted hover:text-text-primary rounded p-1 transition-colors'

export default function WidgetCard({
  widget,
  meta,
  snapshot,
  openInNewTab,
  cardScale = 'medium',
  enableCardResizing = true,
  isEditMode = false,
  isDragging = false,
  dragHandleProps,
  onSizeChange,
  onEdit,
  onDelete,
  onRefresh,
}: WidgetCardProps) {
  const definition = getWidgetDefinition(widget.type)
  const allowedSizes = meta?.allowed_sizes ?? [widget.card_size]
  const canResize = enableCardResizing && allowedSizes.length > 1
  // In edit mode the SortableTile wrapper does the grid spanning
  const gridSpan = isEditMode ? '' : sizeToGridSpan[widget.card_size]
  const padding = cardScale === 'small' ? 'p-3' : 'p-4'

  const editClasses = isEditMode
    ? `border-dashed border-2 hover:border-primary ${canResize ? 'cursor-pointer' : ''}`
    : ''
  const dragClasses = isDragging ? 'scale-105 shadow-2xl ring-2 ring-primary' : ''

  const handleClick = () => {
    if (isEditMode && canResize && onSizeChange) {
      onSizeChange(widget, getNextAllowedSize(widget.card_size, allowedSizes))
    }
  }

  const polled = definition?.polled ?? false
  const hasPayload = snapshot?.payload != null
  // Old data is shown with a warning; without data the error replaces it
  const warning =
    polled && hasPayload && (snapshot?.error || snapshot?.stale)
      ? snapshot?.error
        ? `Last update failed: ${snapshot.error}`
        : 'This data may be out of date'
      : null

  const actions = (
    <>
      {onRefresh && polled && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onRefresh(widget)
          }}
          className={actionClass}
          aria-label="Refresh widget"
        >
          <ArrowPathIcon className="h-4 w-4" />
        </button>
      )}
      {onEdit && definition && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onEdit(widget)
          }}
          className={actionClass}
          aria-label="Edit widget"
        >
          <PencilIcon className="h-4 w-4" />
        </button>
      )}
      {onDelete && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onDelete(widget)
          }}
          className={`${actionClass} hover:text-error`}
          aria-label="Delete widget"
        >
          <TrashIcon className="h-4 w-4" />
        </button>
      )}
    </>
  )

  return (
    <div
      onClick={handleClick}
      className={`${gridSpan} bg-card border-card-border relative flex h-full flex-col rounded-lg border ${padding} transition-all ${editClasses} ${dragClasses}`}
    >
      {warning && !isEditMode && (
        <span className="text-warning absolute top-2 right-2" title={warning}>
          <ExclamationTriangleIcon className="h-4 w-4" aria-label={warning} />
        </span>
      )}
      {isEditMode && (
        <EditOverlay
          dragHandleProps={dragHandleProps}
          cardSize={widget.card_size}
          showSizeBadge={canResize}
          actions={actions}
        />
      )}
      {/* Edit mode leaves room for the overlay, and the content can't be
          clicked, so iframes don't swallow drags */}
      <div
        className={`flex min-h-0 flex-1 flex-col ${isEditMode ? 'pointer-events-none mt-7' : ''}`}
      >
        {widget.title && (
          <h3 className="text-text-muted mb-2 truncate text-xs font-semibold tracking-wide uppercase">
            {widget.title}
          </h3>
        )}
        {/* The content fills the tile but never sizes it: the grid row decides the
            height, and long content scrolls inside */}
        <div className="relative min-h-0 flex-1">
          <div className="absolute inset-0 overflow-auto">
            {definition && polled && !hasPayload ? (
              snapshot?.error ? (
                <p className="text-error text-sm">Could not load: {snapshot.error}</p>
              ) : (
                <div className="bg-background h-full min-h-12 animate-pulse rounded" role="status">
                  <span className="sr-only">Loading</span>
                </div>
              )
            ) : definition ? (
              <definition.Renderer
                widget={widget}
                config={widgetConfig(definition, widget)}
                cardSize={widget.card_size}
                openInNewTab={openInNewTab}
                snapshot={snapshot}
              />
            ) : (
              <p className="text-text-muted text-sm">
                This widget type ({widget.type}) is not supported by this version of Nimbus.
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
