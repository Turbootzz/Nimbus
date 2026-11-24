'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { api } from '@/lib/api'
import { isValidUrl } from '@/lib/utils/url'
import IconSelector from '@/components/IconSelector'
import GroupSelector from '@/components/GroupSelector'
import IntegrationSelector from '@/components/integrations/IntegrationSelector'
import { useIntegrations } from '@/hooks/useIntegrations'
import { Toggle } from '@/components/ui/Toggle'
import type { IconType, Group, Service } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { useGroupMonitoringLock } from '@/hooks/useGroupMonitoringLock'
import { useConfigFile } from '@/hooks/useConfigFile'
import { canEdit } from '@/lib/config-file'
import { ReadOnlyNote } from '@/components/ConfigFileBanner'
import BoardSelect from '@/components/boards/BoardSelect'

interface ServiceFormProps {
  // Edits this service when set, otherwise creates a new one
  service?: Service
  // Group preselected for a new service; falls back to the default group
  initialGroupId?: string | null
  // The board whose groups are offered and a new service goes to; the
  // user's default board when undefined
  boardId?: string
  // 'page' is a card on a page, 'modal' fills a dialog and scrolls inside it
  variant?: 'page' | 'modal'
  onSaved: (service: Service) => void
  // Cancel is a link on pages and a button in the modal
  cancelHref?: string
  onCancel?: () => void
  onSavingChange?: (isSaving: boolean) => void
}

const labelClass = 'text-text-secondary mb-2 block text-sm font-medium'
const inputClass =
  'border-card-border bg-background text-text-primary focus:border-primary focus:ring-opacity-50 w-full rounded-md border px-4 py-2 transition focus:ring-2 focus:outline-none'
const cancelClass =
  'hover:bg-card-border text-text-secondary hover:text-text-primary rounded-md px-4 py-2 text-sm font-medium transition-colors'

