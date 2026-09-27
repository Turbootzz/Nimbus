'use client'

import type { EmbedWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'

export default function EmbedWidget({ widget, config }: WidgetRendererProps<EmbedWidgetConfig>) {
  if (!config.url) {
    return <p className="text-text-muted text-sm">No URL set</p>
  }
  // The sandbox keeps the page from navigating the dashboard itself
  return (
    <iframe
      src={config.url}
      title={widget.title || 'Embedded page'}
      className="h-full w-full rounded border-0"
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
      referrerPolicy="no-referrer"
      loading="lazy"
    />
  )
}
