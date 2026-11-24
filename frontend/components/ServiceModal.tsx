'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { XMarkIcon } from '@heroicons/react/24/outline'
import type { Service } from '@/types'
import ServiceForm from '@/components/ServiceForm'
import { useModalLayer } from '@/hooks/useModalLayer'

interface ServiceModalProps {
  service: Service
  // The board the dashboard shows; undefined is the default board
  boardId?: string
  onClose: () => void
  onSaved: (service: Service) => void
}

// Edits a service in place on the dashboard; looks like WidgetModal
export default function ServiceModal({ service, boardId, onClose, onSaved }: ServiceModalProps) {
  const t = useTranslations('services.modal')
  const [isSaving, setIsSaving] = useState(false)

  // Esc closes, except while a save is running
  const dialogRef = useModalLayer(onClose, !isSaving)

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
        aria-labelledby="service-modal-title"
        className="bg-card border-card-border relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col rounded-lg border shadow-lg focus:outline-none"
      >
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <h3 id="service-modal-title" className="text-text-primary text-lg font-semibold">
            {t('title')}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label={t('close')}
            disabled={isSaving}
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        {/* The form focuses its first field when it opens */}
        <ServiceForm
          service={service}
          boardId={boardId}
          variant="modal"
          onSaved={onSaved}
          onCancel={onClose}
          onSavingChange={setIsSaving}
        />
      </div>
    </div>
  )
}
