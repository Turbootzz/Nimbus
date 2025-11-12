import { Category } from '@/types'

interface CategorySelectProps {
  value: string | null
  onChange: (categoryId: string | null) => void
  categories: Category[]
  label?: string
  error?: string
}

export default function CategorySelect({
  value,
  onChange,
  categories,
  label = 'Category',
  error,
}: CategorySelectProps) {
  return (
    <div>
      <label
        htmlFor="category"
        className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
      >
        {label}
      </label>
      <select
        id="category"
        value={value || ''}
        onChange={(e) => onChange(e.target.value || null)}
        className="w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-gray-900 focus:border-transparent focus:ring-2 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
      >
        <option value="">No category</option>
        {categories.map((category) => (
          <option key={category.id} value={category.id}>
            {category.name}
          </option>
        ))}
      </select>
      {error && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>}
    </div>
  )
}
