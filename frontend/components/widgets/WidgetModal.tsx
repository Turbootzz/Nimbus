'use client'

import { useState } from 'react'
import { ArrowLeftIcon, XMarkIcon } from '@heroicons/react/24/outline'
import { useLocale, useTranslations } from 'next-intl'
import { api } from '@/lib/api'
import type { Widget, WidgetTypeMeta } from '@/types'
import BoardSelect from '@/components/boards/BoardSelect'
import { getWidgetDefinition, widgetConfig } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'
import IntegrationSelector from '@/components/integrations/IntegrationSelector'
import { kindName, useIntegrations } from '@/hooks/useIntegrations'
import { useModalLayer } from '@/hooks/useModalLayer'

interface WidgetModalProps {
  types: WidgetTypeMeta[]
  // Edits this widget when set, otherwise adds a new one
  widget?: Widget | null
  // Group for a new widget
  groupId?: string | null
  // Board for a new widget; the default board when undefined
  boardId?: string
  onClose: () => void
  onSaved: (widget: Widget) => void
}

// Types this version can add: the ones the frontend knows
export function addableTypes(types: WidgetTypeMeta[]): WidgetTypeMeta[] {
  return types.filter((meta) => getWidgetDefinition(meta.type))
}

interface IntegrationFieldProps {
  wanted: string[] // kinds the widget type can use
  value: string
  onChange: (id: string) => void
  disabled?: boolean
}

// Picks one of the user's integrations of the kinds a widget type uses
function IntegrationField({ wanted, value, onChange, disabled }: IntegrationFieldProps) {
  const t = useTranslations('widgets')
  const locale = useLocale()
  const { integrations, kinds, isLoading, error } = useIntegrations()
  const names = new Intl.ListFormat(locale, { type: 'disjunction' }).format(
    wanted.map((kind) => kindName(kinds, kind))
  )
  if (error) {
    return <p className="text-error text-sm">{t('common.integrationsError', { error })}</p>
  }
  return (
    <IntegrationSelector
      value={value}
      onChange={onChange}
      integrations={integrations.filter((i) => wanted.includes(i.kind))}
      kinds={kinds}
      isLoading={isLoading}
      disabled={disabled}
      label={t('modal.integration.label')}
      description={t('modal.integration.description', { names })}
      emptyOption={t('modal.integration.empty', { names })}
    />
  )
}

export default function WidgetModal({
  types,
  widget,
  groupId,
  boardId,
  onClose,
  onSaved,
}: WidgetModalProps) {
  const t = useTranslations('widgets')
  const isEdit = !!widget
  const [type, setType] = useState<string | null>(widget?.type ?? null)
  const [title, setTitle] = useState(widget?.title ?? '')
  const [config, setConfig] = useState<Record<string, unknown>>(() => {
    const definition = widget && getWidgetDefinition(widget.type)
    return definition ? widgetConfig(definition, widget) : {}
  })
  const [integrationId, setIntegrationId] = useState(widget?.integration_id ?? '')
  // Another board moves the widget there; '' keeps it where it is
  const [moveTo, setMoveTo] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  // Esc closes, except while a save is running
  const dialogRef = useModalLayer(onClose, !isSaving)

  const definition = type ? getWidgetDefinition(type) : undefined
  const typeMeta = types.find((meta) => meta.type === type)
  const wantedKinds = typeMeta?.integration_kinds ?? []

  const selectType = (next: string) => {
    const nextDefinition = getWidgetDefinition(next)
    if (!nextDefinition) return
    setType(next)
    setConfig(nextDefinition.defaultConfig)
    // An integration picked for another type may be of the wrong kind
    setIntegrationId('')
    setError(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!type) return
    if (wantedKinds.length > 0 && !integrationId) {
      setError(t('modal.chooseIntegration'))
      return
    }
    setError(null)
    setIsSaving(true)
    const integration = wantedKinds.length > 0 ? { integration_id: integrationId } : {}
    try {
      const response = widget
        ? await api.updateWidget(widget.id, {
            title: title.trim(),
            config,
            ...integration,
            ...(moveTo && { board_id: moveTo }),
          })
        : await api.createWidget({
            type,
            title: title.trim(),
            config,
            group_id: groupId || undefined,
            board_id: boardId,
            ...integration,
          })
      if (response.error || !response.data) {
        setError(response.error?.message || t('modal.saveFailed'))
        return
      }
      onSaved(response.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('modal.saveFailed'))
    } finally {
      setIsSaving(false)
    }
  }

  const heading = definition
    ? t(isEdit ? 'modal.editTitle' : 'modal.addTitle', {
        name: t(`types.${definition.type}.name`),
      })
    : t('modal.addWidget')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className={`absolute inset-0 bg-black/50 ${isSaving ? 'cursor-not-allowed' : ''}`}
        onClick={isSaving ? undefined : onClose}
        aria-hidden="true"
      />

      <div
        ref={dialogRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby="widget-modal-title"
        className="bg-card border-card-border relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col rounded-lg border shadow-lg focus:outline-none"
      >
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <div className="flex items-center gap-2">
            {definition && !isEdit && (
              <button
                type="button"
                onClick={() => setType(null)}
                className="text-text-secondary hover:text-text-primary transition-colors"
                aria-label={t('modal.back')}
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
            aria-label={t('modal.close')}
            disabled={isSaving}
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        {isEdit && !definition ? (
          <p className="text-text-secondary p-4 text-sm">
            {t('modal.cannotEdit', { type: widget.type })}
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
                      {t(`types.${typeDefinition.type}.name`)}
                    </span>
                    <span className="text-text-muted block text-xs">
                      {t(`types.${typeDefinition.type}.description`)}
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
                  {t.rich('modal.titleLabel', {
                    muted: (chunks) => (
                      <span className="text-text-muted font-normal">{chunks}</span>
                    ),
                  })}
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
              <definition.ConfigForm
                config={config}
                onChange={setConfig}
                disabled={isSaving}
                meta={typeMeta}
              />
              {widget && (
                <BoardSelect
                  id="widget-board"
                  currentBoardId={widget.board_id}
                  value={moveTo}
                  onChange={setMoveTo}
                  disabled={isSaving}
                />
              )}
            </div>

            <div className="border-card-border flex justify-end gap-2 border-t p-4">
              <button
                type="button"
                onClick={onClose}
                className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
                disabled={isSaving}
              >
                {t('modal.cancel')}
              </button>
              <button
                type="submit"
                disabled={isSaving}
                className="bg-primary hover:bg-primary-hover disabled:bg-primary/50 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed"
              >
                {isSaving
                  ? t('modal.saving')
                  : isEdit
                    ? t('modal.saveChanges')
                    : t('modal.addWidget')}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
