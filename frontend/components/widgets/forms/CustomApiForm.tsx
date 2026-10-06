'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import type { CustomApiField, CustomApiHeader, CustomApiWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { Toggle } from '@/components/ui/Toggle'
import {
  addRowClass,
  inputClass,
  labelClass,
  removeRowClass,
} from '@/components/widgets/forms/fieldStyles'

// Match the limits in the backend
const MAX_FIELDS = 4
const MAX_HEADERS = 10

export default function CustomApiForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<CustomApiWidgetConfig>) {
  const setFields = (fields: CustomApiField[]) => onChange({ ...config, fields })
  const updateField = (index: number, patch: Partial<CustomApiField>) =>
    setFields(config.fields.map((field, i) => (i === index ? { ...field, ...patch } : field)))
  const setHeaders = (headers: CustomApiHeader[]) => onChange({ ...config, headers })
  const updateHeader = (index: number, patch: Partial<CustomApiHeader>) =>
    setHeaders(config.headers.map((header, i) => (i === index ? { ...header, ...patch } : header)))

  return (
    <div className="space-y-4">
      <div>
        <label htmlFor="custom-api-url" className={labelClass}>
          URL
        </label>
        <input
          id="custom-api-url"
          type="url"
          value={config.url}
          onChange={(e) => onChange({ ...config, url: e.target.value })}
          placeholder="http://192.168.1.10:8080/api/stats"
          className={inputClass}
          disabled={disabled}
        />
        <p className="text-text-muted mt-1 text-xs">
          The Nimbus server fetches this URL; it must return JSON.
        </p>
      </div>

      <Toggle
        id="custom-api-verify-tls"
        enabled={config.verify_tls}
        onChange={(verify_tls) => onChange({ ...config, verify_tls })}
        label="Verify TLS certificate"
        description="Turn off for apps with a self-signed certificate"
        disabled={disabled}
      />

      <div>
        <span className={labelClass}>Values</span>
        <div className="space-y-2">
          {config.fields.map((field, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={`Value ${index + 1} label`}
                value={field.label}
                onChange={(e) => updateField(index, { label: e.target.value })}
                placeholder="Label"
                maxLength={40}
                className={inputClass}
                disabled={disabled}
              />
              <input
                aria-label={`Value ${index + 1} path`}
                value={field.path}
                onChange={(e) => updateField(index, { path: e.target.value })}
                placeholder="data.items.length"
                maxLength={200}
                className={`${inputClass} font-mono`}
                disabled={disabled}
              />
              <input
                aria-label={`Value ${index + 1} unit`}
                value={field.unit}
                onChange={(e) => updateField(index, { unit: e.target.value })}
                placeholder="Unit"
                maxLength={10}
                className={`${inputClass} w-20! shrink-0`}
                disabled={disabled}
              />
              <button
                type="button"
                onClick={() => setFields(config.fields.filter((_, i) => i !== index))}
                className={removeRowClass}
                aria-label={`Remove value ${index + 1}`}
                disabled={disabled}
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
        {config.fields.length < MAX_FIELDS && (
          <button
            type="button"
            onClick={() => setFields([...config.fields, { label: '', path: '', unit: '' }])}
            className={addRowClass}
            disabled={disabled}
          >
            <PlusIcon className="mr-1 h-4 w-4" />
            Add value
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">
          Paths use dots and indexes: <code>stats.cpu</code>, <code>items[0].name</code>,{' '}
          <code>items[-1].name</code> for the last item, <code>items.length</code> to count.
        </p>
      </div>

      <div>
        <span className={labelClass}>Headers (optional)</span>
        <div className="space-y-2">
          {config.headers.map((header, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={`Header ${index + 1} name`}
                value={header.name}
                onChange={(e) => updateHeader(index, { name: e.target.value })}
                placeholder="X-Api-Key"
                className={inputClass}
                disabled={disabled}
              />
              <input
                aria-label={`Header ${index + 1} value`}
                value={header.value}
                onChange={(e) => updateHeader(index, { value: e.target.value })}
                placeholder="Value"
                className={inputClass}
                autoComplete="off"
                disabled={disabled}
              />
              <button
                type="button"
                onClick={() => setHeaders(config.headers.filter((_, i) => i !== index))}
                className={removeRowClass}
                aria-label={`Remove header ${index + 1}`}
                disabled={disabled}
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
        {config.headers.length < MAX_HEADERS && (
          <button
            type="button"
            onClick={() => setHeaders([...config.headers, { name: '', value: '' }])}
            className={addRowClass}
            disabled={disabled}
          >
            <PlusIcon className="mr-1 h-4 w-4" />
            Add header
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">
          Saved values show as ******** and are kept as long as you leave them. They are stored with
          the widget, not encrypted; for apps Nimbus supports, use an integration instead.
        </p>
      </div>
    </div>
  )
}
