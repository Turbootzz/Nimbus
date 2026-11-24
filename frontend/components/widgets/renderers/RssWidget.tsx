'use client'

import { useTranslations } from 'next-intl'
import type { RssPayload, RssWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { useRelativeTime } from '@/hooks/useRelativeTime'

export default function RssWidget({
  snapshot,
  openInNewTab,
}: WidgetRendererProps<RssWidgetConfig>) {
  const t = useTranslations('widgets.rss')
  const relativeTime = useRelativeTime()
  const feed = snapshot?.payload as RssPayload | null | undefined
  if (!feed) return null // WidgetCard shows loading and errors

  const failed = feed.failed ?? []
  return (
    <div>
      {failed.length > 0 && (
        <p className="text-warning mb-1 px-2 text-xs" title={failed.join('\n')}>
          {t('failed', { count: failed.length })}
        </p>
      )}
      {feed.items.length === 0 ? (
        <p className="text-text-muted px-2 text-sm">{t('empty')}</p>
      ) : (
        <ul className="space-y-0.5">
          {feed.items.map((item, index) => {
            const body = (
              <>
                <p className="text-text-primary line-clamp-2 text-sm">
                  {item.title || t('untitled')}
                </p>
                <p className="text-text-muted truncate text-xs">
                  {item.source}
                  {item.date && <time dateTime={item.date}> · {relativeTime(item.date)}</time>}
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
