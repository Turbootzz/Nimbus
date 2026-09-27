'use client'

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
  return (
    <div>
      <label htmlFor="note-content" className={labelClass}>
        Note
      </label>
      <textarea
        id="note-content"
        rows={8}
        value={config.content}
        onChange={(e) => onChange({ ...config, content: e.target.value })}
        placeholder={'# Title\n- a list item\n[a link](https://example.com)'}
        maxLength={MAX_NOTE_LENGTH}
        className={`${inputClass} font-mono`}
        disabled={disabled}
      />
      <p className="text-text-muted mt-1 text-xs">Markdown is supported. HTML is not.</p>
    </div>
  )
}
