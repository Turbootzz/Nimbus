'use client'

import type { RssPayload, RssWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { formatRelativeTime } from '@/lib/date-utils'

export default function RssWidget({
  snapshot,
  openInNewTab,
}: WidgetRendererProps<RssWidgetConfig>) {
  const feed = snapshot?.payload as RssPayload | null | undefined
  if (!feed) return null // WidgetCard shows loading and errors

  const failed = feed.failed ?? []
  return (
    <div>
      {failed.length > 0 && (
        <p className="text-warning mb-1 text-xs" title={failed.join('\n')}>
          {failed.length === 1 ? '1 feed' : `${failed.length} feeds`} could not be loaded
        </p>
      )}
      {feed.items.length === 0 ? (
        <p className="text-text-muted text-sm">No items yet</p>
      ) : (
        <ul className="-mx-2 space-y-0.5">
          {feed.items.map((item, index) => {
            const body = (
              <>
                <p className="text-text-primary line-clamp-2 text-sm">{item.title || 'Untitled'}</p>
                <p className="text-text-muted truncate text-xs">
                  {item.source}
                  {item.date && (
                    <time dateTime={item.date}> · {formatRelativeTime(item.date)}</time>
                  )}
                </p>
              </>
            )
            return (
              <li key={`${index}-${item.link ?? item.title}`}>
                {item.link ? (
                  <a
                    href={item.link}
                    className="hover:bg-card-hover block rounded px-2 py-1.5 transition-colors"
                    {...(openInNewTab && { target: '_blank', rel: 'noopener noreferrer' })}
                  >
                    {body}
                  </a>
                ) : (
                  <div className="px-2 py-1.5">{body}</div>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
