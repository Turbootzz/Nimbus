'use client'

import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type HTMLAttributes,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { useLocale, useTranslations } from 'next-intl'
import type { MetricsResponse, Service } from '@/types'
import { useRelativeTime } from '@/hooks/useRelativeTime'
import { loadServiceDetail } from '@/lib/service-detail'
import { getResponseTimeColor } from '@/lib/status-utils'

// The chart library is only loaded once a popover opens
const UptimeSparkline = lazy(() => import('@/components/graphs/UptimeSparkline'))

// Long enough that moving the mouse across the tiles doesn't open any
const HOVER_DELAY_MS = 350
const LONG_PRESS_MS = 500
// A tap also sends mouse events; they must not count as hovering
const TOUCH_MOUSE_GAP_MS = 1000
const WIDTH_PX = 256
const MARGIN_PX = 8
// Room the popover needs below the tile before it goes above it
const HEIGHT_PX = 190

// undefined while loading, null when the request failed
type Detail = MetricsResponse | null | undefined

function isFocusVisible(element: Element): boolean {
  try {
    return element.matches(':focus-visible')
  } catch {
    return true // browsers without the selector: any focus counts
  }
}

interface ServiceDetail {
  // Spread on the tile's link
  triggerProps: HTMLAttributes<HTMLElement>
  popover: ReactNode
}

/**
 * Detail popover of a service tile: 24h uptime with a sparkline, last check
 * and last error. Opens on hover, keyboard focus or a long press, and closes
 * on Escape, scrolling, or leaving the tile. Nothing is requested until it
 * first opens.
 */
export function useServiceDetail(service: Service, monitored: boolean, enabled = true) {
  const [anchor, setAnchor] = useState<DOMRect | null>(null)
  const [detail, setDetail] = useState<Detail>(undefined)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const lastTouch = useRef(0)
  const touching = useRef(false)
  const longPressed = useRef(false)
  const id = useId()
  const open = anchor !== null

  const show = (element: Element) => {
    setAnchor(element.getBoundingClientRect())
    if (monitored) loadServiceDetail(service.id).then(setDetail)
  }
  const hide = useCallback(() => {
    clearTimeout(timer.current)
    setAnchor(null)
  }, [])
  const showAfter = (element: Element, delay: number, onShow?: () => void) => {
    clearTimeout(timer.current)
    timer.current = setTimeout(() => {
      onShow?.()
      show(element)
    }, delay)
  }

  // Unmounting, or edit mode turning it off, cancels a popover that is on
  // its way and closes an open one
  useEffect(() => (enabled ? hide : undefined), [enabled, hide])

  useEffect(() => {
    if (!open) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') hide()
    }
    document.addEventListener('keydown', onKeyDown)
    // The popover is fixed, so it would stay behind when the page scrolls;
    // a tap elsewhere closes one that a long press opened
    window.addEventListener('scroll', hide, true)
    document.addEventListener('pointerdown', hide)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('scroll', hide, true)
      document.removeEventListener('pointerdown', hide)
    }
  }, [open, hide])

  if (!enabled) return { triggerProps: {}, popover: null } satisfies ServiceDetail

  // Lifting the finger early is not a long press; neither is a touch the
  // browser took over
  const endTouch = () => {
    lastTouch.current = Date.now()
    touching.current = false
    clearTimeout(timer.current)
  }

  const triggerProps: HTMLAttributes<HTMLElement> = {
    'aria-describedby': open ? id : undefined,
    // iOS would open its own link preview on a long press
    style: { WebkitTouchCallout: 'none' },
    onMouseEnter: (event) => {
      if (Date.now() - lastTouch.current < TOUCH_MOUSE_GAP_MS) return
      showAfter(event.currentTarget, HOVER_DELAY_MS)
    },
    onMouseLeave: hide,
    onFocus: (event) => {
      if (isFocusVisible(event.currentTarget)) show(event.currentTarget)
    },
    onBlur: hide,
    onTouchStart: (event) => {
      lastTouch.current = Date.now()
      touching.current = true
      longPressed.current = false
      showAfter(event.currentTarget, LONG_PRESS_MS, () => (longPressed.current = true))
    },
    // Scrolling is not a long press
    onTouchMove: () => clearTimeout(timer.current),
    onTouchEnd: endTouch,
    onTouchCancel: endTouch,
    // The long press opened the popover; it must not follow the link too
    onClick: (event) => {
      if (longPressed.current) event.preventDefault()
      longPressed.current = false
    },
    // Android opens its link menu on a long press, often just before the
    // popover is due; a right click keeps its menu
    onContextMenu: (event) => {
      if (touching.current || longPressed.current) event.preventDefault()
    },
  }

  const popover =
    anchor &&
    createPortal(
      <DetailPopover
        id={id}
        anchor={anchor}
        service={service}
        monitored={monitored}
        detail={detail}
      />,
      document.body
    )

  return { triggerProps, popover } satisfies ServiceDetail
}

