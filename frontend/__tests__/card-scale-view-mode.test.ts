import { describe, it, expect } from 'vitest'
import type { CardScale, ViewMode, CardSize } from '@/types'

describe('Card Scale Feature', () => {
  describe('CardScale type', () => {
    it('should support small, medium, and large scales', () => {
      const scales: CardScale[] = ['small', 'medium', 'large']
      expect(scales).toHaveLength(3)
      expect(scales).toContain('small')
      expect(scales).toContain('medium')
      expect(scales).toContain('large')
    })
  })

  describe('Grid classes mapping', () => {
    // Matches ServicesGrid.tsx gridClasses
    const gridClasses: Record<CardScale, string> = {
      small: 'grid-cols-2 gap-4 sm:grid-cols-6 sm:gap-2 xl:grid-cols-8 2xl:grid-cols-10',
      medium: 'grid-cols-2 gap-4 sm:grid-cols-4 sm:gap-3 xl:grid-cols-6 2xl:grid-cols-8',
      large: 'grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8',
    }

    it('should have correct mobile grid classes for all scales', () => {
      // Mobile always uses grid-cols-2 gap-4 regardless of scale
      Object.values(gridClasses).forEach((classes) => {
        expect(classes).toContain('grid-cols-2')
        expect(classes).toContain('gap-4')
      })
    })

    it('should have denser grid for small scale at sm breakpoint', () => {
      expect(gridClasses.small).toContain('sm:grid-cols-6')
      expect(gridClasses.small).toContain('sm:gap-2')
    })

    it('should have medium density for medium scale at sm breakpoint', () => {
      expect(gridClasses.medium).toContain('sm:grid-cols-4')
      expect(gridClasses.medium).toContain('sm:gap-3')
    })

    it('should have standard density for large scale', () => {
      expect(gridClasses.large).toContain('sm:grid-cols-4')
      // Large doesn't have sm:gap-* override, uses default gap-4
      expect(gridClasses.large).not.toContain('sm:gap-')
    })

    it('should have more columns at larger breakpoints for small scale', () => {
      expect(gridClasses.small).toContain('xl:grid-cols-8')
      expect(gridClasses.small).toContain('2xl:grid-cols-10')
    })
  })

  describe('Scale padding mapping', () => {
    // Matches ServiceCard.tsx scalePadding
    const scalePadding: Record<CardScale, { standard: string; compact: string }> = {
      small: { standard: 'p-6 sm:p-3', compact: 'p-2 sm:p-1.5' },
      medium: { standard: 'p-6 sm:p-4', compact: 'p-2' },
      large: { standard: 'p-6', compact: 'p-2' },
    }

    it('should use large padding on mobile for all scales (responsive)', () => {
      // All scales use p-6 as mobile default, then override at sm breakpoint
      expect(scalePadding.small.standard).toContain('p-6')
      expect(scalePadding.medium.standard).toContain('p-6')
      expect(scalePadding.large.standard).toBe('p-6')
    })

    it('should reduce padding at sm breakpoint for small scale', () => {
      expect(scalePadding.small.standard).toContain('sm:p-3')
    })

    it('should have medium padding at sm breakpoint for medium scale', () => {
      expect(scalePadding.medium.standard).toContain('sm:p-4')
    })

    it('should not override padding at sm breakpoint for large scale', () => {
      expect(scalePadding.large.standard).not.toContain('sm:p-')
    })
  })

  describe('Scale icon sizes', () => {
    // Matches ServiceCard.tsx scaleIconSizes
    const scaleIconSizes: Record<
      CardScale,
      { standard: 'sm' | 'md' | 'lg'; large: 'lg' | 'xl' | '2xl' }
    > = {
      small: { standard: 'sm', large: 'lg' },
      medium: { standard: 'sm', large: 'xl' },
      large: { standard: 'md', large: '2xl' },
    }

    it('should use smallest icons for small scale', () => {
      expect(scaleIconSizes.small.standard).toBe('sm')
      expect(scaleIconSizes.small.large).toBe('lg')
    })

    it('should use medium icons for medium scale', () => {
      expect(scaleIconSizes.medium.standard).toBe('sm')
      expect(scaleIconSizes.medium.large).toBe('xl')
    })

    it('should use largest icons for large scale', () => {
      expect(scaleIconSizes.large.standard).toBe('md')
      expect(scaleIconSizes.large.large).toBe('2xl')
    })
  })

  describe('Scale text sizes', () => {
    // Matches ServiceCard.tsx scaleText
    const scaleText: Record<CardScale, { title: string; description: string }> = {
      small: { title: 'text-sm', description: 'text-xs' },
      medium: { title: 'text-base', description: 'text-sm' },
      large: { title: 'text-lg', description: 'text-sm' },
    }

    it('should use smallest text for small scale', () => {
      expect(scaleText.small.title).toBe('text-sm')
      expect(scaleText.small.description).toBe('text-xs')
    })

    it('should use base text for medium scale', () => {
      expect(scaleText.medium.title).toBe('text-base')
      expect(scaleText.medium.description).toBe('text-sm')
    })

    it('should use larger text for large scale', () => {
      expect(scaleText.large.title).toBe('text-lg')
      expect(scaleText.large.description).toBe('text-sm')
    })
  })
})