// The one form to create and edit a service
export default function ServiceForm({
  service,
  initialGroupId,
  boardId,
  variant = 'page',
  onSaved,
  cancelHref,
  onCancel,
  onSavingChange,
}: ServiceFormProps) {
  const isEdit = !!service
  const isModal = variant === 'modal'
  const { enableServiceGrouping } = useTheme()
  const [isSaving, setIsSaving] = useState(false)
  // Locked by a config file: saving would be refused
  const configFile = useConfigFile()
  const readOnly = configFile?.read_only ?? false
  const [error, setError] = useState('')
  const [uploadedFile, setUploadedFile] = useState<File | null>(null)
  const [groups, setGroups] = useState<Group[]>([])
  const [groupsLoading, setGroupsLoading] = useState(true)
  // Another board moves the service there, out of its group; '' stays
  const [moveTo, setMoveTo] = useState('')

  const [formData, setFormData] = useState({
    name: service?.name ?? '',
    url: service?.url ?? '',
    icon: service?.icon || '🔗',
    icon_type: (service?.icon_type || 'emoji') as IconType,
    icon_image_path: service?.icon_image_path || '',
    description: service?.description || '',
    group_id: service?.group_id || '',
    integration_id: service?.integration_id || '',
    monitoring_enabled: service?.monitoring_enabled ?? true,
  })
  const { integrations, kinds, isLoading: integrationsLoading } = useIntegrations()

  const { groupMonitoringDisabled, monitoringDescription } = useGroupMonitoringLock({
    groups,
    selectedGroupId: formData.group_id,
    enableServiceGrouping,
    monitoringEnabled: formData.monitoring_enabled,
    setMonitoringEnabled: (next) => setFormData((prev) => ({ ...prev, monitoring_enabled: next })),
  })

  // Fetch groups when grouping is enabled
  useEffect(() => {
    const fetchGroups = async () => {
      if (!enableServiceGrouping) {
        setGroupsLoading(false)
        return
      }

      setGroupsLoading(true)
      try {
        // A service is edited with the groups of the board it is on
        const response = await api.getGroups(service?.board_id ?? boardId)
        if (response.error) {
          setError(`Failed to load groups: ${response.error.message}`)
          return
        }
        if (response.data) {
          setGroups(response.data)
          // A new service starts in the preselected group, or the default one
          if (!isEdit) {
            const preselectedGroup = initialGroupId
              ? response.data.find((g) => g.id === initialGroupId)
              : null
            const groupToSelect = preselectedGroup || response.data.find((g) => g.is_default)
            if (groupToSelect) {
              setFormData((prev) => ({ ...prev, group_id: groupToSelect.id }))
            }
          }
        }
      } catch (error) {
        console.error('Failed to fetch groups:', error)
        setError('Failed to load groups. Please try again.')
      } finally {
        setGroupsLoading(false)
      }
    }

    fetchGroups()
  }, [enableServiceGrouping, initialGroupId, isEdit, boardId, service?.board_id])

  const setSaving = (next: boolean) => {
    setIsSaving(next)
    onSavingChange?.(next)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!formData.name.trim()) {
      setError('Service name is required')
      return
    }
    if (!formData.url.trim()) {
      setError('Service URL is required')
      return
    }
    if (!isValidUrl(formData.url)) {
      setError('Please enter a valid URL (e.g., https://example.com)')
      return
    }

    setSaving(true)
    try {
      // Upload image if needed
      let iconImagePath = formData.icon_image_path
      if (formData.icon_type === 'image_upload' && uploadedFile) {
        const uploadResponse = await api.uploadServiceIcon(uploadedFile)
        if (uploadResponse.error) {
          setError(`Image upload failed: ${uploadResponse.error.message}`)
          return
        }
        iconImagePath = uploadResponse.data?.icon_image_path || ''
      }

      const data = {
        name: formData.name.trim(),
        url: formData.url.trim(),
        icon: formData.icon.trim() || '🔗',
        icon_type: formData.icon_type,
        icon_image_path: iconImagePath,
        description: formData.description.trim(),
        monitoring_enabled: formData.monitoring_enabled,
      }
      // An update sends '' to clear the group or unlink the integration; a
      // new service leaves them out. A move leaves the group behind.
      const response = service
        ? await api.updateService(service.id, {
            ...data,
            group_id: moveTo ? '' : enableServiceGrouping ? formData.group_id : undefined,
            integration_id: formData.integration_id,
            ...(moveTo && { board_id: moveTo }),
          })
        : await api.createService({
            ...data,
            group_id: enableServiceGrouping && formData.group_id ? formData.group_id : undefined,
            board_id: boardId,
            integration_id: formData.integration_id || undefined,
          })

      if (response.error || !response.data) {
        setError(response.error?.message || 'Unable to save service. Please try again.')
        return
      }
      onSaved(response.data)
    } catch (error) {
      console.error('Failed to save service:', error)
      const action = isEdit ? 'update' : 'create'
      setError(
        error instanceof Error ? error.message : `Unable to ${action} service. Please try again.`
      )
    } finally {
      setSaving(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
  }

  const submitLabel = isEdit
    ? isSaving
      ? 'Saving...'
      : 'Save Changes'
    : isSaving
      ? 'Creating...'
      : 'Create Service'

  return (
    <>
      {error && (
        <div
          role="alert"
          className={`bg-error border-error rounded-lg border p-3 text-sm text-white opacity-90 ${
            isModal ? 'mx-4 mt-4' : 'mb-4'
          }`}
        >
          {error}
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className={
          isModal ? 'flex min-h-0 flex-col' : 'glass-card border-card-border rounded-lg border p-6'
        }
      >
        <div className={isModal ? 'space-y-6 overflow-y-auto p-4' : 'space-y-6'}>
          {/* Service Name */}
          <div>
            <label htmlFor="name" className={labelClass}>
              Service Name <span className="text-error">*</span>
            </label>
            <input
              type="text"
              id="name"
              name="name"
              value={formData.name}
              onChange={handleChange}
              className={inputClass}
              placeholder="e.g., Plex Media Server"
              required
              disabled={isSaving}
              autoFocus={isModal}
            />
          </div>

          {/* Service URL */}
          <div>
            <label htmlFor="url" className={labelClass}>
              Service URL <span className="text-error">*</span>
            </label>
            <input
              type="url"
              id="url"
              name="url"
              value={formData.url}
              onChange={handleChange}
              className={inputClass}
              placeholder="https://plex.example.com"
              required
              disabled={isSaving}
            />
            <p className="text-text-muted mt-1 text-xs">
              The URL where your service can be accessed
            </p>
          </div>

          {/* Service Icon */}
          <IconSelector
            icon={formData.icon}
            iconType={formData.icon_type}
            iconImagePath={formData.icon_image_path}
            serviceName={formData.name}
            serviceUrl={formData.url}
            onIconChange={(icon) => setFormData((prev) => ({ ...prev, icon }))}
            onIconTypeChange={(icon_type) => {
              setFormData((prev) => ({ ...prev, icon_type }))
              setUploadedFile(null) // Clear uploaded file when switching icon type
            }}
            onIconImagePathChange={(icon_image_path) =>
              setFormData((prev) => ({ ...prev, icon_image_path }))
            }
            onFileSelect={(file) => setUploadedFile(file)}
          />

          {service && (
            <BoardSelect
              id="service-board"
              currentBoardId={service.board_id}
              value={moveTo}
              onChange={setMoveTo}
              disabled={isSaving}
            />
          )}

          {/* Group Selector (only when grouping is enabled); a move clears it */}
          {enableServiceGrouping && !moveTo && (
            <GroupSelector
              value={formData.group_id}
              onChange={(value) => setFormData((prev) => ({ ...prev, group_id: value }))}
              groups={groups}
              isLoading={groupsLoading}
              disabled={isSaving}
            />
          )}

          <IntegrationSelector
            value={formData.integration_id}
            onChange={(value) => setFormData((prev) => ({ ...prev, integration_id: value }))}
            integrations={integrations}
            kinds={kinds}
            isLoading={integrationsLoading}
            disabled={isSaving}
          />

          {/* Service Description */}
          <div>
            <label htmlFor="description" className={labelClass}>
              Description
            </label>
            <textarea
              id="description"
              name="description"
              value={formData.description}
              onChange={handleChange}
              rows={3}
              className={inputClass}
              placeholder="Brief description of what this service does"
              disabled={isSaving}
            />
          </div>

          {/* Monitoring Toggle */}
          <Toggle
            id="monitoring_enabled"
            enabled={formData.monitoring_enabled}
            onChange={(enabled) =>
              setFormData((prev) => ({ ...prev, monitoring_enabled: enabled }))
            }
            label="Enable Monitoring"
            description={monitoringDescription}
            disabled={isSaving || groupMonitoringDisabled}
          />
        </div>

        {/* Form Actions */}
        <div
          className={`border-card-border flex items-center justify-end gap-3 border-t ${
            isModal ? 'p-4' : 'mt-6 pt-6'
          }`}
        >
          {cancelHref ? (
            <Link href={cancelHref} className={cancelClass}>
              Cancel
            </Link>
          ) : (
            <button type="button" onClick={onCancel} disabled={isSaving} className={cancelClass}>
              Cancel
            </button>
          )}
          <button
            type="submit"
            disabled={isSaving || !canEdit(configFile)}
            className="bg-primary hover:bg-primary-hover rounded-md px-6 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitLabel}
          </button>
        </div>
        {readOnly && <ReadOnlyNote />}
      </form>
    </>
  )
}
