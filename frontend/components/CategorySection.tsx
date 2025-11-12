'use client'

import { useState } from 'react'
import { ChevronDownIcon, ChevronRightIcon } from '@heroicons/react/24/outline'
import type { Service, Category } from '@/types'
import CategoryBadge from './CategoryBadge'

interface CategorySectionProps {
  category: Category | null
  services: Service[]
  renderService: (service: Service) => React.ReactNode
  defaultExpanded?: boolean
}

export default function CategorySection({
  category,
  services,
  renderService,
  defaultExpanded = true,
}: CategorySectionProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded)

  if (services.length === 0) {
    return null
  }

  return (
    <div className="mb-6">
      {/* Category header - collapsible */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="text-text-primary mb-3 flex w-full items-center gap-2 text-left text-xl font-semibold transition-colors hover:opacity-80"
      >
        {isExpanded ? (
          <ChevronDownIcon className="h-5 w-5 flex-shrink-0" />
        ) : (
          <ChevronRightIcon className="h-5 w-5 flex-shrink-0" />
        )}
        {category ? (
          <CategoryBadge category={category} size="md" />
        ) : (
          <span className="text-text-secondary">Uncategorized</span>
        )}
        <span className="text-text-muted text-sm font-normal">({services.length})</span>
      </button>

      {/* Services grid - only shown when expanded */}
      {isExpanded && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {services.map((service) => renderService(service))}
        </div>
      )}
    </div>
  )
}
