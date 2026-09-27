'use client'

import Markdown, { type Components } from 'react-markdown'
import type { NoteWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { widgetMaxHeight } from '@/lib/card-utils'

// react-markdown never renders raw HTML and drops unsafe link protocols
// like javascript:, so user notes can't inject scripts.
function markdownComponents(openInNewTab: boolean): Components {
  return {
    h1: ({ children }) => <h1 className="mb-2 text-lg font-semibold">{children}</h1>,
    h2: ({ children }) => <h2 className="mb-2 text-base font-semibold">{children}</h2>,
    h3: ({ children }) => <h3 className="mb-1 text-sm font-semibold">{children}</h3>,
    p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
    ul: ({ children }) => <ul className="mb-2 list-disc pl-5">{children}</ul>,
    ol: ({ children }) => <ol className="mb-2 list-decimal pl-5">{children}</ol>,
    code: ({ children }) => (
      <code className="bg-background rounded px-1 py-0.5 text-xs">{children}</code>
    ),
    a: ({ href, children }) => (
      <a
        href={href}
        className="text-primary underline"
        {...(openInNewTab && { target: '_blank', rel: 'noopener noreferrer' })}
      >
        {children}
      </a>
    ),
  }
}

export default function NoteWidget({
  config,
  cardSize,
  openInNewTab,
}: WidgetRendererProps<NoteWidgetConfig>) {
  if (!config.content.trim()) {
    return <p className="text-text-muted text-sm">Empty note</p>
  }
  return (
    <div className={`text-text-primary overflow-auto text-sm ${widgetMaxHeight[cardSize]}`}>
      <Markdown components={markdownComponents(openInNewTab)}>{config.content}</Markdown>
    </div>
  )
}
