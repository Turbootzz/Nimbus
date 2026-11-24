'use client'

import { useState } from 'react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import { useTranslations } from 'next-intl'
import { Toggle } from '@/components/ui/Toggle'
import { useModalLayer } from '@/hooks/useModalLayer'
import type { Group, GroupCreateRequest, GroupUpdateRequest, TileStyle } from '@/types'

interface GroupFormProps {
  group?: Group | null // If provided, edit mode; otherwise create mode
  onSubmit: (data: GroupCreateRequest | GroupUpdateRequest) => Promise<void>
  onClose: () => void
  isLoading?: boolean
}

// Preset colors for quick selection, by the key of the name a screen reader says
const colorNames = {
  '#6366f1': 'indigo', // the default
  '#ef4444': 'red',
  '#f97316': 'orange',
  '#eab308': 'yellow',
  '#22c55e': 'green',
  '#14b8a6': 'teal',
  '#3b82f6': 'blue',
  '#8b5cf6': 'violet',
  '#ec4899': 'pink',
  '#64748b': 'slate',
} as const
const presetColors = Object.keys(colorNames)

const tileStyles: TileStyle[] = ['cards', 'chips']

// Get initial custom color value
const getInitialCustomColor = (groupColor: string | undefined) => {
  const color = groupColor || '#6366f1'
  return presetColors.includes(color) ? '' : color
}

