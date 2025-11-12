import { Category } from '@/types'

interface CategoryBadgeProps {
  category: Category
  size?: 'sm' | 'md' | 'lg'
  showName?: boolean
}

export default function CategoryBadge({
  category,
  size = 'md',
  showName = true,
}: CategoryBadgeProps) {
  const sizeClasses = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-sm px-2.5 py-1',
    lg: 'text-base px-3 py-1.5',
  }

  const dotSizes = {
    sm: 'w-2 h-2',
    md: 'w-2.5 h-2.5',
    lg: 'w-3 h-3',
  }

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full bg-gray-100 dark:bg-gray-800 ${sizeClasses[size]}`}
    >
      <span
        className={`${dotSizes[size]} rounded-full`}
        style={{ backgroundColor: category.color }}
        aria-label={`Category color: ${category.color}`}
      />
      {showName && (
        <span className="font-medium text-gray-700 dark:text-gray-300">{category.name}</span>
      )}
    </span>
  )
}
