import type { CardSize } from '@/types'

// Grid span classes for each card size
// Grid uses 2/4/6/8 columns at different breakpoints
// 1x1 = 1 col, 2x1 = 2 cols (standard), 1x2 = 1 col + 2 rows (widgets only),
// 2x2 = 2 cols + 2 rows
export const sizeToGridSpan: Record<CardSize, string> = {
  '1x1': 'col-span-1 row-span-1',
  '2x1': 'col-span-2 row-span-1',
  '1x2': 'col-span-1 row-span-2',
  '2x2': 'col-span-2 row-span-2',
}

// Card size cycle order for services in edit mode
export const sizeOrder: CardSize[] = ['1x1', '2x1', '2x2']

// Widgets can also be 1x2
const widgetSizeOrder: CardSize[] = ['1x1', '2x1', '1x2', '2x2']

export function getNextSize(current: CardSize): CardSize {
  const idx = sizeOrder.indexOf(current)
  return sizeOrder[(idx + 1) % sizeOrder.length]
}

// Default card size when not specified
export const DEFAULT_CARD_SIZE: CardSize = '2x1'

// Next size in the cycle that the widget's type allows
export function getNextAllowedSize(current: CardSize, allowed: CardSize[]): CardSize {
  const start = widgetSizeOrder.indexOf(current)
  for (let step = 1; step <= widgetSizeOrder.length; step++) {
    const size = widgetSizeOrder[(start + step) % widgetSizeOrder.length]
    if (allowed.includes(size)) return size
  }
  return current
}