export default function GroupForm({ group, onSubmit, onClose, isLoading = false }: GroupFormProps) {
  const t = useTranslations('dashboard.groups.form')
  const [name, setName] = useState(group?.name || '')
  const [color, setColor] = useState(group?.color || '#6366f1')
  const [customColor, setCustomColor] = useState(() => getInitialCustomColor(group?.color))
  const [monitoringEnabled, setMonitoringEnabled] = useState(group?.monitoring_enabled ?? true)
  const [tileStyle, setTileStyle] = useState<TileStyle>(group?.tile_style ?? 'cards')
  const [error, setError] = useState<string | null>(null)

  const isEditMode = !!group
  // Esc closes, Tab stays inside, focus returns on close; not while saving
  const dialogRef = useModalLayer(onClose, !isLoading)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    // Validation - must match backend MaxGroupNameLen (35 chars)
    const trimmedName = name.trim()
    if (!trimmedName) {
      setError(t('nameRequired'))
      return
    }
    if (trimmedName.length > 35) {
      setError(t('nameTooLong'))
      return
    }

    // Validate color format
    const colorRegex = /^#[0-9A-Fa-f]{6}$/
    if (!colorRegex.test(color)) {
      setError(t('invalidColor'))
      return
    }

    try {
      await onSubmit({
        name: trimmedName,
        color,
        monitoring_enabled: monitoringEnabled,
        tile_style: tileStyle,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : t('saveFailed'))
    }
  }

  const handleColorSelect = (selectedColor: string) => {
    setColor(selectedColor)
    // Clear custom color if selecting a preset, otherwise update it
    if (presetColors.includes(selectedColor)) {
      setCustomColor('')
    } else {
      setCustomColor(selectedColor)
    }
  }

  const handleCustomColorChange = (value: string) => {
    setCustomColor(value)
    // Auto-update color if valid hex
    if (/^#[0-9A-Fa-f]{6}$/.test(value)) {
      setColor(value)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop - disabled during loading to prevent accidental closes */}
      <div
        className={`absolute inset-0 bg-black/50 ${isLoading ? 'cursor-not-allowed' : ''}`}
        onClick={isLoading ? undefined : onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        ref={dialogRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby="group-form-title"
        className="bg-card border-card-border relative z-10 w-full max-w-md rounded-lg border shadow-lg focus:outline-none"
      >
        {/* Header */}
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <h3 id="group-form-title" className="text-text-primary text-lg font-semibold">
            {isEditMode ? t('editTitle') : t('createTitle')}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label={t('close')}
            disabled={isLoading}
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-4">
          {/* Error message */}
          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">
              {error}
            </div>
          )}

          {/* Name input */}
          <div className="mb-4">
            <label
              htmlFor="group-name"
              className="text-text-primary mb-1 block text-sm font-medium"
            >
              {t('name')}
            </label>
            <input
              id="group-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('namePlaceholder')}
              maxLength={35}
              className="bg-background border-card-border text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-primary w-full rounded-md border px-3 py-2 text-sm focus:ring-1 focus:outline-none"
              autoFocus
              disabled={isLoading}
            />
            <p className="text-text-muted mt-1 text-xs">{t('charCount', { count: name.length })}</p>
          </div>

          {/* Color picker */}
          <div className="mb-4">
            <p id="group-color-label" className="text-text-primary mb-2 block text-sm font-medium">
              {t('color')}
            </p>

            {/* Preset colors */}
            <div
              role="group"
              aria-labelledby="group-color-label"
              className="mb-3 flex flex-wrap gap-2"
            >
              {presetColors.map((presetColor) => (
                <button
                  key={presetColor}
                  type="button"
                  onClick={() => handleColorSelect(presetColor)}
                  className={`h-8 w-8 rounded-full border-2 transition-transform hover:scale-110 ${
                    color === presetColor ? 'border-text-primary scale-110' : 'border-transparent'
                  }`}
                  style={{ backgroundColor: presetColor }}
                  title={presetColor}
                  aria-label={t(`colors.${colorNames[presetColor as keyof typeof colorNames]}`)}
                  aria-pressed={color === presetColor}
                  disabled={isLoading}
                />
              ))}
            </div>

            {/* Custom color input */}
            <div className="flex items-center gap-2">
              <input
                type="color"
                aria-label={t('customColor')}
                value={color}
                onChange={(e) => handleColorSelect(e.target.value)}
                className="h-8 w-8 cursor-pointer rounded border-0 p-0"
                disabled={isLoading}
              />
              <input
                type="text"
                aria-label={t('colorCode')}
                value={customColor || color}
                onChange={(e) => handleCustomColorChange(e.target.value)}
                placeholder="#6366f1"
                className="bg-background border-card-border text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-primary flex-1 rounded-md border px-3 py-1.5 text-sm focus:ring-1 focus:outline-none"
                disabled={isLoading}
              />
            </div>
          </div>

          {/* Preview */}
          <div className="border-card-border mb-4 rounded-md border p-3">
            <p className="text-text-secondary mb-2 text-xs">{t('preview')}</p>
            <div className="flex items-center gap-2">
              <div className="h-4 w-4 rounded-full" style={{ backgroundColor: color }} />
              <span className="text-text-primary text-sm font-medium">
                {name.trim() || t('previewName')}
              </span>
            </div>
          </div>

          {/* Tile style */}
          <fieldset className="mb-4" disabled={isLoading}>
            <legend className="text-text-primary mb-2 block text-sm font-medium">
              {t('tileStyle')}
            </legend>
            <div className="grid grid-cols-2 gap-2">
              {tileStyles.map((style) => (
                <label
                  key={style}
                  className={`has-focus-visible:ring-primary cursor-pointer rounded-md border p-2 transition-colors has-focus-visible:ring-2 ${
                    tileStyle === style
                      ? 'border-primary'
                      : 'border-card-border hover:border-text-muted'
                  }`}
                >
                  <input
                    type="radio"
                    name="tile_style"
                    value={style}
                    checked={tileStyle === style}
                    onChange={() => setTileStyle(style)}
                    className="sr-only"
                  />
                  <span className="text-text-primary block text-sm font-medium">
                    {t(`tileStyles.${style}.label`)}
                  </span>
                  <span className="text-text-muted block text-xs">
                    {t(`tileStyles.${style}.description`)}
                  </span>
                </label>
              ))}
            </div>
          </fieldset>

          {/* Monitoring Toggle */}
          <div className="mb-4">
            <Toggle
              id="monitoring_enabled"
              enabled={monitoringEnabled}
              onChange={setMonitoringEnabled}
              label={t('monitoring')}
              description={t('monitoringDescription')}
              disabled={isLoading}
            />
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
              disabled={isLoading}
            >
              {t('cancel')}
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="bg-primary hover:bg-primary-hover disabled:bg-primary/50 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed"
            >
              {isLoading ? t('saving') : isEditMode ? t('save') : t('create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
