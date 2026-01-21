'use client'

import { useCallback } from 'react'

type StyleProperty = 'backgroundColor' | 'color' | 'borderColor'

interface HoverStyleConfig {
  property: StyleProperty
  base: string
  hover: string
}

interface UseHoverStyleOptions {
  disabled?: boolean
}

/**
 * Creates onMouseEnter/onMouseLeave handlers for CSS variable-based hover effects.
 *
 * @example
 * // Icon button with background and color change
 * const iconHover = useHoverStyle([
 *   { property: 'backgroundColor', base: 'transparent', hover: 'var(--color-card-border)' },
 *   { property: 'color', base: 'var(--color-text-secondary)', hover: 'var(--color-text-primary)' },
 * ])
 * <button {...iconHover}>...</button>
 *
 * @example
 * // Primary button (disabled when loading)
 * const buttonHover = useHoverStyle(
 *   [{ property: 'backgroundColor', base: 'var(--color-primary)', hover: 'var(--color-primary-hover)' }],
 *   { disabled: isLoading }
 * )
 */
export function useHoverStyle(configs: HoverStyleConfig[], options: UseHoverStyleOptions = {}) {
  const { disabled = false } = options

  const onMouseEnter = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      if (disabled) return
      configs.forEach(({ property, hover }) => {
        e.currentTarget.style[property] = hover
      })
    },
    [configs, disabled]
  )

  const onMouseLeave = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      configs.forEach(({ property, base }) => {
        e.currentTarget.style[property] = base
      })
    },
    [configs]
  )

  return { onMouseEnter, onMouseLeave }
}

// Pre-configured hover styles for common patterns
export const hoverStyles = {
  /** Icon button: transparent bg → card-border, secondary text → primary text */
  iconButton: [
    {
      property: 'backgroundColor' as const,
      base: 'transparent',
      hover: 'var(--color-card-border)',
    },
    {
      property: 'color' as const,
      base: 'var(--color-text-secondary)',
      hover: 'var(--color-text-primary)',
    },
  ],
  /** Menu item: transparent bg → card-border */
  menuItem: [
    {
      property: 'backgroundColor' as const,
      base: 'transparent',
      hover: 'var(--color-card-border)',
    },
  ],
  /** Primary button: primary bg → primary-hover bg */
  primaryButton: [
    {
      property: 'backgroundColor' as const,
      base: 'var(--color-primary)',
      hover: 'var(--color-primary-hover)',
    },
  ],
  /** Link: primary color → primary-hover color */
  primaryLink: [
    {
      property: 'color' as const,
      base: 'var(--color-primary)',
      hover: 'var(--color-primary-hover)',
    },
  ],
}
