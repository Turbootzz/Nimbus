import DashboardExport from '@/components/backup/DashboardExport'
import DashboardImport from '@/components/backup/DashboardImport'

export default function BackupPage() {
  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <div className="mb-8">
        <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">
          Export &amp; Import
        </h1>
        <p className="text-text-secondary text-sm sm:text-base">
          Back up your dashboard, move it to another Nimbus, or share its layout.
        </p>
      </div>
      <DashboardExport />
      <DashboardImport />
    </div>
  )
}
