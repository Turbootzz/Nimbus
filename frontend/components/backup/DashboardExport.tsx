'use client'

import { useState } from 'react'
import { ArrowDownTrayIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import type { DashboardDocument } from '@/types'

// Hands the document to the browser as a file download
function downloadDocument(doc: DashboardDocument) {
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(doc, null, 2)], { type: 'application/json' })
  )
  const link = document.createElement('a')
  link.href = url
  link.download = `nimbus-dashboard-${new Date().toISOString().slice(0, 10)}.json`
  link.click()
  URL.revokeObjectURL(url)
}

export default function DashboardExport() {
  const [exporting, setExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [omitted, setOmitted] = useState(0)

  const handleExport = async () => {
    setExporting(true)
    setError(null)
    setOmitted(0)
    const response = await api.exportDashboard()
    setExporting(false)
    if (!response.data) {
      setError(response.error?.message ?? 'Failed to export the dashboard')
      return
    }
    downloadDocument(response.data)
    setOmitted(response.data.omitted_images ?? 0)
  }

  return (
    <section className="glass-card border-card-border mb-8 rounded-lg border p-6">
      <h2 className="text-text-primary mb-1 text-lg font-semibold">Export</h2>
      <p className="text-text-secondary mb-4 text-sm">
        Download your groups, services, widgets, integrations and preferences as one JSON file, with
        uploaded icons and wallpaper inside. Credentials and other secrets are never included.
      </p>
      <button
        type="button"
        onClick={handleExport}
        disabled={exporting}
        className="bg-primary hover:bg-primary/90 focus-visible:ring-primary inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
      >
        <ArrowDownTrayIcon className="h-4 w-4" aria-hidden="true" />
        {exporting ? 'Exporting…' : 'Export dashboard'}
      </button>
      <div aria-live="polite">
        {error && <p className="text-error mt-3 text-sm">{error}</p>}
        {omitted > 0 && (
          <p className="text-warning mt-3 text-sm">
            {omitted} uploaded {omitted === 1 ? 'image was' : 'images were'} missing or too large
            and left out of the file.
          </p>
        )}
      </div>
    </section>
  )
}
