'use client'

import Image from 'next/image'
import type { Bookmark, BookmarksWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { widgetMaxHeight } from '@/lib/card-utils'

function BookmarkIcon({ bookmark }: { bookmark: Bookmark }) {
  if (bookmark.icon.startsWith('http://') || bookmark.icon.startsWith('https://')) {
    return (
      <Image
        src={bookmark.icon}
        alt=""
        width={16}
        height={16}
        className="h-4 w-4 shrink-0 object-contain"
        unoptimized
      />
    )
  }
  if (bookmark.icon) {
    return <span className="w-4 shrink-0 text-center text-sm leading-none">{bookmark.icon}</span>
  }
  return (
    <span className="bg-primary/15 text-primary flex h-4 w-4 shrink-0 items-center justify-center rounded text-[10px] font-semibold uppercase">
      {bookmark.name.charAt(0)}
    </span>
  )
}

export default function BookmarksWidget({
  config,
  cardSize,
  openInNewTab,
}: WidgetRendererProps<BookmarksWidgetConfig>) {
  if (config.items.length === 0) {
    return <p className="text-text-muted text-sm">No bookmarks yet</p>
  }
  return (
    <ul className={`-mx-2 space-y-0.5 overflow-auto ${widgetMaxHeight[cardSize]}`}>
      {config.items.map((bookmark, index) => (
        <li key={`${index}-${bookmark.url}`}>
          <a
            href={bookmark.url}
            className="text-text-primary hover:bg-card-hover flex items-center gap-2 rounded px-2 py-1.5 text-sm transition-colors"
            {...(openInNewTab && { target: '_blank', rel: 'noopener noreferrer' })}
          >
            <BookmarkIcon bookmark={bookmark} />
            <span className="truncate">{bookmark.name}</span>
          </a>
        </li>
      ))}
    </ul>
  )
}
