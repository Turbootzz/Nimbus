'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import type { Bookmark, BookmarksWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import {
  addRowClass,
  inputClass,
  labelClass,
  removeRowClass,
} from '@/components/widgets/forms/fieldStyles'

// Matches maxBookmarks in the backend
const MAX_BOOKMARKS = 50

export default function BookmarksForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<BookmarksWidgetConfig>) {
  const setItems = (items: Bookmark[]) => onChange({ ...config, items })
  const updateItem = (index: number, patch: Partial<Bookmark>) =>
    setItems(config.items.map((item, i) => (i === index ? { ...item, ...patch } : item)))

  return (
    <div>
      <span className={labelClass}>Bookmarks</span>
      <div className="space-y-2">
        {config.items.map((item, index) => (
          <div key={index} className="flex items-center gap-2">
            <input
              aria-label={`Bookmark ${index + 1} icon`}
              value={item.icon}
              onChange={(e) => updateItem(index, { icon: e.target.value })}
              placeholder="🔗"
              className={`${inputClass} w-14! shrink-0 text-center`}
              disabled={disabled}
            />
            <input
              aria-label={`Bookmark ${index + 1} name`}
              value={item.name}
              onChange={(e) => updateItem(index, { name: e.target.value })}
              placeholder="Name"
              maxLength={100}
              className={inputClass}
              disabled={disabled}
            />
            <input
              aria-label={`Bookmark ${index + 1} URL`}
              type="url"
              value={item.url}
              onChange={(e) => updateItem(index, { url: e.target.value })}
              placeholder="https://"
              className={inputClass}
              disabled={disabled}
            />
            <button
              type="button"
              onClick={() => setItems(config.items.filter((_, i) => i !== index))}
              className={removeRowClass}
              aria-label={`Remove bookmark ${index + 1}`}
              disabled={disabled}
            >
              <TrashIcon className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
      {config.items.length < MAX_BOOKMARKS && (
        <button
          type="button"
          onClick={() => setItems([...config.items, { name: '', url: '', icon: '' }])}
          className={addRowClass}
          disabled={disabled}
        >
          <PlusIcon className="mr-1 h-4 w-4" />
          Add bookmark
        </button>
      )}
      <p className="text-text-muted mt-1 text-xs">The icon can be an emoji or an image URL.</p>
    </div>
  )
}