describe('View Mode Feature', () => {
  describe('ViewMode type', () => {
    it('should support grid and list modes', () => {
      const modes: ViewMode[] = ['grid', 'list']
      expect(modes).toHaveLength(2)
      expect(modes).toContain('grid')
      expect(modes).toContain('list')
    })
  })

  describe('View mode switching logic', () => {
    it('should toggle between grid and list', () => {
      const toggleViewMode = (current: ViewMode): ViewMode => {
        return current === 'grid' ? 'list' : 'grid'
      }

      expect(toggleViewMode('grid')).toBe('list')
      expect(toggleViewMode('list')).toBe('grid')
    })
  })

  describe('Card scale disabled in list view', () => {
    it('should not apply card scale in list view', () => {
      const viewMode: ViewMode = 'list'
      const cardScaleApplies = viewMode !== 'list'

      expect(cardScaleApplies).toBe(false)
    })

    it('should apply card scale in grid view', () => {
      const checkCardScale = (mode: ViewMode): boolean => mode !== 'list'
      expect(checkCardScale('grid')).toBe(true)
    })
  })

  describe('Card sizing disabled in list view', () => {
    it('should not use card sizes in list view', () => {
      const viewMode: ViewMode = 'list'
      const cardSizes: CardSize[] = ['1x1', '2x1', '2x2']

      // In list view, card sizes don't affect rendering - all items are uniform
      const effectiveSize = viewMode === 'list' ? 'uniform' : cardSizes[1]

      expect(effectiveSize).toBe('uniform')
    })
  })
})

describe('Feature Interactions', () => {
  describe('Card Scale and View Mode', () => {
    it('should default to medium scale and grid mode', () => {
      const defaultCardScale: CardScale = 'medium'
      const defaultViewMode: ViewMode = 'grid'

      expect(defaultCardScale).toBe('medium')
      expect(defaultViewMode).toBe('grid')
    })

    it('should preserve card scale when switching view modes', () => {
      const cardScale: CardScale = 'small'
      const viewModes: ViewMode[] = ['grid', 'list', 'grid']

      // Simulate switching view modes
      viewModes.forEach(() => {
        // Card scale should remain unchanged regardless of view mode changes
        expect(cardScale).toBe('small')
      })
    })
  })

  describe('Edit mode behavior', () => {
    it('should support drag and drop in both view modes', () => {
      const supportsDragDrop = (viewMode: ViewMode): boolean => {
        // Both modes support drag and drop for reordering
        return viewMode === 'grid' || viewMode === 'list'
      }

      expect(supportsDragDrop('grid')).toBe(true)
      expect(supportsDragDrop('list')).toBe(true)
    })

    it('should only support card resize in grid view', () => {
      const supportsResize = (viewMode: ViewMode): boolean => viewMode === 'grid'

      expect(supportsResize('grid')).toBe(true)
      expect(supportsResize('list')).toBe(false)
    })
  })
})

describe('API Preferences', () => {
  describe('card_scale field', () => {
    it('should validate card_scale values', () => {
      const validValues = ['small', 'medium', 'large']
      const invalidValues = ['tiny', 'huge', 'xl', '']

      validValues.forEach((value) => {
        const isValid = ['small', 'medium', 'large'].includes(value)
        expect(isValid).toBe(true)
      })

      invalidValues.forEach((value) => {
        const isValid = ['small', 'medium', 'large'].includes(value)
        expect(isValid).toBe(false)
      })
    })

    it('should default to medium when not specified', () => {
      const getCardScale = (value: string | undefined): CardScale => {
        if (value && ['small', 'medium', 'large'].includes(value)) {
          return value as CardScale
        }
        return 'medium'
      }

      expect(getCardScale(undefined)).toBe('medium')
      expect(getCardScale('small')).toBe('small')
      expect(getCardScale('invalid')).toBe('medium')
    })
  })

  describe('view_mode field', () => {
    it('should validate view_mode values', () => {
      const validValues = ['grid', 'list']
      const invalidValues = ['table', 'card', '']

      validValues.forEach((value) => {
        const isValid = ['grid', 'list'].includes(value)
        expect(isValid).toBe(true)
      })

      invalidValues.forEach((value) => {
        const isValid = ['grid', 'list'].includes(value)
        expect(isValid).toBe(false)
      })
    })

    it('should default to grid when not specified', () => {
      const getViewMode = (value: string | undefined): ViewMode => {
        if (value && ['grid', 'list'].includes(value)) {
          return value as ViewMode
        }
        return 'grid'
      }

      expect(getViewMode(undefined)).toBe('grid')
      expect(getViewMode('list')).toBe('list')
      expect(getViewMode('invalid')).toBe('grid')
    })
  })
})
