'use client'

import { useTranslations } from 'next-intl'
import type { Group } from '@/types'

interface GroupSelectorProps {
  value: string
  onChange: (value: string) => void
  groups: Group[]
  isLoading?: boolean
  disabled?: boolean
}

export default function GroupSelector({
  value,
  onChange,
  groups,
  isLoading = false,
  disabled = false,
}: GroupSelectorProps) {
  const t = useTranslations('dashboard.groups.selector')
  return (
    <div>
      <label htmlFor="group_id" className="text-text-secondary mb-2 block text-sm font-medium">
        {t('label')}
      </label>
      <div className="relative">
        <select
          id="group_id"
          name="group_id"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="border-card-border focus:border-primary focus:ring-opacity-50 w-full appearance-none rounded-md border px-4 py-2 pr-10 transition focus:ring-2 focus:outline-none"
          style={{
            backgroundColor: 'var(--color-background)',
            color: 'var(--color-text-primary)',
          }}
          disabled={disabled || isLoading}
        >
          {isLoading ? (
            <option value="">{t('loading')}</option>
          ) : groups.length === 0 ? (
            <option value="">{t('noGroups')}</option>
          ) : (
            <>
              <option value="">{t('none')}</option>
              {groups.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.is_default ? t('defaultOption', { name: group.name }) : group.name}
                </option>
              ))}
            </>
          )}
        </select>
        {/* Dropdown arrow */}
        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3">
          <svg
            className="text-text-muted h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>
      <p className="text-text-muted mt-1 text-xs">{t('hint')}</p>
    </div>
  )
}
