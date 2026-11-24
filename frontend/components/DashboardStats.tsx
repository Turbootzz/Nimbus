import {
  ServerIcon,
  ClockIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
} from '@heroicons/react/24/outline'
import { useTranslations } from 'next-intl'

export interface DashboardStatsValues {
  total: number
  online: number
  offline: number
  avgResponseTime: number
}

// Stats cards at the top of the classic dashboard
export default function DashboardStats({ stats }: { stats: DashboardStatsValues }) {
  const t = useTranslations('dashboard.stats')
  return (
    <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <div className="glass-card border-card-border rounded-lg border p-6">
        <div className="flex items-center">
          <ServerIcon className="text-primary h-8 w-8" />
          <div className="ml-4">
            <p className="text-text-muted text-sm">{t('totalServices')}</p>
            <p className="text-text-primary text-2xl font-semibold">{stats.total}</p>
          </div>
        </div>
      </div>

      <div className="glass-card border-card-border rounded-lg border p-6">
        <div className="flex items-center">
          <CheckCircleIcon className="text-success h-8 w-8" />
          <div className="ml-4">
            <p className="text-text-muted text-sm">{t('online')}</p>
            <p className="text-text-primary text-2xl font-semibold">{stats.online}</p>
          </div>
        </div>
      </div>

      <div className="glass-card border-card-border rounded-lg border p-6">
        <div className="flex items-center">
          <ExclamationCircleIcon className="text-error h-8 w-8" />
          <div className="ml-4">
            <p className="text-text-muted text-sm">{t('offline')}</p>
            <p className="text-text-primary text-2xl font-semibold">{stats.offline}</p>
          </div>
        </div>
      </div>

      <div className="glass-card border-card-border rounded-lg border p-6">
        <div className="flex items-center">
          <ClockIcon className="text-info h-8 w-8" />
          <div className="ml-4">
            <p className="text-text-muted text-sm">{t('avgResponse')}</p>
            <p className="text-text-primary text-2xl font-semibold">{stats.avgResponseTime}ms</p>
          </div>
        </div>
      </div>
    </div>
  )
}
