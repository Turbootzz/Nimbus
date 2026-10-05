'use client'

import { useState } from 'react'
import { ArrowLeftIcon, XMarkIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import type { Widget, WidgetTypeMeta } from '@/types'
import { getWidgetDefinition, widgetConfig } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'
import IntegrationSelector from '@/components/integrations/IntegrationSelector'
import { kindName, useIntegrations } from '@/hooks/useIntegrations'

interface WidgetModalProps {
  types: WidgetTypeMeta[]
  // Edits this widget when set, otherwise adds a new one
  widget?: Widget | null
  // Group for a new widget
  groupId?: string | null
  onClose: () => void
  onSaved: (widget: Widget) => void
}

// Types this version can add: the ones the frontend knows
export function addableTypes(types: WidgetTypeMeta[]): WidgetTypeMeta[] {
  return types.filter((t) => getWidgetDefinition(t.type))
}

interface IntegrationFieldProps {
  wanted: string[] // kinds the widget type can use
  value: string
  onChange: (id: string) => void
  disabled?: boolean
}

// Picks one of the user's integrations of the kinds a widget type uses
function IntegrationField({ wanted, value, onChange, disabled }: IntegrationFieldProps) {
  const { integrations, kinds, isLoading } = useIntegrations()
  const names = wanted.map((kind) => kindName(kinds, kind)).join(' or ')
  return (
    <IntegrationSelector
      value={value}
      onChange={onChange}
      integrations={integrations.filter((i) => wanted.includes(i.kind))}
      kinds={kinds}
      isLoading={isLoading}
      disabled={disabled}
      label="Integration"
      description={`The ${names} integration this widget reads.`}
      emptyOption={`Choose a ${names} integration`}
    />
  )
}

export default function WidgetModal({
  types,
  widget,
  groupId,
  onClose,
  onSaved,
}: WidgetModalProps) {
  const isEdit = !!widget
  const [type, setType] = useState<string | null>(widget?.type ?? null)
  const [title, setTitle] = useState(widget?.title ?? '')
  const [config, setConfig] = useState<Record<string, unknown>>(() => {
    const definition = widget && getWidgetDefinition(widget.type)
    return definition ? widgetConfig(definition, widget) : {}
  })
  const [integrationId, setIntegrationId] = useState(widget?.integration_id ?? '')
  const [error, setError] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)

  const definition = type ? getWidgetDefinition(type) : undefined
  const wantedKinds = types.find((t) => t.type === type)?.integration_kinds ?? []

  const selectType = (next: string) => {
    const nextDefinition = getWidgetDefinition(next)
    if (!nextDefinition) return
    setType(next)
    setConfig(nextDefinition.defaultConfig)
    setError(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!type) return
    if (wantedKinds.length > 0 && !integrationId) {
      setError('Choose an integration')
      return
    }
    setError(null)
    setIsSaving(true)
    const integration = wantedKinds.length > 0 ? { integration_id: integrationId } : {}
    try {
      const response = widget
        ? await api.updateWidget(widget.id, { title: title.trim(), config, ...integration })
        : await api.createWidget({
            type,
            title: title.trim(),
            config,
            group_id: groupId || undefined,
            ...integration,
          })
      if (response.error || !response.data) {
        setError(response.error?.message || 'Failed to save widget')
        return
      }
      onSaved(response.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save widget')
    } finally {
      setIsSaving(false)
    }
  }

  const heading = definition ? `${isEdit ? 'Edit' : 'Add'} ${definition.label}` : 'Add Widget'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className={`absolute inset-0 bg-black/50 ${isSaving ? 'cursor-not-allowed' : ''}`}
        onClick={isSaving ? undefined : onClose}
        aria-hidden="true"
      />

      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="widget-modal-title"
        className="bg-card border-card-border relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col rounded-lg border shadow-lg"
      >
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <div className="flex items-center gap-2">
            {definition && !isEdit && (
              <button
                type="button"
                onClick={() => setType(null)}
                className="text-text-secondary hover:text-text-primary transition-colors"
                aria-label="Back to widget types"
                disabled={isSaving}
              >
                <ArrowLeftIcon className="h-5 w-5" />
              </button>
            )}
            <h3 id="widget-modal-title" className="text-text-primary text-lg font-semibold">
              {heading}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label="Close"
            disabled={isSaving}
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        {isEdit && !definition ? (
          <p className="text-text-secondary p-4 text-sm">
            This widget type ({widget.type}) can&apos;t be edited in this version of Nimbus.
          </p>
        ) : !definition ? (
          <div className="grid grid-cols-1 gap-2 overflow-y-auto p-4 sm:grid-cols-2">
            {addableTypes(types).map((meta) => {
              const typeDefinition = getWidgetDefinition(meta.type)
              if (!typeDefinition) return null
              const Icon = typeDefinition.icon
              return (
                <button
                  key={meta.type}
                  type="button"
                  onClick={() => selectType(meta.type)}
                  className="border-card-border hover:border-primary hover:bg-card-hover flex items-start gap-3 rounded-lg border p-3 text-left transition-colors"
                >
                  <Icon className="text-primary h-6 w-6 shrink-0" />
                  <span>
                    <span className="text-text-primary block text-sm font-medium">
                      {typeDefinition.label}
                    </span>
                    <span className="text-text-muted block text-xs">
                      {typeDefinition.description}
                    </span>
                  </span>
                </button>
              )
            })}
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex min-h-0 flex-col">
            <div className="space-y-4 overflow-y-auto p-4">
              {error && (
                <div className="rounded-md bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">
                  {error}
                </div>
              )}
              <div>
                <label htmlFor="widget-title" className={labelClass}>
                  Title <span className="text-text-muted font-normal">(optional)</span>
                </label>
                <input
                  id="widget-title"
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  maxLength={100}
                  className={inputClass}
                  disabled={isSaving}
                />
              </div>
              {wantedKinds.length > 0 && (
                <IntegrationField
                  wanted={wantedKinds}
                  value={integrationId}
                  onChange={setIntegrationId}
                  disabled={isSaving}
                />
              )}
              <definition.ConfigForm config={config} onChange={setConfig} disabled={isSaving} />
            </div>

            <div className="border-card-border flex justify-end gap-2 border-t p-4">
              <button
                type="button"
                onClick={onClose}
                className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
                disabled={isSaving}
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isSaving}
                className="bg-primary hover:bg-primary-hover disabled:bg-primary/50 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed"
              >
                {isSaving ? 'Saving...' : isEdit ? 'Save Changes' : 'Add Widget'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
