'use client'

import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline'
import { useTranslations } from 'next-intl'
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
  const t = useTranslations('widgets')
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
        <p className="text-text-muted mt-1 text-xs">{t('customApi.form.urlHint')}</p>
      </div>

      <Toggle
        id="custom-api-verify-tls"
        enabled={config.verify_tls}
        onChange={(verify_tls) => onChange({ ...config, verify_tls })}
        label={t('common.verifyTls')}
        description={t('customApi.form.verifyTlsHint')}
        disabled={disabled}
      />

      <div>
        <span className={labelClass}>{t('customApi.form.values')}</span>
        <div className="space-y-2">
          {config.fields.map((field, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={t('customApi.form.valueLabel', { n: index + 1 })}
                value={field.label}
                onChange={(e) => updateField(index, { label: e.target.value })}
                placeholder={t('customApi.form.labelPlaceholder')}
                maxLength={40}
                className={inputClass}
                disabled={disabled}
              />
              <input
                aria-label={t('customApi.form.valuePath', { n: index + 1 })}
                value={field.path}
                onChange={(e) => updateField(index, { path: e.target.value })}
                placeholder="data.items.length"
                maxLength={200}
                className={`${inputClass} font-mono`}
                disabled={disabled}
              />
              <input
                aria-label={t('customApi.form.valueUnit', { n: index + 1 })}
                value={field.unit}
                onChange={(e) => updateField(index, { unit: e.target.value })}
                placeholder={t('customApi.form.unitPlaceholder')}
                maxLength={10}
                className={`${inputClass} w-20! shrink-0`}
                disabled={disabled}
              />
              <button
                type="button"
                onClick={() => setFields(config.fields.filter((_, i) => i !== index))}
                className={removeRowClass}
                aria-label={t('customApi.form.removeValue', { n: index + 1 })}
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
            {t('customApi.form.addValue')}
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">
          {t.rich('customApi.form.pathsHint', { code: (chunks) => <code>{chunks}</code> })}
        </p>
      </div>

      <div>
        <span className={labelClass}>{t('customApi.form.headers')}</span>
        <div className="space-y-2">
          {config.headers.map((header, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                aria-label={t('customApi.form.headerName', { n: index + 1 })}
                value={header.name}
                onChange={(e) => updateHeader(index, { name: e.target.value })}
                placeholder="X-Api-Key"
                className={inputClass}
                disabled={disabled}
              />
              <input
                aria-label={t('customApi.form.headerValue', { n: index + 1 })}
                value={header.value}
                onChange={(e) => updateHeader(index, { value: e.target.value })}
                placeholder={t('customApi.form.valuePlaceholder')}
                className={inputClass}
                autoComplete="off"
                disabled={disabled}
              />
              <button
                type="button"
                onClick={() => setHeaders(config.headers.filter((_, i) => i !== index))}
                className={removeRowClass}
                aria-label={t('customApi.form.removeHeader', { n: index + 1 })}
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
            {t('customApi.form.addHeader')}
          </button>
        )}
        <p className="text-text-muted mt-1 text-xs">{t('customApi.form.headersHint')}</p>
      </div>
    </div>
  )
}
