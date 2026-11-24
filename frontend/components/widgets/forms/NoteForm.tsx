'use client'

import { useTranslations } from 'next-intl'
import type { NoteWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

// Matches maxMarkdownRunes in the backend
const MAX_NOTE_LENGTH = 10000

export default function NoteForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<NoteWidgetConfig>) {
  const t = useTranslations('widgets.note.form')
  return (
    <div>
      <label htmlFor="note-content" className={labelClass}>
        {t('label')}
      </label>
      <textarea
        id="note-content"
        rows={8}
        value={config.content}
        onChange={(e) => onChange({ ...config, content: e.target.value })}
        placeholder={t('placeholder')}
        maxLength={MAX_NOTE_LENGTH}
        className={`${inputClass} font-mono`}
        disabled={disabled}
      />
      <p className="text-text-muted mt-1 text-xs">{t('hint')}</p>
    </div>
  )
}