interface DetailPopoverProps {
  id: string
  anchor: DOMRect
  service: Service
  monitored: boolean
  detail: Detail
}

function DetailPopover({ id, anchor, service, monitored, detail }: DetailPopoverProps) {
  const t = useTranslations('services')
  const locale = useLocale()
  const relativeTime = useRelativeTime()
  // Rounded as toFixed does; the locale picks the decimal mark
  const uptime = (percentage: number) =>
    Number(percentage.toFixed(1)).toLocaleString(locale, { minimumFractionDigits: 1 })
  // Under the tile, or above it near the bottom of the window, kept inside
  // the window sideways
  const left = Math.max(MARGIN_PX, Math.min(anchor.left, window.innerWidth - WIDTH_PX - MARGIN_PX))
  const below = anchor.bottom + HEIGHT_PX <= window.innerHeight || anchor.top < HEIGHT_PX
  const position = below
    ? { top: anchor.bottom + MARGIN_PX }
    : { bottom: window.innerHeight - anchor.top + MARGIN_PX }

  const online = service.status === 'online' && service.response_time != null

  return (
    <div
      id={id}
      role="tooltip"
      style={{ left, width: WIDTH_PX, ...position }}
      className="bg-card border-card-border pointer-events-none fixed z-50 rounded-lg border p-3 text-sm shadow-lg"
    >
      <p className="flex items-baseline justify-between gap-2">
        <span className="text-text-primary truncate font-semibold">{service.name}</span>
        {/* In words too: a chip only has its coloured dot */}
        {monitored && (
          <span className="text-text-muted shrink-0 text-xs">{t(`status.${service.status}`)}</span>
        )}
      </p>
      <p className="text-text-muted truncate text-xs">{service.url}</p>

      <div className="border-card-border mt-2 border-t pt-2">
        {!monitored ? (
          <p className="text-text-muted text-xs">{t('tile.notMonitored')}</p>
        ) : detail === undefined ? (
          <p className="text-text-muted text-xs">{t('hoverDetail.loading')}</p>
        ) : detail === null ? (
          <p className="text-text-muted text-xs">{t('hoverDetail.loadFailed')}</p>
        ) : (
          <dl className="space-y-1 text-xs">
            <div className="flex items-center justify-between gap-2">
              <dt className="text-text-muted">{t('hoverDetail.uptime')}</dt>
              <dd className="flex items-center gap-2">
                {detail.total_checks > 0 && (
                  <Suspense fallback={<div className="h-8 w-24" />}>
                    <UptimeSparkline dataPoints={detail.data_points} />
                  </Suspense>
                )}
                <span className="text-text-primary font-semibold tabular-nums">
                  {detail.total_checks > 0
                    ? `${uptime(detail.uptime_percentage)}%`
                    : t('hoverDetail.noChecks')}
                </span>
              </dd>
            </div>
            {online && (
              <div className="flex justify-between gap-2">
                <dt className="text-text-muted">{t('hoverDetail.responseTime')}</dt>
                <dd className={`tabular-nums ${getResponseTimeColor(service.response_time!)}`}>
                  {service.response_time}ms
                </dd>
              </div>
            )}
            <div className="flex justify-between gap-2">
              <dt className="text-text-muted">{t('hoverDetail.lastCheck')}</dt>
              <dd className="text-text-primary">
                {detail.last_checked_at
                  ? relativeTime(detail.last_checked_at)
                  : t('hoverDetail.never')}
              </dd>
            </div>
            {detail.last_error && (
              <div>
                <dt className="text-text-muted">
                  {t('hoverDetail.lastError')}
                  {detail.last_error_at && ` · ${relativeTime(detail.last_error_at)}`}
                </dt>
                <dd className="text-error line-clamp-3 break-words">{detail.last_error}</dd>
              </div>
            )}
          </dl>
        )}
      </div>
    </div>
  )
}
