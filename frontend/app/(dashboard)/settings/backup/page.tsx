import { getTranslations } from 'next-intl/server'
import DashboardExport from '@/components/backup/DashboardExport'
import DashboardImport from '@/components/backup/DashboardImport'
import ImportWizard from '@/components/backup/ImportWizard'

export default async function BackupPage() {
  const t = await getTranslations('backup.page')
  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <div className="wallpaper-panel mb-8">
        <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">{t('title')}</h1>
        <p className="text-text-secondary text-sm sm:text-base">{t('description')}</p>
      </div>
      <DashboardExport />
      <DashboardImport />
      <ImportWizard />
    </div>
  )
}
