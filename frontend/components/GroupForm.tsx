'use client'

import { useState } from 'react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import type { Group, GroupCreateRequest, GroupUpdateRequest } from '@/types'

interface GroupFormProps {
  group?: Group | null // If provided, edit mode; otherwise create mode
  onSubmit: (data: GroupCreateRequest | GroupUpdateRequest) => Promise<void>
  onClose: () => void
  isLoading?: boolean
}

// Preset colors for quick selection
const presetColors = [
  '#6366f1', // Indigo (default)
  '#ef4444', // Red
  '#f97316', // Orange
  '#eab308', // Yellow
  '#22c55e', // Green
  '#14b8a6', // Teal
  '#3b82f6', // Blue
  '#8b5cf6', // Violet
  '#ec4899', // Pink
  '#64748b', // Slate
]

// Get initial custom color value
const getInitialCustomColor = (groupColor: string | undefined) => {
  const color = groupColor || '#6366f1'
  return presetColors.includes(color) ? '' : color
}

export default function GroupForm({ group, onSubmit, onClose, isLoading = false }: GroupFormProps) {
  const [name, setName] = useState(group?.name || '')
  const [color, setColor] = useState(group?.color || '#6366f1')
  const [customColor, setCustomColor] = useState(() => getInitialCustomColor(group?.color))
  const [monitoringEnabled, setMonitoringEnabled] = useState(group?.monitoring_enabled ?? true)
  const [error, setError] = useState<string | null>(null)

  const isEditMode = !!group

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    // Validation - must match backend MaxGroupNameLen (35 chars)
    const trimmedName = name.trim()
    if (!trimmedName) {
      setError('Group name is required')
      return
    }
    if (trimmedName.length > 35) {
      setError('Group name must be 35 characters or less')
      return
    }

    // Validate color format
    const colorRegex = /^#[0-9A-Fa-f]{6}$/
    if (!colorRegex.test(color)) {
      setError('Invalid color format. Use hex format (e.g., #6366f1)')
      return
    }

    try {
      if (isEditMode) {
        await onSubmit({
          name: trimmedName,
          color,
          monitoring_enabled: monitoringEnabled,
        } as GroupUpdateRequest)
      } else {
        await onSubmit({
          name: trimmedName,
          color,
          monitoring_enabled: monitoringEnabled,
        } as GroupCreateRequest)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save group')
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
      <div className="bg-card border-card-border relative z-10 w-full max-w-md rounded-lg border shadow-lg">
        {/* Header */}
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <h3 className="text-text-primary text-lg font-semibold">
            {isEditMode ? 'Edit Group' : 'Create Group'}
          </h3>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label="Close"
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
              Name
            </label>
            <input
              id="group-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Enter group name"
              maxLength={35}
              className="bg-background border-card-border text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-primary w-full rounded-md border px-3 py-2 text-sm focus:ring-1 focus:outline-none"
              autoFocus
              disabled={isLoading}
            />
            <p className="text-text-muted mt-1 text-xs">{name.length}/35 characters</p>
          </div>

          {/* Color picker */}
          <div className="mb-4">
            <label className="text-text-primary mb-2 block text-sm font-medium">Color</label>

            {/* Preset colors */}
            <div className="mb-3 flex flex-wrap gap-2">
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
                  disabled={isLoading}
                />
              ))}
            </div>

            {/* Custom color input */}
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={color}
                onChange={(e) => handleColorSelect(e.target.value)}
                className="h-8 w-8 cursor-pointer rounded border-0 p-0"
                disabled={isLoading}
              />
              <input
                type="text"
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
            <p className="text-text-secondary mb-2 text-xs">Preview</p>
            <div className="flex items-center gap-2">
              <div className="h-4 w-4 rounded-full" style={{ backgroundColor: color }} />
              <span className="text-text-primary text-sm font-medium">
                {name.trim() || 'Group Name'}
              </span>
            </div>
          </div>

          {/* Monitoring Toggle */}
          <div className="mb-4 flex items-center justify-between">
            <div className="flex-1">
              <label htmlFor="monitoring_enabled" className="text-text-primary text-sm font-medium">
                Enable Monitoring
              </label>
              <p className="text-text-muted mt-0.5 text-xs">
                When disabled, services in this group won&apos;t be health-checked and won&apos;t
                appear in metrics
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={monitoringEnabled}
              onClick={() => setMonitoringEnabled(!monitoringEnabled)}
              disabled={isLoading}
              className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:ring-2 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50 ${
                monitoringEnabled ? 'bg-primary' : 'bg-gray-400'
              }`}
            >
              <span
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  monitoringEnabled ? 'translate-x-5' : 'translate-x-0'
                }`}
              />
            </button>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
              disabled={isLoading}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="bg-primary hover:bg-primary-hover disabled:bg-primary/50 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed"
            >
              {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Create Group'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
