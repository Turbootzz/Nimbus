'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { bestScore } from '@/lib/fuzzy'
import { isValidUrl } from '@/lib/utils/url'
import { useTheme } from '@/contexts/ThemeContext'
import ServiceIcon from '@/components/ServiceIcon'
import type { Group, Service } from '@/types'

const OPEN_EVENT = 'nimbus:open-palette'
const MAX_RESULTS = 8

// openCommandPalette opens the palette from a button (e.g. on mobile)
export function openCommandPalette() {
  window.dispatchEvent(new Event(OPEN_EVENT))
}

// isTyping is true when a key press belongs to a text field
function isTyping(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  return !!el && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName))
}

// isShortcut is Cmd+K on a Mac and Ctrl+K elsewhere (where Ctrl+K on a Mac
// edits text). By key position, so other keyboard layouts work too.
function isShortcut(e: KeyboardEvent): boolean {
  const mac = /Mac|iPhone|iPad/.test(navigator.platform)
  return e.code === 'KeyK' && (mac ? e.metaKey : e.ctrlKey)
}

// Search over the user's services: "/" or Cmd/Ctrl+K opens it, arrows pick,
// Enter opens the service, Esc closes
export default function CommandPalette() {
  const { openInNewTab } = useTheme()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)
  const [services, setServices] = useState<Service[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)
  const openerRef = useRef<HTMLElement | null>(null)
  const requestRef = useRef(0)

  const show = useCallback(async () => {
    if (inputRef.current) {
      inputRef.current.focus() // already open: keep what was typed
      return
    }
    openerRef.current = document.activeElement as HTMLElement | null
    setOpen(true)
    setQuery('')
    setActive(0)
    setError(null)
    setLoading(true)
    // Fresh on every open: services change while the page is open. Only
    // the latest request counts.
    const request = ++requestRef.current
    const [servicesResponse, groupsResponse] = await Promise.all([
      api.getServices(),
      api.getGroups(),
    ])
    if (request !== requestRef.current) return
    if (servicesResponse.data) setServices(servicesResponse.data)
    if (groupsResponse.data) setGroups(groupsResponse.data)
    setError(servicesResponse.error?.message ?? null)
    setLoading(false)
  }, [])

  const close = useCallback(() => {
    setOpen(false)
    requestRef.current++
    openerRef.current?.focus()
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const slash = e.key === '/' && !isTyping(e.target) && !e.metaKey && !e.ctrlKey
      if (slash || isShortcut(e)) {
        e.preventDefault()
        show()
      }
    }
    window.addEventListener('keydown', onKey)
    window.addEventListener(OPEN_EVENT, show)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener(OPEN_EVENT, show)
    }
  }, [show])

  const results = useMemo(() => {
    const groupName = new Map(groups.map((g) => [g.id, g.name]))
    return services
      .filter((service) => isValidUrl(service.url)) // never navigate to javascript: and the like
      .map((service) => {
        const group = service.group_id ? groupName.get(service.group_id) : undefined
        const score = bestScore(query, [
          { text: service.name, weight: 3 },
          { text: service.description, weight: 1.5 },
          { text: group, weight: 1.5 },
          { text: service.url, weight: 1 },
        ])
        return { service, group, score }
      })
      .filter((r): r is typeof r & { score: number } => r.score !== null)
      .sort((a, b) => b.score - a.score || a.service.name.localeCompare(b.service.name))
      .slice(0, MAX_RESULTS)
  }, [services, groups, query])

  // The highlighted row stays within the results and in view
  const current = Math.min(active, results.length - 1)
  useEffect(() => {
    listRef.current?.children[current]?.scrollIntoView?.({ block: 'nearest' })
  }, [current])

  if (!open) return null

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActive(Math.min(current + 1, results.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive(Math.max(current - 1, 0))
    } else if (e.key === 'Enter' && current >= 0) {
      e.preventDefault()
      listRef.current?.querySelectorAll('a')[current]?.click()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation() // a modal underneath must stay open
      close()
    } else if (e.key === 'Tab') {
      e.preventDefault() // focus stays in the palette
      inputRef.current?.focus()
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center p-4 pt-[15vh]">
      <div className="absolute inset-0 bg-black/50" onClick={close} aria-hidden="true" />
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Search services"
        onKeyDown={onKeyDown}
        className="bg-card border-card-border relative z-10 w-full max-w-lg overflow-hidden rounded-lg border shadow-xl"
      >
        <div className="border-card-border flex items-center gap-2 border-b px-4">
          <MagnifyingGlassIcon className="text-text-muted h-5 w-5 shrink-0" />
          <input
            ref={inputRef}
            autoFocus
            role="combobox"
            aria-expanded="true"
            aria-controls="palette-results"
            aria-activedescendant={
              current >= 0 ? `palette-${results[current].service.id}` : undefined
            }
            aria-label="Search services"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setActive(0)
            }}
            placeholder="Search services"
            className="text-text-primary placeholder:text-text-muted h-12 w-full bg-transparent text-sm focus:outline-none"
          />
          <kbd className="text-text-muted border-card-border hidden rounded border px-1.5 text-xs sm:inline">
            Esc
          </kbd>
        </div>
        <ul
          id="palette-results"
          ref={listRef}
          role="listbox"
          className="max-h-96 overflow-y-auto p-1"
        >
          {results.map(({ service, group }, index) => (
            <li
              key={service.id}
              id={`palette-${service.id}`}
              role="option"
              aria-selected={index === current}
            >
              {/* A real link: middle-click, copy link, and React's URL checks */}
              <a
                href={service.url}
                tabIndex={-1}
                onMouseEnter={() => setActive(index)}
                onClick={close}
                {...(openInNewTab && { target: '_blank', rel: 'noopener noreferrer' })}
                className={`flex items-center gap-3 rounded px-3 py-2 ${index === current ? 'bg-primary/10' : ''}`}
              >
                <span className="h-8 w-8 shrink-0 overflow-hidden" aria-hidden="true">
                  <span className="block h-12 w-12 origin-top-left scale-[0.667]">
                    <ServiceIcon service={service} size="sm" />
                  </span>
                </span>
                <span className="min-w-0 flex-1">
                  <span className="text-text-primary block truncate text-sm font-medium">
                    {service.name}
                  </span>
                  <span className="text-text-muted block truncate text-xs">
                    {group ? `${group} · ` : ''}
                    {service.url}
                  </span>
                </span>
              </a>
            </li>
          ))}
        </ul>
        {(loading || error || results.length === 0) && (
          <p className="text-text-muted px-4 py-3 text-sm" role="status">
            {loading
              ? 'Loading services...'
              : error
                ? `Could not load your services: ${error}`
                : 'No services match'}
          </p>
        )}
      </div>
    </div>
  )
}
